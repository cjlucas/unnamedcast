package db

import "gopkg.in/mgo.v2/bson"

type Query struct {
	Filter         bson.M
	SortField      string
	SortDesc       bool
	SelectedFields []string
	OmittedFields  []string
	Limit          int
}
