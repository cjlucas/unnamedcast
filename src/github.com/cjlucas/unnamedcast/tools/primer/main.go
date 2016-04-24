package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/cjlucas/unnamedcast/api"
	"github.com/cjlucas/unnamedcast/koda"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: primer <hostname>")
	}

	host := os.Args[1]

	koda.Configure(&koda.Options{
		URL: fmt.Sprintf("redis://%s:6379", host),
	})

	baseURL, err := url.Parse(fmt.Sprintf("http://%s", host))
	if err != nil {
		panic(err)
	}

	apiTransport := api.API{BaseURL: baseURL}

	fmt.Println("Creating user")
	user, err := apiTransport.CreateUser("chris", "blah")
	if err != nil {
		panic(err)
	}

	urls := []string{
		"https://daringfireball.net/thetalkshow/rss",
	}

	for _, url := range urls {
		fmt.Println("Creating feed:", url)
		feed, err := apiTransport.CreateFeed(&api.Feed{URL: url})
		if err != nil {
			panic(err)
		}
		user.FeedIDs = append(user.FeedIDs, feed.ID)

		koda.Submit("update-feed", 100, map[string]string{
			"feed_id": feed.ID,
		})
	}

	fmt.Println("Updating user")

	if err := apiTransport.UpdateUserFeeds(user.ID, user.FeedIDs); err != nil {
		panic(err)
	}
}
