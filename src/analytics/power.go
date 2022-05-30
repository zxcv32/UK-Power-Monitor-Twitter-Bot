package analytics

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
	influxdb "zxcv32/upmtb/src/database"
)

func GetState(config *influxdb.InfluxDbConfig) string {
	state := "skip"
	lastPowerStatus := influxdb.GetLastPowerStatus(config)
	lastTweetStatus := influxdb.GetLastTweetStatus(config)
	if len(lastPowerStatus) < 1 || len(lastTweetStatus) < 1 {
		// disallow empty status
		log.Printf("disallow atleast one empty status-> %+v | %+v", lastPowerStatus,
			lastTweetStatus)
		return state
	}
	if lastPowerStatus == lastTweetStatus {
		state = "skip"
	} else {
		state = lastPowerStatus
	}
	return state
}

func getLocalTime() string {
	tz, _ := time.LoadLocation("Asia/Kolkata")
	return time.Now().In(tz).Format(time.RFC1123)
}

func PowerUpTweet(config *influxdb.InfluxDbConfig) string {
	// TODO query db for more data
	tweetEmoji := "🟢"
	content := fmt.Sprintf("Power Status: %s %s\n"+
		"Detection Timestamp: %s",
		"live", tweetEmoji, getLocalTime())
	return content
}

func PowerDownTweet(config *influxdb.InfluxDbConfig) string {
	tweetEmoji := "🔴"
	content := fmt.Sprintf("Power Status: %s %s\n"+
		"Detection Timestamp: %s\n"+
		"Outages in last 6h,12h,24h: %s",
		"down", tweetEmoji, getLocalTime(),
		influxdb.CountOutages(config, []string{"-6h", "-12h", "-24h"}))
	return content
}
