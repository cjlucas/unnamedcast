package yajq

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/redis.v3"
)

type QueueInfo struct {
	Host   string
	Port   int
	Prefix string
	Name   string
}

type Queue struct {
	info *QueueInfo
	r    *redis.Client
}

type Job struct {
	ID             int64
	Done           bool
	CreationTime   time.Time
	CompletionTime time.Time
	Payload        interface{}
}

func buildKey(s ...string) string {
	return strings.Join(s, ":")
}

func (q *Queue) incrJobId() (int64, error) {
	key := buildKey(q.info.Prefix, "jobidcnt")
	return q.r.Incr(key).Result()
}

func (q *Queue) Submit() *Job {

}

func New(info *QueueInfo) *Queue {
	if info.Host == "" {
		info.Host = "localhost"
	}

	if info.Port == 0 {
		info.Port = 6379
	}

	q := &Queue{}
	q.info = info
	q.r = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", info.Host, info.Port),
	})

	return q
}
