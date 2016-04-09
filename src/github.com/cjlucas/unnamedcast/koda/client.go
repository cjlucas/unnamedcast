package koda

import (
	"net/url"
	"strconv"
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
	URL         string
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

	if opts.URL == "" {
		opts.URL = "redis://localhost:6379"
	}

	if opts.Prefix == "" {
		opts.Prefix = "koda"
	}

	if opts.ConnFactory == nil {
		url, err := url.Parse(opts.URL)
		db, _ := strconv.Atoi(url.Path)

		// TODO: return err
		if err != nil {
			panic(err)
		}

		opts.ConnFactory = func() Conn {
			r := redis.NewClient(&redis.Options{
				Addr: url.Host,
				DB:   int64(db),
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

func (c *Client) getConn() Conn {
	return c.connPool.Get().(Conn)
}

func (c *Client) putConn(conn Conn) {
	c.connPool.Put(conn)
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

func SubmitDelayed(queue string, d time.Duration, payload interface{}) (*Job, error) {
	return defaultClient.GetQueue(queue).SubmitDelayed(d, payload)
}
