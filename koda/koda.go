package koda

import (
	"strconv"
	"time"
)

const minPriority = 0
const maxPriority = 100

type Queue struct {
	name   string
	client *Client
}

func timeAsFloat(t time.Time) float64 {
	// time.Second is the number of nanoseconds in one second
	// return float64(t.Unix())
	return float64(t.UTC().UnixNano()) / float64(time.Second)
}

func (q *Queue) persistNewJob(j *Job, c Conn) error {
	id, err := q.incrJobID(c)
	if err != nil {
		return err
	}

	j.ID = id
	j.CreationTime = time.Now().UTC()

	return q.persistJob(j, c)
}

func (q *Queue) key(priority int) string {
	return q.client.buildKey("queue", q.name, strconv.Itoa(priority))
}

func (q *Queue) delayedKey() string {
	return q.client.buildKey("delayed_queue", q.name)
}

func (q *Queue) jobKey(j *Job) string {
	return q.client.buildKey("jobs", strconv.Itoa(j.ID))
}

func (q *Queue) logKey(j *Job) string {
	return q.client.buildKey("logs", strconv.Itoa(j.ID))
}

func (q *Queue) incrJobID(c Conn) (int, error) {
	return c.Incr(q.client.buildKey("cur_job_id"))
}

func (q *Queue) persistJob(j *Job, c Conn, fields ...string) error {
	jobKey := q.jobKey(j)
	hash := j.asHash()

	if len(fields) == 0 {
		for k := range hash {
			fields = append(fields, k)
		}
	}

	// TODO: Should probably do some cleanup if an error was hit
	for _, field := range fields {
		if _, err := c.HSet(jobKey, field, hash[field]); err != nil {
			return err
		}
	}

	return nil
}

func (q *Queue) UpdateProgress(j *Job, progress int) error {
	conn := q.client.getConn()
	defer q.client.putConn(conn)

	j.Progress = progress
	return q.persistJob(j, conn, "progress")
}

func (q *Queue) Submit(payload interface{}, priority int) (*Job, error) {
	conn := q.client.getConn()
	defer q.client.putConn(conn)

	j := &Job{
		Payload:  payload,
		Delayed:  false,
		Priority: priority,
	}

	if err := q.persistNewJob(j, conn); err != nil {
		return nil, err
	}

	if _, err := conn.LPush(q.key(j.Priority), q.jobKey(j)); err != nil {
		return nil, err
	}

	return j, nil
}

func (q *Queue) SubmitDelayed(payload interface{}, d time.Duration) (*Job, error) {
	conn := q.client.getConn()
	defer q.client.putConn(conn)

	j := &Job{
		Payload:      payload,
		Delayed:      true,
		DelayedUntil: time.Now().Add(d).UTC(),
	}

	if err := q.persistNewJob(j, conn); err != nil {
		return nil, err
	}

	if _, err := conn.ZAddNX(q.delayedKey(), timeAsFloat(j.DelayedUntil), q.jobKey(j)); err != nil {
		return nil, err
	}

	return j, nil
}

func (q *Queue) Retry(j *Job, d time.Duration) error {
	conn := q.client.getConn()
	defer q.client.putConn(conn)

	j.Delayed = true
	j.DelayedUntil = time.Now().UTC().Add(d)

	if err := q.persistJob(j, conn, "delayed", "delayed_until"); err != nil {
		return err
	}

	if _, err := conn.ZAddNX(q.delayedKey(), timeAsFloat(j.DelayedUntil), q.jobKey(j)); err != nil {
		return err
	}

	return nil
}

func (q *Queue) Wait() (*Job, error) {
	conn := q.client.getConn()
	defer q.client.putConn(conn)

	queues := make([]string, maxPriority-minPriority+1)
	for i := minPriority; i <= maxPriority; i++ {
		queues[i] = q.key(i)
	}

	results, err := conn.BRPop(0, queues...)
	if err != nil {
		return nil, err
	}

	jobKey := results[1]

	if _, err := conn.HIncr(jobKey, "num_attempts"); err != nil {
		return nil, err
	}

	j, err := unmarshalJob(conn, jobKey)
	j.Queue = q
	j.Client = q.client

	return j, nil
}
