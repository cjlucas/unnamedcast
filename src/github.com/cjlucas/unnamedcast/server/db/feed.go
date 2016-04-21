package db

import (
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Feed struct {
	ID                bson.ObjectId   `bson:"_id,omitempty" json:"id"`
	Title             string          `json:"title" bson:"title"`
	URL               string          `json:"url" bson:"url"`
	Author            string          `json:"author" bson:"author"`
	Items             []bson.ObjectId `json:"-" bson:"items"`
	CreationTime      time.Time       `json:"creation_time" bson:"creation_time"`
	ModificationTime  time.Time       `json:"modification_time" bson:"modification_time"`
	ImageURL          string          `json:"image_url" bson:"image_url"`
	ITunesID          int             `json:"itunes_id" bson:"itunes_id"`
	ITunesReviewCount int             `json:"itunes_review_count" bson:"itunes_review_count"`
	ITunesRatingCount int             `json:"itunes_rating_count" bson:"itunes_rating_count"`

	Category struct {
		Name          string   `json:"name" bson:"name"`
		Subcategories []string `json:"subcategories" bson:"subcategories"`
	} `json:"category"`
}

func (f *Feed) HasItemWithID(id bson.ObjectId) bool {
	for i := range f.Items {
		if f.Items[i] == id {
			return true
		}
	}

	return false
}

type Item struct {
	ID               bson.ObjectId `json:"id" bson:"_id,omitempty"`
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

func (db *DB) feeds() *mgo.Collection {
	return db.db().C("feeds")
}

func (db *DB) FindFeeds(q interface{}) Query {
	return &query{
		s: db.s,
		q: db.feeds().Find(q),
	}
}

func (db *DB) FindFeedByID(id bson.ObjectId) Query {
	return db.FindFeeds(bson.M{"_id": id})
}

func (db *DB) FeedByID(id bson.ObjectId) (*Feed, error) {
	var feed Feed
	if err := db.FindFeedByID(id).One(&feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func (db *DB) CreateFeed(feed *Feed) error {
	feed.ID = bson.NewObjectId()
	return db.feeds().Insert(feed)
}

func (db *DB) UpdateFeed(feed *Feed) error {
	var origFeed Feed
	if err := db.FindFeedByID(feed.ID).One(&origFeed); err != nil {
		return err
	}

	ignoredFields := []string{"ID", "CreationTime", "ModificationTime"}

	// Ignore Category if both are equal in the case where both subcats are 0 len
	// This is necessary due to how DeepEqual and JSON/BSON unmarshalling work.
	// BSON unmarshalling will still make the slice even if there is no subcat,
	// while JSON's unmarshaller will leave it as nil. This causes a problem
	// with DeepEqual as it does not consider slice of len 0 and nil to be equal.
	//
	// An alternative solution would be to override the behavior of BSON
	// unmarshalling for category using the bson.Setter interface to mimic
	// the behavior of JSON's unmarshaller.
	if len(origFeed.Category.Subcategories) == 0 &&
		len(feed.Category.Subcategories) == 0 &&
		origFeed.Category.Name == feed.Category.Name {
		ignoredFields = append(ignoredFields, "Category")
	}

	if CopyModel(&origFeed, feed, ignoredFields...) {
		origFeed.ModificationTime = time.Now().UTC()
	}

	return db.feeds().UpdateId(origFeed.ID, origFeed)
}

func (db *DB) EnsureFeedIndex(idx Index) error {
	return db.feeds().EnsureIndex(mgoIndexForIndex(idx))
}

func (db *DB) items() *mgo.Collection {
	return db.db().C("items")
}

func (db *DB) CreateItem(item *Item) error {
	item.ID = bson.NewObjectId()
	return db.items().Insert(item)
}

func (db *DB) UpdateItem(item *Item) error {
	var origItem Item
	if err := db.FindItemByID(item.ID).One(&origItem); err != nil {
		return err
	}

	// Time needs to be UTC because CopyModel will detect
	// a change if the time zones don't match.
	//
	// An alternate solution would be require UTC for all times, everywhere.
	// This would have to be done at a choke point like the JSON/BSON
	// [un]marshallers
	item.PublicationTime = item.PublicationTime.UTC()
	if CopyModel(&origItem, item, "CreationTime", "ModificationTime") {
		item.ModificationTime = time.Now().UTC()
	}

	return db.items().UpdateId(item.ID, item)
}

func (db *DB) FindItems(q interface{}) Query {
	return &query{
		s: db.s,
		q: db.items().Find(q),
	}
}

func (db *DB) FindItemByID(id bson.ObjectId) Query {
	return db.FindItems(bson.M{"_id": id})
}
