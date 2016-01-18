package yajq

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"gopkg.in/redis.v3"
)

var defaultClient = NewClient(nil)

type Client struct {
	opts     *Options
	connPool sync.Pool
}

type Options struct {
	Host        string
	Port        int
	Prefix      string
	ConnFactory func() Conn
}

func Configure(opts *Options) {
	defaultClient = NewClient(opts)
}

func NewClient(opts *Options) *Client {
	if opts == nil {
		opts = &Options{}
	}

	if opts.Host == "" {
		opts.Host = "localhost"
	}

	if opts.Port == 0 {
		opts.Port = 6379
	}

	if opts.Prefix == "" {
		opts.Prefix = "yajq"
	}

	if opts.ConnFactory == nil {
		opts.ConnFactory = func() Conn {
			r := redis.NewClient(&redis.Options{
				Addr: fmt.Sprintf("%s:%d", opts.Host, opts.Port),
			})
			return &GoRedisAdapter{R: r}
		}
	}

	c := Client{opts: opts}
	c.connPool = sync.Pool{New: func() interface{} {
		return opts.ConnFactory()
	}}

	return &c
}

func (c *Client) GetQueue(name string) *Queue {
	return &Queue{
		name:   name,
		client: c,
	}
}

func (c *Client) buildKey(s ...string) string {
	s = append([]string{c.opts.Prefix}, s...)
	return strings.Join(s, ":")
}

func GetQueue(name string) *Queue {
	return defaultClient.GetQueue(name)
}

func Submit(queue string, priority int, payload interface{}) (*Job, error) {
	return defaultClient.GetQueue(queue).Submit(priority, payload)
}

func SubmitDelayed(queue string, priority int, payload interface{}, t time.Time) (*Job, error) {
	return defaultClient.GetQueue(queue).SubmitDelayed(priority, payload, t)
}
