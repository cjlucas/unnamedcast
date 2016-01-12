package yajq

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type Job struct {
	ID             int
	Delayed        bool
	DelayedUntil   time.Time
	Done           bool // TODO: done should not be public
	CreationTime   time.Time
	CompletionTime time.Time
	Priority       int
	Payload        interface{}
	Progress       int
	Logs           []string
	queue          *Queue
}

func (j *Job) key() string {
	return j.queue.buildKey("jobs", strconv.Itoa(j.ID))
}

func (j *Job) logKey() string {
	return j.queue.buildKey("logs", strconv.Itoa(j.ID))
}

func (j *Job) UpdateProgress(progress int) error {
	j.Progress = progress

	conn := j.queue.getConn()
	defer j.queue.putConn(conn)

	_, err := conn.HSet(j.key(), "progress", strconv.Itoa(progress))
	return err
}

func (j *Job) Logf(f string, vals ...interface{}) error {
	s := fmt.Sprintf(f, vals...)

	conn := j.queue.getConn()
	defer j.queue.putConn(conn)

	_, err := conn.RPush(j.logKey(), s)
	return err
}

type jobUnmarshaller struct {
	Err error
}

func (u *jobUnmarshaller) atoi(s string) int {
	if u.Err != nil || s == "" {
		return 0
	}

	val, err := strconv.Atoi(s)
	if err != nil {
		u.Err = err
	}

	return val
}

func (u *jobUnmarshaller) atob(s string) bool {
	if u.Err != nil || s == "" {
		return false
	}

	val, err := strconv.ParseBool(s)
	if err != nil {
		u.Err = err
	}

	return val
}

func (u *jobUnmarshaller) parseJSON(s string) interface{} {
	if u.Err != nil || s == "" {
		return nil
	}

	var val interface{}
	if err := json.Unmarshal([]byte(s), &val); err != nil {
		u.Err = err
	}

	return val
}

func (u *jobUnmarshaller) atot(s string) time.Time {
	secs := u.atoi(s)
	if u.Err != nil {
		return time.Time{}
	}

	return time.Unix(int64(secs), 0).UTC()
}

func unmarshalJob(c Conn, key string) (*Job, error) {
	propMap := make(map[string]string)
	jobProps := []string{
		"id",
		"delayed",
		"delayed_until",
		"done",
		"creation_time",
		"priority",
		"payload",
	}

	for _, prop := range jobProps {
		s, err := c.HGet(key, prop)
		if err != nil {
			return nil, err
		}

		propMap[prop] = s
	}

	u := jobUnmarshaller{}
	job := Job{
		ID:             u.atoi(propMap["id"]),
		Delayed:        u.atob(propMap["delayed"]),
		DelayedUntil:   u.atot(propMap["delayed_until"]),
		Done:           u.atob(propMap["done"]),
		CreationTime:   u.atot(propMap["creation_time"]),
		CompletionTime: u.atot(propMap["completion_time"]),
		Priority:       u.atoi(propMap["priority"]),
		Payload:        u.parseJSON(propMap["payload"]),
	}

	if u.Err != nil {
		panic(fmt.Sprintf("error during unmarshalling: %s", u.Err))
	}

	return &job, nil
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
