package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

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
		id := c.MustGet(boundName).(bson.ObjectId)

		var user db.User
		if err := app.DB.FindUserByID(id).One(&user); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Set("user", &user)
	}
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
		{Key: []string{"itunes_id"}, Unique: true},
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

	api := app.g.Group("/api")

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

		if user, err := app.DB.CreateUser(username, password); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, user)
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
	api.GET("/users/:id/states", app.loadUserWithID("id"), func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		c.JSON(http.StatusOK, &user.ItemStates)
	})

	// PUT /api/users/:id/states
	api.PUT("/users/:id/states", app.loadUserWithID("id"), func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)

		var states []db.ItemState
		if err := c.BindJSON(&states); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		user.ItemStates = states
		if err := app.DB.UpdateUser(user); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// TODO(clucas): Return user instead of user.ItemStates to be consistent
		// with PUT /api/users/:id/feeds
		c.JSON(http.StatusOK, &user.ItemStates)
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
			c.AbortWithError(http.StatusNotFound, err)
			return
		}

		c.JSON(200, &feed)
	})

	// POST /api/feeds
	api.POST("/feeds", func(c *gin.Context) {
		var feed db.Feed
		if err := c.BindJSON(&feed); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		if err := app.DB.CreateFeed(&feed); err != nil {
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
	api.PUT("/feeds/:id", app.requireFeedID("id"), func(c *gin.Context) {
		var feed db.Feed
		if err := c.BindJSON(&feed); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
		}
		feed.ID = c.MustGet("feedID").(bson.ObjectId)

		if err := app.DB.UpdateFeed(&feed); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &feed)
	})

	// GET /api/feeds/:id/users
	api.GET("/feeds/:id/users", app.requireFeedID("id"), func(c *gin.Context) {
		id := c.MustGet("feedID").(bson.ObjectId)
		query := bson.M{
			"feedids": bson.M{
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

	// POST /feeds/:id/items
	api.POST("/feeds/:id/items", app.requireFeedID("id"), func(c *gin.Context) {
		var item db.Item
		if err := c.BindJSON(&item); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		if err := app.DB.CreateItem(&item); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
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

	// GET /feeds/:id/items/:itemID
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

	// PUT /feeds/:id/items/:itemID
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
	c.AddFunc("@hourly", func() {
		fmt.Println("Updating user feeds")
		koda.Submit("update-user-feeds", 0, nil)
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
