package database

import (
	"context"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

// InfluxDbConfig for reading InfluxDB data
type InfluxDbConfig struct {
	BaseUrl     string
	BucketPower string
	BucketTweet string
	Org         string
	Token       string
}

// TweetDbRecord records what was tweeted
type TweetDbRecord struct {
	Status       string
	TweetContent string
	TweetId      string
	Error        string
}

func build(config *InfluxDbConfig) (influxdb2.Client, error) {
	client := influxdb2.NewClient(config.BaseUrl, config.Token)
	pong, err := client.Ping(context.Background())
	if !pong || err != nil {
		return nil, err
	}
	return client, nil
}

func query(config *InfluxDbConfig, query string) (*api.QueryTableResult, error) {
	client, err := build(config)
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

// WriteToDb Write tweet data to InfluxDB
func WriteToDb(config *InfluxDbConfig, data TweetDbRecord) {
	client, err := build(config)
	if err != nil {
		log.Errorln(err)
		return
	}
	record := fmt.Sprintf(`%s,tweet=monitor id=%s,status="%s",content="%s",error="%s"`, config.BucketTweet, data.TweetId, data.Status, data.TweetContent, data.Error)
	writeAPI := client.WriteAPI(config.Org, config.BucketTweet)
	writeAPI.WriteRecord(record)
	writeAPI.Flush()
	defer client.Close()
}

func GetLastPowerStatus(config *InfluxDbConfig) string {
	// Range start is -2s because there may be few milliseconds delay when the sensor data is
	// written and this bot reads last status from db.
	return getQueryResult(config, `
			from(bucket: "`+config.BucketPower+`")
			  |> range(start: -2s, stop: now())
			  |> filter(fn: (r) => r["_measurement"] == "lab")
			  |> filter(fn: (r) => r["_field"] == "status")
			  |> yield(name: "last")
		`)
}

func GetLastTweetStatus(config *InfluxDbConfig) string {
	return getQueryResult(config, `
			from(bucket: "`+config.BucketTweet+`")
			  |> range(start: -1d)
			  |> filter(fn: (r) => r["_measurement"] == "twitter")
			  |> filter(fn: (r) => r["_field"] == "status")
			  |> filter(fn: (r) => r["tweet"] == "monitor")
		      |> last()
		`)
}

func CountOutages(config *InfluxDbConfig, intervals []string) string {
	var results []string
	for _, interval := range intervals {
		count := getQueryResult(config, `
			from(bucket: "`+config.BucketTweet+`")
			  |> range(start: `+interval+`)
			  |> filter(fn: (r) => r["_measurement"] == "twitter")
			  |> filter(fn: (r) => r["_field"] == "status")
			  |> filter(fn: (r) => r["_value"] == "down")
			  |> count()
			  |> yield(name: "count")
		`)
		value := "1"
		if len(count) > 0 {
			c, err := strconv.Atoi(count)
			if nil != err {
				log.Errorln(err.Error())
			} else {
				c++ // count this outage too
				value = strconv.Itoa(c)
			}
		}
		results = append(results, value)
	}
	return strings.Join(results, ",")
}

func CalculateOutagesDuration(config *InfluxDbConfig) string {
	response, err := query(config, `
		from(bucket: "`+config.BucketTweet+`")
  		  |> range(start: 0, stop: now())
  		  |> filter(fn: (r) => r["_measurement"] == "twitter")
		  |> tail(n: 1)
		`)
	if err != nil {
		log.Errorln(err)
	}
	var result []string
	for response.Next() {
		if len(result) > 0 {
			break
		}
		values := response.Record().Values()
		value, exists := values["_time"]
		if exists {
			fmtTime := fmt.Sprint(value)
			recordTime, _ := time.Parse("2006-01-02 15:04:05.999999999 +0000 UTC", fmtTime)
			duration := time.Now().Sub(recordTime)
			result = append(result, duration.Truncate(time.Second).String())
		}
	}
	if len(result) < 1 {
		return ""
	}
	return result[0]
}

func getQueryResult(config *InfluxDbConfig, queryStr string) string {
	response, err := query(config, queryStr)
	if err != nil {
		log.Errorln(err)
	}
	var result []string
	for response.Next() {
		values := response.Record().Values()
		value, exists := values["_value"]
		if exists {
			result = append(result, fmt.Sprint(value))
		}
	}
	if len(result) < 1 {
		return ""
	}
	return result[0]
}
