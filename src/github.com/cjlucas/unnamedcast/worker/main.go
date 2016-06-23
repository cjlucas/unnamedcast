package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cjlucas/unnamedcast/api"
	"github.com/cjlucas/unnamedcast/koda"
	"github.com/cjlucas/unnamedcast/server/db"
)

const (
	queueScrapeiTunesFeeds = "scrape-itunes-feeds"
	queueUpdateFeed        = "update-feed"
	queueUpdateUserFeeds   = "update-user-feeds"
)

const MaxAttempts = 5

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

type PersistedJob struct {
	Job *db.Job
}

func (j PersistedJob) AppendLogf(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	if j.Job == nil {
		fmt.Println(line)
		return
	}

	jobCollection.AppendLog(j.Job.ID, line)
}

func (j PersistedJob) UpdateState(state koda.JobState) {
	if j.Job == nil {
		return
	}
	var s string
	switch state {
	case koda.Working:
		s = "working"
	case koda.Finished:
		s = "finished"
	case koda.Dead:
		s = "dead"
	default:
		panic(fmt.Errorf("unknown state: %d", state))
	}
	jobCollection.UpdateState(j.Job.ID, s)
}

var jobCollection *db.JobCollection

func persistedJobByID(kodaID int) PersistedJob {
	var persistedJob PersistedJob

	if jobCollection == nil {
		return persistedJob
	}

	var job db.Job
	if err := jobCollection.FindByKodaID(kodaID).One(&job); err != nil {
		fmt.Println("Error fetching persisted job, log will not persist", err)
	} else {
		persistedJob.Job = &job
	}

	return persistedJob
}

func runQueueWorker(wg *sync.WaitGroup, q *koda.Queue, w Worker) {
	defer wg.Done()

	for {
		j, err := q.Wait()
		if err != nil {
			fmt.Println("Error occured while waiting for job:", err)
			continue
		}

		fmt.Printf("Job %d: Dequeued\n", j.ID)
		persistedJob := persistedJobByID(j.ID)

		persistedJob.UpdateState(koda.Working)
		if err := w.Work(q, j); err != nil {
			persistedJob.AppendLogf("Job %d: Failed with error: %s", j.ID, err)
			if j.NumAttempts == MaxAttempts {
				persistedJob.AppendLogf("Job %d: Max attempts reached, killing job", j.ID)
				persistedJob.UpdateState(koda.Dead)
				j.Kill()
			} else {
				persistedJob.AppendLogf("Job %d: Failed on attempt %d, will retry\n", j.ID, j.NumAttempts)
				j.Retry(5 * time.Minute)
			}
			continue
		}
		fmt.Printf("Job %d: Done\n", j.ID)
		persistedJob.AppendLogf("Job completed successfully")
		persistedJob.UpdateState(koda.Finished)
		j.Finish()
	}
}

func main() {
	var queueList queueOptList
	flag.Var(&queueList, "q", "queueName[:numWorkers]")
	flag.Parse()

	koda.Configure(&koda.Options{
		URL: os.Getenv("REDIS_URL"),
	})

	koda.Submit(queueScrapeiTunesFeeds, 0, nil)

	// koda.Submit(queueUpdateFeed, 0, &UpdateFeedPayload{
	// 	FeedID: "56d5c158c87472028649f39a",
	// })
	//
	// koda.Submit(queueUpdateUserFeeds, 0, nil)
	var wg sync.WaitGroup

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		panic("DB_URL not specified")
	}
	dbConn, err := db.New(db.Config{URL: dbURL})
	if err == nil {
		fmt.Println("setting job collection")
		jobCollection = &dbConn.Jobs
	} else {
		fmt.Println("error connecting to db", err)
	}

	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		panic("API_URL not specified")
	}

	url, err := url.Parse(apiURL)
	if err != nil {
		panic(fmt.Sprintf("Invalid API_URL given: %s", apiURL))
	}
	api := api.API{Host: url.Host}

	handlers := map[string]Worker{
		queueScrapeiTunesFeeds: &ScrapeiTunesFeeds{API: api},
		queueUpdateFeed:        &UpdateFeedWorker{API: api},
		queueUpdateUserFeeds:   &UpdateUserFeedsWorker{API: api},
	}

	fmt.Println(apiURL)
	fmt.Printf("%+v\n", handlers[queueScrapeiTunesFeeds])

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
