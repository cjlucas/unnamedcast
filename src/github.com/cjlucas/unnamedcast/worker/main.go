package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/cjlucas/koda-go"
	"github.com/cjlucas/unnamedcast/api"
	"github.com/cjlucas/unnamedcast/db"
)

const (
	queueScrapeiTunesFeeds = "scrape-itunes-feeds"
	queueUpdateFeed        = "update-feed"
	queueUpdateUserFeeds   = "update-user-feeds"
)

type Job struct {
	KodaJob    *koda.Job
	collection *db.JobCollection
	dbID       db.ID
}

func (j *Job) Logf(format string, args ...interface{}) {
	if j.dbID.Valid() {
		j.collection.AppendLog(j.dbID, fmt.Sprintf(format, args...))
	} else {
		fmt.Printf(format, args...)
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

func main() {
	var queueList queueOptList
	flag.Var(&queueList, "q", "queueName[:numWorkers]")
	flag.Parse()

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

	q := koda.Queue{
		Name: queueUpdateUserFeeds,
	}

	updateUserFeedsWorker := &UpdateUserFeedsWorker{
		API:  api,
		Koda: kodaClient,
	}
	kodaClient.Register(q, func(job *koda.Job) error {
		j := &Job{
			KodaJob:    job,
			collection: &dbConn.Jobs,
		}

		var dbJob db.Job
		if err := dbConn.Jobs.FindByKodaID(job.ID).One(&dbJob); err != nil {
			fmt.Println("Could not fetch job")
		} else {
			j.dbID = dbJob.ID
		}
		j.Logf("Starting job")
		dbConn.Jobs.UpdateState(dbJob.ID, "working")

		if err := updateUserFeedsWorker.Work(j); err != nil {
			j.Logf("Failed with error: %s", err)
		}

		dbConn.Jobs.UpdateState(dbJob.ID, "finished")
		return err
	})
}
