package db

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Log struct {
	ID            bson.ObjectId       `bson:"_id,omitempty" json:"id"`
	Method        string              `bson:"method" json:"method"`
	RequestHeader map[string][]string `bson:"request_header" json:"request_header"`
	RequestBody   string              `bson:"request_body" json:"request_body"`
	URL           string              `bson:"url" json:"url"`
	StatusCode    int                 `bson:"status_code" json:"status_code"`
	RemoteAddr    string              `bson:"remote_addr" json:"remote_addr"`
	Errors        interface{}         `bson:"errors" json:"errors"`
	CreationTime  time.Time           `bson:"creation_time" json:"creation_time"`
}

type LogCollection struct {
	collection
}

func (c LogCollection) LogByID(id bson.ObjectId) (*Log, error) {
	var log Log
	if err := c.FindByID(id).One(&log); err != nil {
		return nil, err
	}
	return &log, nil
}

func (c LogCollection) Create(log *Log) error {
	log.ID = bson.NewObjectId()
	log.CreationTime = time.Now().UTC()
	return c.insert(log)
}
