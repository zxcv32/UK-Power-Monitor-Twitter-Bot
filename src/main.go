package main

import (
	"fmt"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
	influxdb "zxcv32/upmtb/src/database"
	"zxcv32/upmtb/src/twitter"
)

// main tweets the change in main power status
func main() {
	if err := godotenv.Load(); err != nil {
		log.Warningln("No .env file found")
	}
	twitterCredentials := twitter.TwitterCredentials{
		ApiKey:       os.Getenv("API_KEY"),
		ApiSecret:    os.Getenv("API_SECRET"),
		AccessToken:  os.Getenv("ACCESS_TOKEN"),
		AccessSecret: os.Getenv("ACCESS_SECRET"),
	}

	influxDbConfig := influxdb.InfluxDbConfig{
		BaseUrl:     os.Getenv("INFLUXDB_BASE_URL"),
		BucketPower: os.Getenv("INFLUXDB_BUCKET_POWER"),
		BucketTweet: os.Getenv("INFLUXDB_BUCKET_TWEET"),
		Org:         os.Getenv("INFLUXDB_ORG"),
		Token:       os.Getenv("INFLUXDB_TOKEN"),
	}

	for range time.Tick(1 * time.Second) {
		lastPowerStatus := influxdb.GetLastPowerStatus(&influxDbConfig)
		lastTweetStatus := influxdb.GetLastTweetStatus(&influxDbConfig)
		if lastTweetStatus != lastPowerStatus {
			log.Printf("Status change detected -> %v | %v\n", lastPowerStatus, lastTweetStatus)
			if len(lastPowerStatus) < 1 || len(lastTweetStatus) < 1 {
				// disallow empty status
				log.Printf("disallow atleast one empty status-> %+v | %+v", lastPowerStatus,
					lastTweetStatus)
				continue
			}
			tz, _ := time.LoadLocation("Asia/Kolkata")
			tweetEmoji := "🔴"
			if lastPowerStatus == "live" {
				tweetEmoji = "🟢"
			}
			tweetContent := fmt.Sprintf("Power Status: %s %s\nDetection Timestamp: %s",
				lastPowerStatus, tweetEmoji, time.Now().In(tz).Format(time.RFC1123))
			tw, err := twitter.Tweet(&twitterCredentials, tweetContent)
			errorString := ""
			var twId int64 = -1 // Not tweet was made
			if err != nil {
				errorString = err.Error()
				log.Errorln(errorString)
			} else {
				twId = tw.ID
			}
			tweetDbRecord := influxdb.TweetDbRecord{
				Status:       lastPowerStatus,
				TweetContent: tweetContent,
				TweetId:      twId,
				Error:        errorString,
			}
			influxdb.WriteToDb(&influxDbConfig, tweetDbRecord)
		}
	}
}
