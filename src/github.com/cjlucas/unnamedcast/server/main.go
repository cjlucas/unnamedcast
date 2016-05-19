package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/cjlucas/unnamedcast/koda"
	"github.com/cjlucas/unnamedcast/server/db"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"

	"gopkg.in/mgo.v2/bson"
)

type App struct {
	DB *db.DB
	g  *gin.Engine
}

func NewApp(dbURL string) (*App, error) {
	app := App{}

	db, err := db.New(dbURL)
	if err != nil {
		return nil, err
	}
	app.DB = db

	if err := app.setupIndexes(); err != nil {
		return nil, err
	}

	app.setupRoutes()

	return &app, nil
}

func newObjectIDFromHex(idHex string) (bson.ObjectId, error) {
	if !bson.IsObjectIdHex(idHex) {
		var id bson.ObjectId
		return id, errors.New("invalid object id")
	}
	return bson.ObjectIdHex(idHex), nil
}

func unmarshalFeed(c *gin.Context) {
	var feed db.Feed
	if err := c.BindJSON(&feed); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
	}

	if len(feed.Items) > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"reason": "items is a read-only property",
		})
		c.Abort()
	}

	c.Set("feed", &feed)
}

func (app *App) logErrors(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	c.Request.Body.Close()

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.New("could not read body"))
		return
	}

	c.Request.Body = ioutil.NopCloser(bytes.NewReader(body))
	c.Next()

	if len(c.Errors) == 0 {
		return
	}

	app.DB.CreateLog(&db.Log{
		Method:        c.Request.Method,
		RequestHeader: c.Request.Header,
		RequestBody:   string(body),
		URL:           c.Request.URL.String(),
		StatusCode:    c.Writer.Status(),
		RemoteAddr:    c.ClientIP(),
		Errors:        c.Errors,
	})
}

func (app *App) requireModelID(f func(id bson.ObjectId) db.Query, paramName, boundName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := newObjectIDFromHex(c.Param(paramName))
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		n, err := f(id).Count()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		} else if n < 1 {
			c.AbortWithStatus(http.StatusNotFound)
		}

		c.Set(boundName, id)
	}
}

func (app *App) loadUserWithID(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		boundName := "userID"
		app.requireModelID(app.DB.FindUserByID, paramName, boundName)(c)
		if c.IsAborted() {
			return
		}

		id := c.MustGet(boundName).(bson.ObjectId)

		var user db.User
		if err := app.DB.FindUserByID(id).One(&user); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Set("user", &user)
	}
}

func (app *App) requireUserID(paramName string) gin.HandlerFunc {
	return app.requireModelID(app.DB.FindUserByID, paramName, "userID")
}

func (app *App) requireFeedID(paramName string) gin.HandlerFunc {
	return app.requireModelID(app.DB.FindFeedByID, paramName, "feedID")
}

func (app *App) requireItemID(paramName string) gin.HandlerFunc {
	return app.requireModelID(app.DB.FindItemByID, paramName, "itemID")
}

func (app *App) setupIndexes() error {
	userIndexes := []db.Index{
		{Key: []string{"username"}, Unique: true},
		{Key: []string{"feedids"}},
	}

	feedIndexes := []db.Index{
		{Key: []string{"url"}, Unique: true},
		{Key: []string{"itunes_id"}},
		{Key: []string{"modification_time"}},
		{Key: []string{"$text:title"}},
	}

	for _, idx := range userIndexes {
		if err := app.DB.EnsureUserIndex(idx); err != nil {
			return err
		}
	}

	for _, idx := range feedIndexes {
		if err := app.DB.EnsureFeedIndex(idx); err != nil {
			return err
		}
	}

	return nil
}

