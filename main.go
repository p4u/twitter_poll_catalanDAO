package main

import (
	"context"
	"encoding/json"
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
	followersThreshold := flag.Int("threshold", 10, "number of minimum followers per user to count it as valid voter")
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
			if strings.Contains(strings.ToLower(t.Text), "catalandao") {
				catalanVoters = append(catalanVoters, fmt.Sprintf(t.AuthorID))
			}
			if strings.Contains(strings.ToLower(t.Text), "dataverse") {
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
	fmt.Printf("total catalan voters found: %d\n", len(catalanVoters))
	fmt.Printf("total dataverse voters found: %d\n", len(degens))

	catalanValid := checkValidUsers(client, catalanVoters, *followersThreshold)
	degenValid := checkValidUsers(client, degens, *followersThreshold)
	fmt.Printf("catalanDAO valid voters: %d\n", catalanValid)
	fmt.Printf("dataverse valid voters: %d\n", degenValid)

}

func checkValidUsers(client *twitter.Client, ids []string, threshold int) int {
	valid := 0

	// Lookup for the number of followers for each user (TODO)
	for i := 0; i < len(ids); i += 50 {
		j := i + 50
		if j > len(ids) {
			j = len(ids)
		}
		tctx, cancel := context.WithCancel(context.TODO())
		user, err := client.UserLookup(tctx, ids[i:j], twitter.UserLookupOpts{
			UserFields: []twitter.UserField{
				twitter.UserFieldName,
				twitter.UserFieldPublicMetrics,
			},
		})
		if err != nil {
			panic(err)
		}
		for _, u := range user.Raw.Users {
			if u.PublicMetrics.Followers >= threshold {
				valid++
			}
			//fmt.Printf("user %s have %d followers\n", u.Name, u.PublicMetrics.Followers)
		}
		cancel()
		time.Sleep(time.Second * 1)
	}
	return valid
}

// Useful for printing th reply in JSON (nice) format
func printResponse(resp interface{}) {
	enc, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Errorf("error marshaling: %v", err)
	}
	fmt.Println(string(enc))
}
