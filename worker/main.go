package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/cjlucas/unnamedcast/koda"
)

const (
	queueScrapeiTunesFeeds = "scrape-itunes-feeds"
	queueUpdateFeed        = "update-feed"
)

type queueOpt struct {
	Name       string
	NumWorkers int
}

type queueOptList []queueOpt

func (l *queueOptList) String() string {
	return fmt.Sprintf("%#v", l)
}

func (l *queueOptList) Set(s string) error {
	// format: name[:num_workers]
	split := strings.Split(s, ":")
	q := queueOpt{
		Name:       split[0],
		NumWorkers: 1,
	}

	if len(split) == 2 {
		i, err := strconv.Atoi(split[1])
		if err != nil {
			return err
		}
		q.NumWorkers = i
	}

	*l = append(*l, q)
	return nil
}

func runQueueWorker(wg *sync.WaitGroup, q *koda.Queue, w Worker) {
	defer wg.Done()

	for {
		j, err := q.Wait()
		if err != nil {
			fmt.Println("Error occured while waiting for job:", err)
			continue
		}

		if err := w.Work(q, j); err != nil {
			fmt.Printf("Job %d: Failed with error: %s\n", j.ID, err)
			continue
		}
		fmt.Printf("Job #%d: Done\n", j.ID)
		j.Done()
	}
}

func main() {
	var queueList queueOptList
	flag.Var(&queueList, "q", "Usage goes here")
	flag.Parse()

	koda.Submit(queueScrapeiTunesFeeds, 0, nil)

	var wg sync.WaitGroup

	handlers := map[string]Worker{
		queueScrapeiTunesFeeds: &ScrapeiTunesFeeds{},
		queueUpdateFeed:        &UpdateFeedWorker{},
	}

	for _, opt := range queueList {
		for i := 0; i < opt.NumWorkers; i++ {
			wg.Add(1)
			fmt.Println(opt.Name, i)
			worker := handlers[opt.Name]
			if worker == nil {
				panic(fmt.Sprintf("No worker found for queue: %s", opt.Name))
			}
			go runQueueWorker(&wg, koda.GetQueue(opt.Name), worker)
		}
	}

	wg.Wait()
}
