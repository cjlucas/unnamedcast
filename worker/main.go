package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/cjlucas/unnamedcast/yajq"
)

const (
	queueScrapeiTunesGenreList = "scrape-itunes-genre-list"
	queueScrapeiTunesGenre     = "scrape-itunes-genre"
	queueScrapeiTunesFeedList  = "scrape-itunes-feed-list"
	queueUpdateFeed            = "update-feed"
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

func runQueueWorker(wg *sync.WaitGroup, q *yajq.Queue, w Worker) {
	defer wg.Done()

	for {
		j, err := q.Wait()
		if err != nil {
			fmt.Println("Error occured while waiting for job:", err)
			continue
		}
		if w != nil {
			if err := w.Work(q, j); err != nil {
				fmt.Printf("Job %d: Failed with error: %s\n", j.ID, err)
			} else {
				fmt.Printf("Job #%d: Done\n", j.ID)
				q.Done(j)
			}
		}
	}
}

func main() {
	var queueList queueOptList
	flag.Var(&queueList, "q", "Usage goes here")
	flag.Parse()

	yajq.Submit(queueScrapeiTunesGenreList, 0, nil)

	var wg sync.WaitGroup

	handlers := map[string]Worker{
		queueScrapeiTunesGenreList: &ScrapeiTunesGenreListWorker{},
		queueScrapeiTunesGenre:     &ScrapeiTunesGenreWorker{},
		queueScrapeiTunesFeedList:  &ScrapeiTunesFeedListWorker{},
		queueUpdateFeed:            &UpdateFeedWorker{},
	}

	for _, opt := range queueList {
		for i := 0; i < opt.NumWorkers; i++ {
			wg.Add(1)
			fmt.Println(opt.Name, i)
			go runQueueWorker(&wg, yajq.GetQueue(opt.Name), handlers[opt.Name])
		}
	}

	wg.Wait()
}
