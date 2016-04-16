package db

import (
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Feed struct {
	ID                bson.ObjectId `bson:"_id,omitempty" json:"id"`
	Title             string        `json:"title" bson:"title"`
	URL               string        `json:"url" bson:"url"`
	Author            string        `json:"author" bson:"author"`
	Items             []Item        `json:"items" bson:"items"`
	CreationTime      time.Time     `json:"creation_time" bson:"creation_time"`
	ModificationTime  time.Time     `json:"modification_time" bson:"modification_time"`
	ImageURL          string        `json:"image_url" bson:"image_url"`
	ITunesID          int           `json:"itunes_id" bson:"itunes_id"`
	ITunesReviewCount int           `json:"itunes_review_count" bson:"itunes_review_count"`
	ITunesRatingCount int           `json:"itunes_rating_count" bson:"itunes_rating_count"`

	Category struct {
		Name          string   `json:"name" bson:"name"`
		Subcategories []string `json:"subcategories" bson:"subcategories"`
	} `json:"category"`
}

type Item struct {
	GUID             string        `json:"guid" bson:"guid"`
	Link             string        `json:"link" bson:"link"`
	Title            string        `json:"title" bson:"title"`
	URL              string        `json:"url" bson:"url"`
	Author           string        `json:"author" bson:"author"`
	Description      string        `json:"description" bson:"description"`
	Duration         time.Duration `json:"duration" bson:"duration"`
	Size             int           `json:"size" bson:"size"`
	PublicationTime  time.Time     `json:"publication_time" bson:"publication_time"`
	ModificationTime time.Time     `json:"modification_time" bson:"modification_time"`
	ImageURL         string        `json:"image_url" bson:"image_url"`
}

func generateGUIDToItemMap(items []Item) map[string]*Item {
	guidMap := make(map[string]*Item)
	for i := range items {
		guidMap[items[i].GUID] = &items[i]
	}

	return guidMap
}

func (db *DB) feeds() *mgo.Collection {
	return db.db().C("feeds")
}

func (db *DB) FindFeed(q interface{}) Query {
	return &query{
		s: db.s,
		q: db.feeds().Find(q),
	}
}

func (db *DB) CreateFeed(feed *Feed) error {
	return db.feeds().Insert(feed)
}

func (db *DB) UpdateFeed(feed *Feed) error {
	return db.feeds().UpdateId(feed.ID, feed)
}

func (db *DB) EnsureFeedIndex(idx Index) error {
	return db.feeds().EnsureIndex(mgoIndexForIndex(idx))
}
