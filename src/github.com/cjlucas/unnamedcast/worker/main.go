package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cjlucas/koda-go"
	"github.com/cjlucas/unnamedcast/api"
	"github.com/cjlucas/unnamedcast/db"
)

const (
	queueScrapeiTunesFeeds = "scrape-itunes-feeds"
	queueUpdateFeed        = "update-feed"
	queueUpdateUserFeeds   = "update-user-feeds"
)

type Worker interface {
	Work(job *Job) error
}

type Job struct {
	KodaJob    *koda.Job
	collection *db.JobCollection
	dbID       db.ID
}

func (j *Job) Logf(format string, args ...interface{}) {
	if j.dbID.Valid() {
		j.collection.AppendLog(j.dbID, fmt.Sprintf(format, args...))
	} else {
		fmt.Printf(format+"\n", args...)
	}
}

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

func wrapHandler(dbConn *db.DB, f func(*Job) error) koda.HandlerFunc {
	return func(j *koda.Job) error {
		job := &Job{
			KodaJob:    j,
			collection: &dbConn.Jobs,
		}

		var dbJob db.Job
		if err := dbConn.Jobs.FindByKodaID(j.ID).One(&dbJob); err != nil {
			fmt.Println("Could not fetch job")
		} else {
			job.dbID = dbJob.ID
			fmt.Println("fetched job with id", job.dbID)
		}
		job.Logf("Starting job")
		dbConn.Jobs.UpdateState(dbJob.ID, "working")

		err := f(job)
		if err != nil {
			job.Logf("Failed with error: %s", err)
		} else {
			job.Logf("Job completed successfully")
		}

		dbConn.Jobs.UpdateState(dbJob.ID, "finished")
		return err
	}
}

func main() {
	var queueList queueOptList
	flag.Var(&queueList, "q", "queueName[:numWorkers]")
	flag.Parse()

	if len(queueList) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	kodaClient := koda.NewClient(&koda.Options{
		URL: os.Getenv("REDIS_URL"),
	})

	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		panic("API_URL not specified")
	}

	url, err := url.Parse(apiURL)
	if err != nil {
		panic(fmt.Sprintf("Invalid API_URL given: %s", apiURL))
	}
	api := api.API{Host: url.Host}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		panic("DB_URL not specified")
	}
	dbConn, err := db.New(db.Config{URL: dbURL})
	if err != nil {
		panic(fmt.Errorf("Could not connect to db: %s", err))
	}

	workers := make(map[string]Worker)
	workers[queueUpdateUserFeeds] = &UpdateUserFeedsWorker{
		API:  api,
		Koda: kodaClient,
	}
	workers[queueUpdateFeed] = &UpdateFeedWorker{
		API: api,
	}
	workers[queueScrapeiTunesFeeds] = &ScrapeiTunesFeeds{
		API:  api,
		Koda: kodaClient,
	}

	for _, opt := range queueList {
		worker, ok := workers[opt.Name]
		if !ok {
			fmt.Fprintf(os.Stderr, "%s is not a valid queue\n", opt.Name)
			os.Exit(1)
		}

		q := koda.Queue{
			Name:          opt.Name,
			NumWorkers:    opt.NumWorkers,
			RetryInterval: 5 * time.Minute,
			MaxAttempts:   5,
		}

		kodaClient.Register(q, wrapHandler(dbConn, worker.Work))
	}

	kodaClient.WorkForever()
}
