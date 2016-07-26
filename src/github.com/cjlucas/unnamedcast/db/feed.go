package db

import (
	"errors"
	"time"

	"github.com/cjlucas/unnamedcast/db/utctime"
)

type Feed struct {
	ID                ID           `bson:"_id,omitempty" json:"id"`
	Title             string       `json:"title" bson:"title" index:",text"`
	URL               string       `json:"url" bson:"url" index:",unique"`
	Author            string       `json:"author" bson:"author"`
	CreationTime      utctime.Time `json:"creation_time" bson:"creation_time"`
	ModificationTime  utctime.Time `json:"modification_time" bson:"modification_time" index:"modification_time"`
	LastScrapedTime   utctime.Time `json:"last_scraped_time" bson:"last_scraped_time"`
	ImageURL          string       `json:"image_url" bson:"image_url"`
	ITunesID          int          `json:"itunes_id" bson:"itunes_id" index:"itunes_id"`
	ITunesReviewCount int          `json:"itunes_review_count" bson:"itunes_review_count"`
	ITunesRatingCount int          `json:"itunes_rating_count" bson:"itunes_rating_count"`

	Category struct {
		Name          string   `json:"name" bson:"name"`
		Subcategories []string `json:"subcategories" bson:"subcategories"`
	} `json:"category"`
}

type Item struct {
	ID               ID            `json:"id" bson:"_id,omitempty"`
	FeedID           ID            `json:"-" bson:"feed_id" index:"feed_id"`
	GUID             string        `json:"guid" bson:"guid" index:"guid"`
	Link             string        `json:"link" bson:"link"`
	Title            string        `json:"title" bson:"title"`
	URL              string        `json:"url" bson:"url"`
	Author           string        `json:"author" bson:"author"`
	Summary          string        `json:"summary" bson:"summary"`
	Description      string        `json:"description" bson:"description"`
	Duration         time.Duration `json:"duration" bson:"duration"`
	Size             int           `json:"size" bson:"size"`
	PublicationTime  utctime.Time  `json:"publication_time" bson:"publication_time"`
	CreationTime     utctime.Time  `json:"creation_time" bson:"creation_time"`
	ModificationTime utctime.Time  `json:"modification_time" bson:"modification_time"`
	ImageURL         string        `json:"image_url" bson:"image_url"`
}

type FeedCollection struct {
	collection
}

func (c FeedCollection) FeedByID(id ID) (*Feed, error) {
	var feed Feed
	if err := c.FindByID(id).One(&feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func (c FeedCollection) Create(feed *Feed) error {
	feed.ID = NewID()
	feed.CreationTime = utctime.Now()
	feed.ModificationTime = utctime.Now()
	return c.insert(feed)
}

func (c FeedCollection) Update(feed *Feed) error {
	origFeed, err := c.FeedByID(feed.ID)
	if err != nil {
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

	if CopyModel(origFeed, feed, ignoredFields...) {
		origFeed.ModificationTime = utctime.Now()
	}

	return c.c.UpdateId(origFeed.ID, &origFeed)
}

type ItemCollection struct {
	collection
}

func (c ItemCollection) Create(item *Item) error {
	// Put a method on ID to check for empty ID
	var emptyID ID
	if item.FeedID == emptyID {
		return errors.New("feed id not set")
	}
	if item.ID == emptyID {
		item.ID = NewID()
	}
	item.CreationTime = utctime.Now()
	item.ModificationTime = utctime.Now()
	return c.upsert(M{"guid": item.GUID, "feed_id": item.FeedID}, item)
}

func (c ItemCollection) Update(item *Item) error {
	var origItem Item
	if err := c.FindByID(item.ID).One(&origItem); err != nil {
		return err
	}

	if CopyModel(&origItem, item, "CreationTime", "ModificationTime") {
		item.ModificationTime = utctime.Now()
	}

	return c.c.UpdateId(origItem.ID, &origItem)
}

func (c ItemCollection) ItemsWithFeedID(feedID ID) *Result {
	return c.Find(&Query{
		Filter: M{"feed_id": feedID},
	})
}
