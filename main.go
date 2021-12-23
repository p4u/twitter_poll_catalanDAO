package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/g8rswimmer/go-twitter/v2"
)

type authorize struct {
	Token string
}

func (a authorize) Add(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.Token))
}

// TwitterHandler is a handler for testing that returns always true
type TwitterHandler struct {
	auth authorize
}

// Init does nothing
func (tw *TwitterHandler) Init(bearerToken string) {
	tw.auth = authorize{Token: bearerToken}
}

func main() {
	bearerToken := flag.String("token", "", "bearer APIv2 twitter token")
	tweetID := flag.String("id", "1473394217284255744", "tweet ID")
	flag.Parse()

	tw := TwitterHandler{}
	tw.Init(*bearerToken)

	client := &twitter.Client{
		Authorizer: tw.auth,
		Client:     http.DefaultClient,
		Host:       "https://api.twitter.com",
	}

	catalanVoters := []string{}
	degens := []string{}
	lastID := ""
	tctx, cancel := context.WithCancel(context.TODO())
	for {
		resp, err := client.TweetRecentSearch(
			tctx,
			fmt.Sprintf("conversation_id:%s", *tweetID),
			twitter.TweetRecentSearchOpts{
				MaxResults:  100,
				SinceID:     lastID,
				TweetFields: []twitter.TweetField{twitter.TweetFieldAuthorID, twitter.TweetFieldCreatedAt},
			},
		)
		if err != nil {
			panic(err)
		}

		for _, t := range resp.Raw.Tweets {
			if strings.Contains(strings.ToLower(t.Text), "catalan") {
				catalanVoters = append(catalanVoters, fmt.Sprintf(t.AuthorID))
			}
			if strings.Contains(strings.ToLower(t.Text), "verse") {
				degens = append(degens, fmt.Sprintf(t.AuthorID))
			}
		}

		// Finish if no more tweets
		if len(resp.Raw.Tweets) < 100 {
			break
		}
		lastID = resp.Raw.Tweets[len(resp.Raw.Tweets)-1].ID
	}
	cancel()
	fmt.Printf("catalan voters: %d | dataverse voters: %d\n", len(catalanVoters), len(degens))
	time.Sleep(time.Second)
	// Lookup for the number of followers for each user (TODO)

	tctx, cancel = context.WithCancel(context.TODO())
	for _, v := range catalanVoters {
		resp, err := client.UserFollowersLookup(
			tctx,
			v,
			twitter.UserFollowersLookupOpts{},
		)
		if err != nil {
			panic(err)
		}
		count := 0
		for range resp.Raw.Users {
			count++
		}
		fmt.Printf("User %s have %d followers", v, count)
		time.Sleep(time.Second * 3)
	}
	cancel()

	// Useful for printing th reply in JSON (nice) format
	/*
		enc, err := json.MarshalIndent(resp, "", "    ")
		if err != nil {
			log.Panic(err)
		}
		fmt.Println(string(enc))
	*/
}
