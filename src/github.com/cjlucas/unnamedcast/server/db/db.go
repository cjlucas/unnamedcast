package db

import (
	"errors"
	"io"
	"time"

	"gopkg.in/mgo.v2"
)

var ErrNotFound = mgo.ErrNotFound
var ErrOutdatedResource = errors.New("resource is out of date")

func IsDup(err error) bool {
	return mgo.IsDup(err)
}

// DB provides an access layer to the models within the system through collections.
// It's goal is to abstract all database operations out of the App.
type DB struct {
	db *mgo.Database
	s  *mgo.Session

	Users UserCollection
	Feeds FeedCollection
	Items ItemCollection
	Logs  LogCollection
}

func New(url string) (*DB, error) {
	s, err := mgo.DialWithTimeout(url, 2*time.Second)
	if err != nil {
		return nil, err
	}

	db := s.DB("")

	// fetch indexes, replace if changed
	// TODO: Research EnsureIndex to see if it handles checking if the
	// index has changed, or if we have to write that ourselves

	return &DB{
		db: db,
		s:  s,

		Users: UserCollection{
			collection{c: db.C("users")},
		},

		Feeds: FeedCollection{
			collection{c: db.C("feeds")},
		},

		Items: ItemCollection{
			collection{c: db.C("items")},
		},

		Logs: LogCollection{
			collection{c: db.C("logs")},
		},
	}, nil
}

func (db *DB) Drop() error {
	return db.db.DropDatabase()
}

func handleDBError(s *mgo.Session, f func() error) error {
	for i := 0; i < 5; i++ {
		switch err := f(); err {
		case io.EOF:
			s.Refresh()
		default:
			return err
		}
	}

	return nil
}

type Query interface {
	All(result interface{}) error
	Count() (int, error)
	One(result interface{}) error
	Select(selector interface{}) Query
	Sort(fields ...string) Query
	Limit(n int) Query
}

type Index struct {
	Name   string
	Key    []string
	Unique bool
}

func mgoIndexForIndex(idx Index) mgo.Index {
	return mgo.Index{
		Name:       idx.Name,
		Key:        idx.Key,
		Unique:     idx.Unique,
		Background: true,
		DropDups:   true,
	}
}

type query struct {
	s *mgo.Session
	q *mgo.Query
}

func (q *query) All(result interface{}) error {
	return handleDBError(q.s, func() error {
		return q.q.All(result)
	})
}

func (q *query) Count() (int, error) {
	var n int
	err := handleDBError(q.s, func() error {
		var err error
		n, err = q.q.Count()
		return err
	})

	return n, err
}

func (q *query) One(result interface{}) error {
	return handleDBError(q.s, func() error {
		return q.q.One(result)
	})
}

func (q *query) Select(selector interface{}) Query {
	q.q = q.q.Select(selector)
	return q
}

func (q *query) Sort(fields ...string) Query {
	q.q = q.q.Sort(fields...)
	return q
}

func (q *query) Limit(n int) Query {
	q.q = q.q.Limit(n)
	return q
}

type Pipe struct {
	s *mgo.Session
	p *mgo.Pipe
}

func (p *Pipe) All(result interface{}) error {
	return handleDBError(p.s, func() error {
		return p.p.All(result)
	})
}

func (p *Pipe) One(result interface{}) error {
	return handleDBError(p.s, func() error {
		return p.p.One(result)
	})
}
