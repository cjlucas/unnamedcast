package db

import (
	"io"
	"time"

	"gopkg.in/mgo.v2"
)

type DB struct {
	s *mgo.Session
}

func New(url string) (*DB, error) {
	s, err := mgo.DialWithTimeout(url, 2*time.Second)
	if err != nil {
		return nil, err
	}
	return &DB{s: s}, nil
}

type Query interface {
	All(result interface{}) error
	Count() (int, error)
	One(result interface{}) error
	Select(selector interface{}) Query
	Sort(fields ...string) Query
	Limit(n int) Query
}

type query struct {
	s *mgo.Session
	q *mgo.Query
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

func (db *DB) db() *mgo.Database {
	// when given the empty string, database is defered to db name specified in New()
	return db.s.DB("")
}

func (q *query) handleDBError(f func() error) error {
	i := 0
	err := f()
	for err == io.EOF && i < 5 {
		q.s.Refresh()
		err = f()
		i++
	}
	return err
}

func (db *DB) Drop() error {
	return db.db().DropDatabase()
}

func (q *query) All(result interface{}) error {
	return q.handleDBError(func() error {
		return q.q.All(result)
	})
}

func (q *query) Count() (int, error) {
	var n int
	err := q.handleDBError(func() error {
		var err error
		n, err = q.q.Count()
		return err
	})

	return n, err
}

func (q *query) One(result interface{}) error {
	return q.handleDBError(func() error {
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
