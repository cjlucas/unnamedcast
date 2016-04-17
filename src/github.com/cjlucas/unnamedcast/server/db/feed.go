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

func (db *DB) FindFeeds(q interface{}) Query {
	return &query{
		s: db.s,
		q: db.feeds().Find(q),
	}
}

func (db *DB) FindFeedByID(id bson.ObjectId) Query {
	return db.FindFeeds(bson.M{"_id": id})
}

func (db *DB) CreateFeed(feed *Feed) error {
	return db.feeds().Insert(feed)
}

func (db *DB) UpdateFeed(feed *Feed) error {
	var origFeed Feed
	if err := db.FindFeedByID(feed.ID).One(&origFeed); err != nil {
		return err
	}

	ignoredFields := []string{"ID", "Items", "CreationTime", "ModificationTime"}

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

	didChange := CopyModel(origFeed, feed, ignoredFields...)
	itemGUIDMap := generateGUIDToItemMap(origFeed.Items)

	// TODO(clucas): Refactor to copy items from origFeed into
	// feed and update with new feed instead of old
	for i := range feed.Items {
		item := &feed.Items[i]
		// Time needs to be UTC because CopyModel will detect
		// a change if the time zones don't match.
		//
		// An alternate solution would be require UTC for all times, everywhere.
		// This would have to be done at a choke point like the JSON/BSON
		// [un]marshallers
		item.PublicationTime = item.PublicationTime.UTC()

		if origItem := itemGUIDMap[item.GUID]; origItem != nil {
			// If Item already existed, copy the new feed data over
			if CopyModel(origItem, item, "ModificationTime") {
				origItem.ModificationTime = time.Now().UTC()
				didChange = true
			}
		} else {
			// If Item is new, append it to the Item list
			item.ModificationTime = time.Now().UTC()
			origFeed.Items = append(origFeed.Items, *item)
			didChange = true
		}
	}

	if didChange {
		origFeed.ModificationTime = time.Now().UTC()
	}

	return db.feeds().UpdateId(origFeed.ID, origFeed)
}

func (db *DB) EnsureFeedIndex(idx Index) error {
	return db.feeds().EnsureIndex(mgoIndexForIndex(idx))
}
