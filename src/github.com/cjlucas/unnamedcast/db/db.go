package db

import (
	"errors"
	"fmt"
	"io"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var ErrNotFound = mgo.ErrNotFound
var ErrOutdatedResource = errors.New("resource is out of date")

func IsDup(err error) bool {
	return mgo.IsDup(err)
}

type M bson.M

type ID struct {
	bson.ObjectId
}

func (id ID) GetBSON() (interface{}, error) {
	return id.ObjectId, nil
}

func (id *ID) SetBSON(raw bson.Raw) error {
	id.ObjectId = bson.ObjectId(raw.Data)
	return nil
}

func IDFromString(id string) (ID, error) {
	if !bson.IsObjectIdHex(id) {
		return ID{}, errors.New("invalid id")
	}

	return ID{bson.ObjectIdHex(id)}, nil
}

func NewID() ID {
	return ID{bson.NewObjectId()}
}

// DB provides an access layer to the models within the system through collections.
type DB struct {
	s           *mgo.Session
	cfg         Config
	collections []*collection

	Users UserCollection
	Feeds FeedCollection
	Items ItemCollection
	Logs  LogCollection
	Jobs  JobCollection
}

type Config struct {
	URL                string
	Clean              bool
	ForceIndexCreation bool
}

func New(cfg Config) (*DB, error) {
	s, err := mgo.DialWithTimeout(cfg.URL, 2*time.Second)
	if err != nil {
		return nil, err
	}
	ret := &DB{s: s}

	if cfg.Clean {
		if err := ret.Drop(); err != nil {
			return nil, fmt.Errorf("error dropping database: %s", err)
		}
	}

	ret.addCollection("users", &ret.Users.collection, User{})
	ret.addCollection("feeds", &ret.Feeds.collection, Feed{})
	ret.addCollection("items", &ret.Items.collection, Item{})
	ret.addCollection("logs", &ret.Logs.collection, Log{})
	ret.addCollection("jobs", &ret.Jobs.collection, Job{})
	ret.addSubCollection(&ret.Users.ItemStateCollection, ItemState{})

	for _, c := range ret.collections {
		if err := c.CreateIndexes(cfg.ForceIndexCreation); err != nil {
			return nil, fmt.Errorf("error creating indexes: %s", err)
		}
	}

	return ret, nil
}

func (db *DB) db() *mgo.Database {
	// db specified in url will be used if empty string is given
	return db.s.DB("")
}

func (db *DB) addCollection(name string, c *collection, m interface{}) {
	db.collections = append(db.collections, c)
	c.c = db.db().C(name)
	c.ModelInfo = newModelInfo(m)
}

func (db *DB) addSubCollection(c *collection, m interface{}) {
	db.collections = append(db.collections, c)
	c.ModelInfo = newModelInfo(m)
}

func (db *DB) Drop() error {
	return db.db().DropDatabase()
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

type Result struct {
	s *mgo.Session
	q *mgo.Query
}

func (r *Result) All(result interface{}) error {
	return handleDBError(r.s, func() error {
		return r.q.All(result)
	})
}

func (r *Result) Count() (int, error) {
	var n int
	err := handleDBError(r.s, func() error {
		var err error
		n, err = r.q.Count()
		return err
	})

	return n, err
}

func (r *Result) One(result interface{}) error {
	return handleDBError(r.s, func() error {
		return r.q.One(result)
	})
}

func (r *Result) Distinct(key string, result interface{}) error {
	return handleDBError(r.s, func() error {
		return r.q.Distinct(key, result)
	})
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
