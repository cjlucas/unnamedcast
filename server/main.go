package main

import (
	"time"

	"github.com/gin-gonic/gin"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Person struct {
	ID               bson.ObjectId `bson:"_id,omitempty" json:"id"`
	Name             string        `json:"name"`
	Title            string        `json:"title"`
	CreationTime     time.Time
	ModificationTime time.Time
}

var gSession *mgo.Session

func main() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	gSession = session

	ensureIndex := func(c *mgo.Collection, idx mgo.Index) {
		if err := c.EnsureIndex(idx); err != nil {
			panic(err)
		}
	}

	ensureIndex(feeds(), mgo.Index{
		Key:      []string{"url"},
		Unique:   true,
		DropDups: true,
	})

	ensureIndex(feeds(), mgo.Index{
		Key: []string{"itunes_id"},
	})

	ensureIndex(feeds(), mgo.Index{
		Key: []string{"modification_time"},
	})

	g := gin.Default()

	api := g.Group("/api")

	api.POST("/users", CreateUser)
	api.GET("/users/:id", RequireValidUserID, ReadUser)
	api.GET("/users/:id/feeds", RequireValidUserID, GetUserFeeds)
	api.PUT("/users/:id/feeds", RequireValidUserID, UpdateUserFeeds)
	api.PUT("/users/:id/states", RequireValidUserID, UpdateUserItemStates)

	api.POST("/feeds", CreateFeed)
	api.GET("/feeds/:id", RequireValidFeedID, ReadFeed)
	api.GET("/feeds", FindFeed)
	api.PUT("/feeds/:id", RequireValidFeedID, UpdateFeed)

	g.Run(":8081")
}
