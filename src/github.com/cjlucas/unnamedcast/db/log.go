package db

import "github.com/cjlucas/unnamedcast/db/utctime"

type Log struct {
	ID            ID                  `bson:"_id,omitempty" json:"id"`
	Method        string              `bson:"method" json:"method"`
	RequestHeader map[string][]string `bson:"request_header" json:"request_header"`
	RequestBody   string              `bson:"request_body" json:"request_body"`
	Endpoint      string              `bson:"endpoint" json:"endpoint" index:"endpoint"`
	Params        map[string]string   `bson:"params" json:"params"`
	Query         string              `bson:"query" json:"query"`
	StatusCode    int                 `bson:"status_code" json:"status_code" index:"status_code"`
	RemoteAddr    string              `bson:"remote_addr" json:"remote_addr"`
	Errors        []string            `bson:"errors" json:"errors"`
	ExecutionTime float32             `bson:"execution_time" json:"execution_time" index:"execution_time"`
	CreationTime  utctime.Time        `bson:"creation_time" json:"creation_time" index:"creation_time"`
}

type LogCollection struct {
	collection
}

func (c LogCollection) LogByID(id ID) (*Log, error) {
	var log Log
	if err := c.FindByID(id).One(&log); err != nil {
		return nil, err
	}
	return &log, nil
}

func (c LogCollection) Create(log *Log) error {
	log.ID = NewID()
	log.CreationTime = utctime.Now()
	return c.insert(log)
}
