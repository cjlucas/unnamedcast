package main

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/gin-gonic/gin"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Feed struct {
	ID               bson.ObjectId `bson:"_id,omitempty" json:"id"`
	Title            string        `json:"title"`
	URL              string        `json:"url"`
	Author           string        `json:"author"`
	Items            []Item        `json:"items"`
	CreationTime     time.Time     `json:"creation_time"`
	ModificationTime time.Time     `json:"modification_time"`
	ImageURL         string        `json:"image_url"`
	ITunesID         int           `json:"itunes_id"`

	Category struct {
		Name          string   `json:"name"`
		Subcategories []string `json:"subcategories"`
	} `json:"category"`
}

type Item struct {
	GUID             string        `json:"guid"`
	Link             string        `json:"link"`
	Title            string        `json:"title"`
	URL              string        `json:"url"`
	Author           string        `json:"author"`
	Description      string        `json:"description"`
	Duration         time.Duration `json:"duration"`
	Size             int           `json:"size"`
	PublicationTime  time.Time     `json:"publication_time"`
	ModificationTime time.Time     `json:"modification_time"`
	ImageURL         string        `json:"image_url"`
}

func ItemsEqual(a, b *Item) bool {
	return a.GUID == b.GUID &&
		a.Title == b.Title &&
		a.Description == b.Description &&
		a.Duration == b.Duration &&
		a.Size == b.Size
}

func feeds() *mgo.Collection {
	return gSession.DB("test").C("feeds")
}

func loadFeed(idHex string, lastSyncTime time.Duration) *Feed {
	if !bson.IsObjectIdHex(idHex) {
		return nil
	}

	var feed Feed
	err := feeds().FindId(bson.ObjectIdHex(idHex)).One(&feed)

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
	feed := loadFeed(id, 0)

	if feed == nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		c.Abort()
	} else {
		c.Set("feed", feed)
	}
}

func CreateFeed(c *gin.Context) {
	// TODO: Create index for url
	var feed Feed
	rawBody, _ := ioutil.ReadAll(c.Request.Body)

	if err := json.Unmarshal(rawBody, &feed); err != nil {
		c.JSON(500, gin.H{"error": "could not unmarshal payload"})
		return
	}

	feed.ID = bson.NewObjectId()
	feed.CreationTime = time.Now().UTC()
	feed.ModificationTime = feed.CreationTime

	for i := range feed.Items {
		feed.Items[i].ModificationTime = time.Now().UTC()
	}

	if err := feeds().Insert(&feed); err != nil {
		c.JSON(500, gin.H{"error": "could not insert feed"})
	} else {
		c.JSON(200, &feed)
	}
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

func generateGUIDToItemMap(items []Item) map[string]*Item {
	guidMap := make(map[string]*Item)
	for i := range items {
		guidMap[items[i].GUID] = &items[i]
	}

	return guidMap
}

func UpdateFeedItems(c *gin.Context) {
	feed := c.MustGet("feed").(*Feed)
	rawBody, _ := ioutil.ReadAll(c.Request.Body)

	var body []Item

	if err := json.Unmarshal(rawBody, &body); err != nil {
		c.JSON(500, gin.H{"error": "error reading body"})
		return
	}

	curItemsMap := generateGUIDToItemMap(feed.Items)

	// Update modification time (if necessary)
	itemsModified := false
	for i := range body {
		item := &body[i]
		curItem := curItemsMap[item.GUID]

		if curItem == nil || !ItemsEqual(item, curItem) {
			item.ModificationTime = time.Now().UTC()
			itemsModified = true
		} else if curItem != nil {
			item.ModificationTime = curItem.ModificationTime
		}
	}

	if itemsModified {
		feed.ModificationTime = time.Now().UTC()
	}

	feed.Items = body

	if err := feeds().Update(bson.M{"_id": feed.ID}, &feed); err != nil {
		c.JSON(400, gin.H{"error": "could not update feed"})
	} else {
		c.JSON(200, loadFeed(feed.ID.Hex(), 0))
	}
}