func (app *App) setupRoutes() {
	app.g = gin.Default()

	// GET /search_feeds
	app.g.GET("/search_feeds", func(c *gin.Context) {
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

		q := app.DB.FindFeeds(bson.M{
			"$text": bson.M{"$search": query},
		})

		q.Select(bson.M{
			"score":     bson.M{"$meta": "textScore"},
			"title":     1,
			"category":  1,
			"image_url": 1,
		}).Sort("$textScore:score").Limit(limit)

		var results []db.Feed
		if err := q.All(&results); err != nil {
			c.AbortWithError(500, err)
		}

		if results == nil {
			results = make([]db.Feed, 0)
		}

		c.JSON(200, results)
	})

	// GET /login
	app.g.GET("/login", func(c *gin.Context) {
		username := strings.TrimSpace(c.Query("username"))
		password := strings.TrimSpace(c.Query("password"))

		if username == "" || password == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		var user db.User
		if err := app.DB.FindUsers(bson.M{"username": username}).One(&user); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.JSON(200, &user)
	})

	api := app.g.Group("/api", app.logErrors)

	// GET /api/users
	api.GET("/users", func(c *gin.Context) {
		var users []db.User
		if err := app.DB.FindUsers(nil).All(&users); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		c.JSON(http.StatusOK, users)
	})

	// POST /api/users
	api.POST("/users", func(c *gin.Context) {
		username := strings.TrimSpace(c.Query("username"))
		password := strings.TrimSpace(c.Query("password"))

		if username == "" || password == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		switch user, err := app.DB.CreateUser(username, password); {
		case err == nil:
			c.JSON(http.StatusOK, user)
		case db.IsDup(err):
			c.JSON(http.StatusConflict, gin.H{
				"reason": "user already exists",
			})
			c.Abort()
		default:
			c.AbortWithError(http.StatusInternalServerError, err)
		}
	})

	// GET /api/users/:id
	api.GET("/users/:id", app.loadUserWithID("id"), func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		c.JSON(http.StatusOK, &user)
	})

	// GET /api/users/:id/feeds
	api.GET("/users/:id/feeds", app.loadUserWithID("id"), func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		c.JSON(http.StatusOK, &user.FeedIDs)
	})

	// PUT /api/users/:id/feeds
	api.PUT("/users/:id/feeds", app.loadUserWithID("id"), func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)

		var ids []bson.ObjectId
		if err := c.BindJSON(&ids); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		user.FeedIDs = ids
		if err := app.DB.UpdateUser(user); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, &user)
	})

	// GET /api/users/:id/states
	api.GET("/users/:id/states", app.requireUserID("id"), func(c *gin.Context) {
		userID := c.MustGet("userID").(bson.ObjectId)
		modifiedSince := c.Query("modified_since")

		var modifiedSinceDate time.Time
		if modifiedSince != "" {
			t, err := time.Parse(time.RFC3339, modifiedSince)
			if err != nil {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
			modifiedSinceDate = t
		}

		var user db.User
		if modifiedSinceDate.IsZero() {
			if err := app.DB.FindUserByID(userID).One(&user); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		} else {
			pipeline := []bson.M{
				{"$match": bson.M{
					"_id": userID,
				}},
				{"$project": bson.M{
					"states": bson.M{
						"$filter": bson.M{
							"input": "$states",
							"as":    "state",
							"cond": bson.M{
								"$gt": []interface{}{
									"$$state.modification_time",
									modifiedSinceDate,
								},
							},
						},
					},
				},
				},
			}

			if err := app.DB.UserPipeline(pipeline).One(&user); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

		}

		c.JSON(http.StatusOK, &user.ItemStates)
	})

	api.PUT("/users/:id/states/:itemID", app.requireUserID("id"), app.requireItemID("itemID"), func(c *gin.Context) {
		userID := c.MustGet("userID").(bson.ObjectId)
		itemID := c.MustGet("itemID").(bson.ObjectId)

		var state db.ItemState
		if err := c.BindJSON(&state); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		state.ItemID = itemID

		if err := app.DB.UpsertUserState(userID, &state); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &state)
	})

	api.DELETE("/users/:id/states/:itemID", app.requireUserID("id"), app.requireItemID("itemID"), func(c *gin.Context) {
		userID := c.MustGet("userID").(bson.ObjectId)
		itemID := c.MustGet("itemID").(bson.ObjectId)

		if err := app.DB.DeleteUserState(userID, itemID); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusOK)
	})

	// GET /api/feeds?url=http://url.com
	// GET /api/feeds?itunes_id=43912431
	api.GET("/feeds", func(c *gin.Context) {
		var query bson.M
		if val := c.Query("url"); val != "" {
			query = bson.M{"url": val}
		} else if val := c.Query("itunes_id"); val != "" {
			id, err := strconv.Atoi(val)
			if err != nil {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
			query = bson.M{"itunes_id": id}
		}

		if query == nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		var feed db.Feed
		if err := app.DB.FindFeeds(query).One(&feed); err != nil {
			if err == db.ErrNotFound {
				c.AbortWithStatus(http.StatusNotFound)
			} else {
				c.AbortWithError(http.StatusInternalServerError, err)
			}
			return
		}

		c.JSON(200, &feed)
	})

	// POST /api/feeds
	api.POST("/feeds", unmarshalFeed, func(c *gin.Context) {
		feed := c.MustGet("feed").(*db.Feed)

		switch err := app.DB.CreateFeed(feed); {
		case db.IsDup(err):
			c.JSON(http.StatusConflict, gin.H{"reason": "duplicate url found"})
			c.Abort()
			return
		case err != nil:
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		out, err := app.DB.FeedByID(feed.ID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, out)
	})

	// GET /api/feeds/:id
	api.GET("/feeds/:id", app.requireFeedID("id"), func(c *gin.Context) {
		id := c.MustGet("feedID").(bson.ObjectId)
		feed, err := app.DB.FeedByID(id)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, feed)
	})

	// PUT /api/feeds/:id
	api.PUT("/feeds/:id", app.requireFeedID("id"), unmarshalFeed, func(c *gin.Context) {
		feed := c.MustGet("feed").(*db.Feed)
		feed.ID = c.MustGet("feedID").(bson.ObjectId)

		existingFeed, err := app.DB.FeedByID(feed.ID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		// Persist existing items
		feed.Items = existingFeed.Items

		if err := app.DB.UpdateFeed(feed); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &feed)
	})

	// GET /api/feeds/:id/items[?modified_since=2006-01-02T15:04:05Z07:00]
	// NOTE: modified_since date check is strictly greater than
	api.GET("/feeds/:id/items", app.requireFeedID("id"), func(c *gin.Context) {
		feedID := c.MustGet("feedID").(bson.ObjectId)
		modifiedSince := c.Query("modified_since")

		var modifiedSinceDate time.Time
		if modifiedSince != "" {
			t, err := time.Parse(time.RFC3339, modifiedSince)
			if err != nil {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
			modifiedSinceDate = t
		}

		feed, err := app.DB.FeedByID(feedID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		itemsQuery := bson.M{
			"_id": bson.M{"$in": feed.Items},
		}
		if !modifiedSinceDate.IsZero() {
			itemsQuery["modification_time"] = bson.M{"$gt": modifiedSinceDate}
		}
		var items []db.Item
		if err := app.DB.FindItems(itemsQuery).All(&items); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &items)
	})

	// GET /api/feeds/:id/users
	api.GET("/feeds/:id/users", app.requireFeedID("id"), func(c *gin.Context) {
		id := c.MustGet("feedID").(bson.ObjectId)
		query := bson.M{
			"feed_ids": bson.M{
				"$in": []bson.ObjectId{id},
			},
		}

		var users []db.User
		if err := app.DB.FindUsers(query).All(&users); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &users)
	})

	// POST /api/feeds/:id/items
	api.POST("/feeds/:id/items", app.requireFeedID("id"), func(c *gin.Context) {
		var item db.Item
		if err := c.BindJSON(&item); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		if err := app.DB.CreateItem(&item); err != nil {
			if db.IsDup(err) {
				c.JSON(http.StatusConflict, gin.H{
					"reason": "duplicate id",
				})
			} else {
				c.AbortWithError(http.StatusBadRequest, err)
			}
			return
		}

		id := c.MustGet("feedID").(bson.ObjectId)
		feed, err := app.DB.FeedByID(id)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		feed.Items = append(feed.Items, item.ID)

		if err := app.DB.UpdateFeed(feed); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &item)
	})

	// GET /api/feeds/:id/items/:itemID
	// NOTE: Placeholder for feed id MUST be :id due to a limitation in gin's router
	// that is not expected to be resolved. See: https://github.com/gin-gonic/gin/issues/388
	api.GET("/feeds/:id/items/:itemID", app.requireFeedID("id"), app.requireItemID("itemID"), func(c *gin.Context) {
		feedID := c.MustGet("feedID").(bson.ObjectId)
		itemID := c.MustGet("itemID").(bson.ObjectId)

		feed, err := app.DB.FeedByID(feedID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if !feed.HasItemWithID(itemID) {
			c.AbortWithError(http.StatusNotFound, errors.New("item does not belong to feed"))
			return
		}

		var item db.Item
		if err := app.DB.FindItemByID(itemID).One(&item); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &item)
	})

	// PUT /api/feeds/:id/items/:itemID
	api.PUT("/feeds/:id/items/:itemID", app.requireFeedID("id"), app.requireItemID("itemID"), func(c *gin.Context) {
		feedID := c.MustGet("feedID").(bson.ObjectId)
		itemID := c.MustGet("itemID").(bson.ObjectId)

		var item db.Item
		if err := c.BindJSON(&item); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		item.ID = itemID

		feed, err := app.DB.FeedByID(feedID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if !feed.HasItemWithID(itemID) {
			c.AbortWithError(http.StatusNotFound, errors.New("item does not belong to feed"))
			return
		}

		if err := app.DB.UpdateItem(&item); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &item)
	})
}

func (app *App) Run(addr string) error {
	return app.g.Run(addr)
}

func main() {
	c := cron.New()

	rdbURL := os.Getenv("REDIS_URL")
	if rdbURL == "" {
		rdbURL = "redis://localhost:6379"
	}

	koda.Configure(&koda.Options{
		URL: rdbURL,
	})

	c.AddFunc("0 */10 * * * *", func() {
		fmt.Println("Updating user feeds")
		if _, err := koda.Submit("update-user-feeds", 0, nil); err != nil {
			fmt.Println("Error updating user feeds:", err)
		}
	})

	c.Start()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "mongodb://localhost/cast"
	}

	port, _ := strconv.Atoi(os.Getenv("API_PORT"))
	if port == 0 {
		port = 80
	}

	app, err := NewApp(dbURL)
	if err != nil {
		panic(err)
	}

	app.Run(fmt.Sprintf("0.0.0.0:%d", port))
}
