package main

import (
	"fmt"
	"os"

	"github.com/cjlucas/koda-go"
	"github.com/cjlucas/unnamedcast/api"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: api <url>")
		os.Exit(1)
	}

	koda.Configure(&koda.Options{
		URL: "redis://192.168.1.21:6379",
	})

	apiTransport := api.API{Host: "192.168.1.21"}

	url := os.Args[1]
	feed, err := apiTransport.CreateFeed(&api.Feed{
		URL: url,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(feed)
	apiTransport.CreateJob(&api.Job{
		Queue:    "update-feed",
		Priority: 100,
		Payload: map[string]string{
			"feed_id": feed.ID,
		},
	})
}
