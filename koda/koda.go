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

func (q *Queue) getConn() Conn {
	return q.client.connPool.Get().(Conn)
}

func (q *Queue) putConn(c Conn) {
	q.client.connPool.Put(c)
}

func (q *Queue) key(priority int) string {
	return q.client.buildKey("queue", q.name, strconv.Itoa(priority))
}

func (q *Queue) jobKey(j *Job) string {
	return q.client.buildKey("jobs", strconv.Itoa(j.ID))
}

func (q *Queue) logKey(j *Job) string {
	return q.client.buildKey("logs", strconv.Itoa(j.ID))
}

func (q *Queue) incrJobID() (int, error) {
	key := q.client.buildKey("cur_job_id")
	conn := q.getConn()
	defer q.putConn(conn)
	val, err := conn.Incr(key)
	return int(val), err
}

func (q *Queue) submitJob(j *Job) (*Job, error) {
	id, err := q.incrJobID()
	if err != nil {
		return nil, err
	}

	j.ID = id
	j.CreationTime = time.Now().UTC()

	jobKey := q.jobKey(j)

	conn := q.getConn()
	defer q.putConn(conn)

	// TODO: Should probably do some cleanup if an error was hit
	for k, v := range j.asHash() {
		if _, err := conn.HSet(jobKey, k, v); err != nil {
			return nil, err
		}
	}

	if _, err := conn.LPush(q.key(j.Priority), jobKey); err != nil {
		return nil, err
	}

	return j, err
}

func (q *Queue) UpdateProgress(j *Job, progress int) error {
	j.Progress = progress

	conn := q.getConn()
	defer q.putConn(conn)

	_, err := conn.HSet(q.jobKey(j), "progress", strconv.Itoa(progress))
	return err
}

func (q *Queue) Done(j *Job) error {
	conn := q.getConn()
	defer q.putConn(conn)

	j.done = true
	j.CompletionTime = time.Now().UTC()

	hash := j.asHash()

	for _, key := range []string{"done", "completion_time"} {
		if _, err := conn.HSet(q.jobKey(j), key, hash[key]); err != nil {
			return err
		}
	}

	return nil
}

func (q *Queue) Log(j *Job, log string) error {
	conn := q.getConn()
	defer q.putConn(conn)

	_, err := conn.RPush(q.logKey(j), log)
	return err
}

func (q *Queue) Submit(priority int, payload interface{}) (*Job, error) {
	return q.submitJob(&Job{
		Delayed:  false,
		Priority: priority,
		Payload:  payload,
	})
}

func (q *Queue) SubmitDelayed(priority int, payload interface{}, t time.Time) (*Job, error) {
	return q.submitJob(&Job{
		Delayed:      true,
		DelayedUntil: t.UTC(),
		Priority:     priority,
		Payload:      payload,
	})
}

func (q *Queue) Wait() (*Job, error) {
	conn := q.getConn()
	defer q.putConn(conn)

	queues := make([]string, maxPriority-minPriority+1)
	for i := minPriority; i <= maxPriority; i++ {
		queues[i] = q.key(i)
	}

	results, err := conn.BRPop(0, queues...)
	if err != nil {
		return nil, err
	}

	jobKey := results[1]

	j, err := unmarshalJob(conn, jobKey)

	return j, nil
}
