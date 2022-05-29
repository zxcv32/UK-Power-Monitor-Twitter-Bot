package twitter

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	log "github.com/sirupsen/logrus"
	influxdb "zxcv32/upmtb/src/database"
)

// TwitterCredentials for oauth1.0a authentication
type TwitterCredentials struct {
	ApiKey       string
	ApiSecret    string
	AccessToken  string
	AccessSecret string
}

// build a Twitter client and validate credentials
func build(creds *TwitterCredentials) (*twitter.Client, error) {

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

// Tweet stat
func Tweet(twitterCredentials *TwitterCredentials, tweetContent string) influxdb.TweetDbRecord {
	client, err := build(twitterCredentials)
	if err != nil {
		log.Fatalln(err)
	}

	var twId int64 = -1 // Not tweet was made

	errorString := ""
	tweet, resp, err := client.Statuses.Update(tweetContent, nil)
	if err != nil {
		errorString = err.Error()
		log.Errorln(errorString)
	} else {
		twId = tweet.ID
	}
	log.Debugf("Status: %s\n", resp.Status)
	log.Debugln("Tweet ID: %d\n", tweet.ID)

	tweetDbRecord := influxdb.TweetDbRecord{
		TweetContent: tweetContent,
		TweetId:      twId,
		Error:        errorString,
	}

	return tweetDbRecord
}
