package main

import (
	"flag"
	"fmt"

	"github.com/cjlucas/unnamedcast/api"
)

var apiHost = flag.String("api-host", "localhost:80", "Host for API")

func main() {
	flag.Parse()

	apiTransport := api.API{Host: *apiHost}

	fmt.Println("Creating user")
	user, err := apiTransport.CreateUser("chris", "blah")
	if err != nil {
		panic(err)
	}

	urls := []string{
		"https://daringfireball.net/thetalkshow/rss",
		"https://feeds.feedburner.com/SModcasts?format=xml",
		"https://feeds.feedburner.com/HollywoodBabbleOnPod?format=xml",
		"http://feeds.serialpodcast.org/serialpodcast",
		"http://home.cjlucas.net:4567/feed/54fa81f1c87472e5190001ea",
		"http://home.cjlucas.net:4567/feed/54fa81f9c87472e5190001f9",
		"http://home.cjlucas.net:4567/feed/54fb7c9dc874725c74000001",
		"http://home.cjlucas.net:4567/feed/561e6369c874725575000265",
		"http://home.cjlucas.net:4567/feed/562c0cd2c87472d0ec000065",
		"http://home.cjlucas.net:4567/feed/564fde09c8747207240002c9",
		"http://home.cjlucas.net:4567/feed/54fb59cfc874725558000001",
	}

	for _, url := range urls {
		fmt.Println("Creating feed:", url)
		feed, err := apiTransport.CreateFeed(&api.Feed{URL: url})
		if err != nil {
			panic(err)
		}
		user.FeedIDs = append(user.FeedIDs, feed.ID)

		apiTransport.CreateJob(&api.Job{
			Queue:    "update-feed",
			Priority: 100,
			Payload:  map[string]string{"feed_id": feed.ID},
		})
	}

	fmt.Println("Updating user")

	if err := apiTransport.UpdateUserFeeds(user.ID, user.FeedIDs); err != nil {
		panic(err)
	}
}
