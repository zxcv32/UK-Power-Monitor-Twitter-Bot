package main

import (
	"context"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

// TwitterCredentials for oauth1.0a authentication
type TwitterCredentials struct {
	ApiKey       string
	ApiSecret    string
	AccessToken  string
	AccessSecret string
}

// InfluxDbConfig for reading InfluxDB data
type InfluxDbConfig struct {
	BaseUrl     string
	BucketPower string
	BucketTweet string
	Org         string
	Token       string
}

type TweetDbRecord struct {
	Status       string
	TweetContent string
	TweetId      int64
	Error        string
}

// buildTwitterClient builds a Twitter client and validate credentials
func buildTwitterClient(creds *TwitterCredentials) (*twitter.Client, error) {

	config := oauth1.NewConfig(creds.ApiKey, creds.ApiSecret)
	token := oauth1.NewToken(creds.AccessToken, creds.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	// Build Twitter client
	client := twitter.NewClient(httpClient)

	verifyParams := &twitter.AccountVerifyParams{
		IncludeEntities: twitter.Bool(false),
		SkipStatus:      twitter.Bool(true),
		IncludeEmail:    twitter.Bool(true),
	}

	user, _, err := client.Accounts.VerifyCredentials(verifyParams)
	if err != nil {
		log.Debugln(err)
		return nil, err
	}
	log.Debugf("Auth user: %s (%s)\n", user.Name, user.ScreenName)
	return client, nil
}

func buildInfluxDbClient(config *InfluxDbConfig) (influxdb2.Client, error) {
	client := influxdb2.NewClient(config.BaseUrl, config.Token)
	pong, err := client.Ping(context.Background())
	if !pong || err != nil {
		return nil, err
	}
	return client, nil
}

func queryInfluxDb(config *InfluxDbConfig, query string) (*api.QueryTableResult, error) {
	client, err := buildInfluxDbClient(config)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	// Get query client
	queryAPI := client.QueryAPI(config.Org)
	result, err := queryAPI.Query(context.Background(), query)
	if err == nil {
		if result.Err() != nil {
			log.Errorf("Query error: %s", result.Err().Error())
		}
	} else {
		log.Errorf(err.Error())
		return nil, err
	}
	client.Close()
	return result, nil
}

// tweet Tweet stat
func tweet(client *twitter.Client, tweetContent string) (*twitter.Tweet, error) {
	tweet, resp, err := client.Statuses.Update(tweetContent, nil)
	if err != nil {
		return nil, err
		log.Debugln(err)
	}
	log.Debugf("Status: %s\n", resp.Status)
	log.Debugln("Tweet ID: %d\n", tweet.ID)
	return tweet, nil
}

// writeTweetToDb Write tweet data to InfluxDB
func writeTweetToDb(config *InfluxDbConfig, data TweetDbRecord) {
	client, err := buildInfluxDbClient(config)
	if err != nil {
		log.Errorln(err)
		return
	}
	record := fmt.Sprintf(`%s,tweet=monitor id=%d,status="%s",content="%s",error="%s"`, config.BucketTweet, data.TweetId, data.Status, data.TweetContent, data.Error)
	writeAPI := client.WriteAPI(config.Org, config.BucketTweet)
	writeAPI.WriteRecord(record)
	writeAPI.Flush()
	defer client.Close()
}

func getLastPowerStatus(config *InfluxDbConfig) string {
	// Range start is -2s because there may be few milliseconds delay when the sensor data is
	// written and this bot reads last status from db.
	response, err := queryInfluxDb(config, `
			from(bucket: "`+config.BucketPower+`")
			  |> range(start: -2s, stop: now())
			  |> filter(fn: (r) => r["_measurement"] == "lab")
			  |> filter(fn: (r) => r["_field"] == "status")
			  |> yield(name: "last")
		`)
	if err != nil {
		log.Errorln(err)
	}
	var result []string
	for response.Next() {
		values := response.Record().Values()
		value, exists := values["_value"]
		if exists {
			result = append(result, fmt.Sprintf(fmt.Sprintf("%s", value)))
		}
	}
	if len(result) < 1 {
		return ""
	}
	return result[0]
}

func getLastTweetStatus(config *InfluxDbConfig) string {
	response, err := queryInfluxDb(config, `
			from(bucket: "`+config.BucketTweet+`")
			  |> range(start: -1d)
			  |> filter(fn: (r) => r["_measurement"] == "twitter")
			  |> filter(fn: (r) => r["_field"] == "status")
			  |> filter(fn: (r) => r["tweet"] == "monitor")
		      |> last()
		`)
	if err != nil {
		log.Errorln(err)
	}
	var result []string
	for response.Next() {
		values := response.Record().Values()
		value, exists := values["_value"]
		if exists {
			result = append(result, fmt.Sprintf(fmt.Sprintf("%s", value)))
		}
	}
	if len(result) < 1 {
		return ""
	}
	return result[0]
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Warningln("No .env file found")
	}
	twitterCredentials := TwitterCredentials{
		ApiKey:       os.Getenv("API_KEY"),
		ApiSecret:    os.Getenv("API_SECRET"),
		AccessToken:  os.Getenv("ACCESS_TOKEN"),
		AccessSecret: os.Getenv("ACCESS_SECRET"),
	}

	client, err := buildTwitterClient(&twitterCredentials)
	if err != nil {
		log.Fatalln(err)
	}

	influxDbConfig := InfluxDbConfig{
		BaseUrl:     os.Getenv("INFLUXDB_BASE_URL"),
		BucketPower: os.Getenv("INFLUXDB_BUCKET_POWER"),
		BucketTweet: os.Getenv("INFLUXDB_BUCKET_TWEET"),
		Org:         os.Getenv("INFLUXDB_ORG"),
		Token:       os.Getenv("INFLUXDB_TOKEN"),
	}

	for range time.Tick(1 * time.Second) {
		lastPowerStatus := getLastPowerStatus(&influxDbConfig)
		lastTweetStatus := getLastTweetStatus(&influxDbConfig)
		if lastTweetStatus != lastPowerStatus {
			log.Printf("Status change detected -> %v | %v\n", lastPowerStatus, lastTweetStatus)
			if len(lastPowerStatus) < 1 || len(lastTweetStatus) < 1 {
				// disallow empty status
				log.Printf("disallow atleast one empty status-> %+v | %+v", lastPowerStatus, lastTweetStatus)
				continue
			}
			tz, _ := time.LoadLocation("Asia/Kolkata")
			tweetContent := fmt.Sprintf("Power Status: %s\nDetection Timestamp: %s", lastPowerStatus, time.Now().In(tz).Format(time.RFC1123))
			tw, err := tweet(client, tweetContent)
			errorString := ""
			var twId int64 = -1 // Not tweet was made
			if err != nil {
				errorString = err.Error()
				log.Errorln(errorString)
			} else {
				twId = tw.ID
			}
			tweetDbRecord := TweetDbRecord{
				Status:       lastPowerStatus,
				TweetContent: tweetContent,
				TweetId:      twId,
				Error:        errorString,
			}
			writeTweetToDb(&influxDbConfig, tweetDbRecord)
		}
	}
}
