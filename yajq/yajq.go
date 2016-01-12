package yajq

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/redis.v3"
)

const minPriority = 0
const maxPriority = 100

type QueueInfo struct {
	Host        string
	Port        int
	Prefix      string
	Name        string
	ConnFactory func() Conn
}

type Queue struct {
	info     *QueueInfo
	connPool sync.Pool
}

func (q *Queue) getConn() Conn {
	return q.connPool.Get().(Conn)
}

func (q *Queue) putConn(c Conn) {
	q.connPool.Put(c)
}

func (q *Queue) key(priority int) string {
	return q.buildKey("queue", q.info.Name, strconv.Itoa(priority))
}

func (q *Queue) jobKey(j *Job) string {
	return q.buildKey("jobs", strconv.Itoa(j.ID))
}

func (q *Queue) logKey(j *Job) string {
	return q.buildKey("logs", strconv.Itoa(j.ID))
}

func (q *Queue) buildKey(s ...string) string {
	s = append([]string{q.info.Prefix}, s...)
	return strings.Join(s, ":")
}

func (q *Queue) incrJobID() (int, error) {
	key := q.buildKey("cur_job_id")
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

func New(info *QueueInfo) *Queue {
	if info.Host == "" {
		info.Host = "localhost"
	}

	if info.Port == 0 {
		info.Port = 6379
	}

	if info.Prefix == "" {
		info.Prefix = "yajq"
	}

	q := Queue{}
	q.info = info
	if q.info.ConnFactory == nil {
		q.info.ConnFactory = func() Conn {
			c := redis.NewClient(&redis.Options{
				Addr: fmt.Sprintf("%s:%d", info.Host, info.Port),
			})
			return &GoRedisAdapter{R: c}
		}
	}
	q.connPool = sync.Pool{New: func() interface{} {
		return q.info.ConnFactory()
	}}

	return &q
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
	fmt.Printf("%#v\n", hash)

	for _, key := range []string{"done", "completion_time"} {
		if _, err := conn.HSet(q.jobKey(j), key, hash[key]); err != nil {
			return err
		}
	}

	return nil
}

func (q *Queue) Logf(j *Job, f string, vals ...interface{}) error {
	s := fmt.Sprintf(f, vals...)

	conn := q.getConn()
	defer q.putConn(conn)

	_, err := conn.RPush(q.logKey(j), s)
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

func getHash(r redis.Client, key string, hashKeys ...string) (map[string]string, error) {
	m := make(map[string]string)

	for _, hk := range hashKeys {
		resp := r.HGet(key, hk)
		if err := resp.Err(); err != nil {
			return m, err
		}
		m[key] = resp.Val()
	}

	return m, nil
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
