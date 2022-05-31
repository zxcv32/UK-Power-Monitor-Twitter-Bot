package main

import (
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
	"zxcv32/upmtb/src/analytics"
	influxdb "zxcv32/upmtb/src/database"
	"zxcv32/upmtb/src/twitter"
)

// main tweets the change in main power status
func main() {
	if err := godotenv.Load(); err != nil {
		log.Warningln("No .env file found")
	}
	twitterCredentials := twitter.Credentials{
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

	for range time.Tick(time.Second) {
		state := analytics.GetState(&influxDbConfig)
		var content string
		switch state {
		case "live":
			content = analytics.PowerUpTweet(&influxDbConfig)
		case "down":
			content = analytics.PowerDownTweet(&influxDbConfig)
		case "skip": // Do nothing, duh!
			continue
		default:
			log.Errorf("Unknwon state: %s", state)
		}
		tweetDbRecord := twitter.Tweet(&twitterCredentials, content)
		tweetDbRecord.Status = state
		log.Infof("Tweet state: %s", state)
		influxdb.WriteToDb(&influxDbConfig, tweetDbRecord)
	}
}
