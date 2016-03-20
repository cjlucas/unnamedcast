package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/cjlucas/unnamedcast/koda"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var gSession *mgo.Session

func SearchFeeds(c *gin.Context) {
	var limit int
	if limitStr := c.Query("limit"); limitStr == "" {
		limit = 50
	} else if i, err := strconv.Atoi(limitStr); err != nil {
		c.AbortWithError(500, errors.New("Error parsing limit"))
		return
	} else {
		limit = i
	}

	query := c.Query("q")
	if query == "" {
		c.AbortWithError(400, errors.New("No query given"))
		return
	}

	q := feeds().Find(bson.M{
		"$text": bson.M{"$search": query},
	})

	q.Select(bson.M{
		"score":     bson.M{"$meta": "textScore"},
		"title":     1,
		"category":  1,
		"image_url": 1,
	}).Sort("$textScore:score").Limit(limit)

	var results []Feed
	if err := q.All(&results); err != nil {
		c.AbortWithError(500, err)
	}

	if results == nil {
		results = make([]Feed, 0)
	}

	c.JSON(200, results)
}

func UserLogin(c *gin.Context) {
	username := strings.TrimSpace(c.Query("username"))
	password := strings.TrimSpace(c.Query("password"))

	if username == "" || password == "" {
		c.JSON(400, gin.H{"error": "missing required parameter(s)"})
		return
	}

	var user User
	if err := user.FindByName(username); err != nil {
		c.JSON(400, gin.H{"error": "user not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		c.JSON(401, gin.H{"error": "incorrect password"})
		return
	}

	c.JSON(200, &user)
}

func main() {
	c := cron.New()
	c.AddFunc("@hourly", func() {
		fmt.Println("Updating user feeds")
		koda.Submit("update-user-feeds", 0, nil)
	})

	c.Start()

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

	ensureIndex(users(), mgo.Index{
		Key:    []string{"username"},
		Unique: true,
	})

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

	g.GET("/search_feeds", SearchFeeds)
	g.GET("/login", UserLogin)

	api := g.Group("/api")

	api.GET("/users", FindAllUsers)
	api.POST("/users", CreateUser)
	api.GET("/users/:id", RequireValidUserID, ReadUser)
	api.GET("/users/:id/feeds", RequireValidUserID, GetUserFeeds)
	api.PUT("/users/:id/feeds", RequireValidUserID, UpdateUserFeeds)
	api.GET("/users/:id/states", RequireValidUserID, GetUserItemStates)
	api.PUT("/users/:id/states", RequireValidUserID, UpdateUserItemStates)

	api.POST("/feeds", CreateFeed)
	api.GET("/feeds/:id", RequireValidFeedID, ReadFeed)
	api.GET("/feeds", FindFeed)
	api.PUT("/feeds/:id", RequireValidFeedID, UpdateFeed)

	g.Run(":8081")
}
