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

const (
	ISO_8601    = "2006-01-02T15:04:05.000Z"
	CUSTOM_TIME = "Jan 2, 2006 at 3:04pm (MST)"
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
	startDateStr := flag.String("startDate", "", "start time in format: "+CUSTOM_TIME)
	endDateStr := flag.String("endDate", "", "end time in format: "+CUSTOM_TIME)
	flag.Parse()

	tw := TwitterHandler{}
	tw.Init(*bearerToken)
	client := &twitter.Client{
		Authorizer: tw.auth,
		Client:     http.DefaultClient,
		Host:       "https://api.twitter.com",
	}

	searchOpts := twitter.TweetRecentSearchOpts{
		MaxResults: 50,
		TweetFields: []twitter.TweetField{
			twitter.TweetFieldAuthorID,
			twitter.TweetFieldCreatedAt},
	}

	var err error
	if len(*startDateStr) > 0 {
		searchOpts.StartTime, err = time.Parse(CUSTOM_TIME, *startDateStr)
	}
	if len(*endDateStr) > 0 {
		searchOpts.EndTime, err = time.Parse(CUSTOM_TIME, *endDateStr)
	}
	if err != nil {
		panic(err)
	}

	uniqIDs := make(map[string]bool)
	catalanVoters := []string{}
	degens := []string{}
	tctx, cancel := context.WithCancel(context.TODO())
	for {
		resp, err := client.TweetRecentSearch(
			tctx,
			fmt.Sprintf("conversation_id:%s", *tweetID),
			searchOpts,
		)
		if err != nil {
			panic(err)
		}

		for _, t := range resp.Raw.Tweets {
			if _, ok := uniqIDs[t.AuthorID]; ok {
				continue
			}
			if strings.Contains(strings.ToLower(t.Text), "catalandao") {
				catalanVoters = append(catalanVoters, fmt.Sprintf(t.AuthorID))
				uniqIDs[t.AuthorID] = true
			}
			if strings.Contains(strings.ToLower(t.Text), "dataverse") {
				degens = append(degens, fmt.Sprintf(t.AuthorID))
				uniqIDs[t.AuthorID] = true
			}
		}

		searchOpts.NextToken = resp.Meta.NextToken
		// Finish if no more tweets
		if len(searchOpts.NextToken) == 0 {
			break
		}
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
