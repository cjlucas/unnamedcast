package main

import (
	"time"

	"github.com/gin-gonic/gin"

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

func (f *Feed) Create() error {
	f.ID = bson.NewObjectId()
	f.CreationTime = time.Now().UTC()
	f.ModificationTime = f.CreationTime
	return feeds().Insert(f)
}

func (f *Feed) Update(new *Feed) error {
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
	if len(f.Category.Subcategories) == 0 &&
		len(new.Category.Subcategories) == 0 &&
		f.Category.Name == new.Category.Name {
		ignoredFields = append(ignoredFields, "Category")
	}

	didChange := CopyModel(f, new, ignoredFields...)
	itemGUIDMap := generateGUIDToItemMap(f.Items)

	for i := range new.Items {
		item := &new.Items[i]
		if origItem := itemGUIDMap[item.GUID]; origItem != nil {
			// If Item already existed, copy the new feed data over
			if CopyModel(origItem, item, "ModificationTime") {
				origItem.ModificationTime = time.Now().UTC()
				didChange = true
			}
		} else {
			// If Item is new, append it to the Item list
			item.ModificationTime = time.Now().UTC()
			f.Items = append(f.Items, *item)
			didChange = true
		}
	}

	if didChange {
		f.ModificationTime = time.Now().UTC()
	}

	return feeds().Update(bson.M{"_id": f.ID}, f)
}

func feeds() *mgo.Collection {
	return gSession.DB("test").C("feeds")
}

func loadFeed(idHex string) *Feed {
	if !bson.IsObjectIdHex(idHex) {
		return nil
	}

	var feed Feed
	err := feeds().FindId(bson.ObjectIdHex(idHex)).One(&feed)

	// TODO: reconsider swallowing this error
	if err != nil {
		return nil
	}

	feed.ModificationTime = feed.ModificationTime.UTC()
	feed.CreationTime = feed.CreationTime.UTC()
	for i := range feed.Items {
		feed.Items[i].ModificationTime = feed.Items[i].ModificationTime.UTC()
		feed.Items[i].PublicationTime = feed.Items[i].PublicationTime.UTC()
	}

	return &feed
}

func RequireValidFeedID(c *gin.Context) {
	id := c.Param("id")
	feed := loadFeed(id)

	if feed == nil {
		c.AbortWithStatus(404)
		return
	}

	c.Set("feed", feed)
}

func CreateFeed(c *gin.Context) {
	var feed Feed
	if err := c.Bind(&feed); err != nil {
		c.JSON(500, gin.H{"error": "could not unmarshal payload"})
		return
	}

	if err := feed.Create(); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, &feed)
}

func FindFeed(c *gin.Context) {
	var query bson.M

	for _, param := range []string{"url", "itunes_id"} {
		if v := c.Query(param); v != "" {
			query = bson.M{param: v}
			break
		}
	}

	if query == nil {
		c.JSON(400, gin.H{"error": "invalid query parameter"})
		return
	}

	var feed Feed
	if err := feeds().Find(query).One(&feed); err != nil {
		c.JSON(404, gin.H{"error": "no results found"})
	} else {
		c.JSON(200, &feed)
	}
}

func ReadFeed(c *gin.Context) {
	feed := c.MustGet("feed").(*Feed)
	c.JSON(200, &feed)
}

func UpdateFeed(c *gin.Context) {
	feed := c.MustGet("feed").(*Feed)

	var bodyFeed Feed
	if err := c.Bind(&bodyFeed); err != nil {
		c.AbortWithError(500, err)
		return
	}

	if err := feed.Update(&bodyFeed); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, &feed)
}
