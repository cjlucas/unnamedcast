package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/cjlucas/unnamedcast/koda"
	"github.com/cjlucas/unnamedcast/server/db"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"

	"gopkg.in/mgo.v2"
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

// readRequestBody is middleware for reading the request body. This is necessary
// in cases where c.Bind does not work (such as when deserializing to a slice)
func readRequestBody(c *gin.Context) {
	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithError(500, err)
	} else {
		c.Set("body", data)
	}
	c.Request.Body.Close()
}

func (app *App) requireValidUserID(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := bson.ObjectIdHex(c.Param(paramName))
		n, err := app.DB.FindUserByID(id).Count()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		} else if n < 0 {
			c.AbortWithStatus(http.StatusNotFound)
		}

		c.Set("userID", id)
	}
}

func (app *App) setupIndexes() error {
	userIndexes := []db.Index{
		{Key: []string{"username"}, Unique: true},
		{Key: []string{"feedids"}, Unique: true},
	}

	feedIndexes := []db.Index{
		{Key: []string{"url"}, Unique: true},
		{Key: []string{"itunes_id"}, Unique: true},
		{Key: []string{"modification_time"}, Unique: false},
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
		if err := app.DB.FindUsers(bson.M{"username": username}); err != nil {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameter(s)"})
			return
		}

		if user, err := app.DB.CreateUser(username, password); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, user)
		}
	})

	userIDEndpoints := api.Group("/users/:id", app.requireValidUserID("id"), func(c *gin.Context) {
		id := c.MustGet("userID").(bson.ObjectId)
		var user db.User
		if err := app.DB.FindUserByID(id).One(&user); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Set("user", &user)
	})

	// GET /api/users/:id
	userIDEndpoints.GET("/", func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		c.JSON(http.StatusOK, &user)
	})

	// GET /api/users/:id/feeds
	userIDEndpoints.GET("/feeds", func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		c.JSON(http.StatusOK, &user.FeedIDs)
	})

	// PUT /api/users/:id/feeds
	userIDEndpoints.PUT("/feeds", readRequestBody, func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		body := c.MustGet("body").([]byte)

		var ids []bson.ObjectId
		if err := json.Unmarshal(body, ids); err != nil {
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
	api.GET("/states", func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		c.JSON(http.StatusOK, &user.ItemStates)
	})

	// PUT /api/users/:id/states
	api.PUT("/states", func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		body := c.MustGet("body").([]byte)

		var states []db.ItemState
		if err := json.Unmarshal(body, states); err != nil {
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
		for _, param := range []string{"url", "itunes_id"} {
			if v := c.Query(param); v != "" {
				query = bson.M{param: v}
				break
			}
		}

		if query == nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		var feed db.Feed
		if err := app.DB.FindFeeds(query).One(&feed); err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		c.JSON(200, &feed)
	})

	// POST /api/feeds
	api.POST("/feeds", func(c *gin.Context) {
		var feed db.Feed
		if err := c.Bind(&feed); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		if err := app.DB.CreateFeed(&feed); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &feed)
	})

	feedIDEndpoints := api.Group("/feeds/:id", func(c *gin.Context) {
		id := bson.ObjectIdHex(c.Param("id"))
		n, err := app.DB.FindFeedByID(id).Count()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		} else if n < 1 {
			c.AbortWithStatus(http.StatusNotFound)
		}

		c.Set("feedID", id)
	})

	// GET /api/feeds/:id
	feedIDEndpoints.GET("", func(c *gin.Context) {
		id := c.MustGet("feedID").(bson.ObjectId)
		var feed db.Feed
		if err := app.DB.FindUserByID(id).One(&feed); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, &feed)
	})

	// GET /api/feeds/:id/users
	feedIDEndpoints.GET("/users", func(c *gin.Context) {
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

	// PUT /api/feeds/:id
	feedIDEndpoints.PUT("", func(c *gin.Context) {
		var feed db.Feed
		if err := c.Bind(&feed); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
		}
		feed.ID = c.MustGet("feedID").(bson.ObjectId)

		if err := app.DB.UpdateFeed(&feed); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &feed)
	})
}

func (app *App) Run(addr string) error {
	return app.g.Run(addr)
}

var gSession *mgo.Session

// func SearchFeeds(c *gin.Context) {
// }

func main() {
	c := cron.New()
	c.AddFunc("@hourly", func() {
		fmt.Println("Updating user feeds")
		koda.Submit("update-user-feeds", 0, nil)
	})

	c.Start()

	url := os.Getenv("DB_URL")
	if url == "" {
		url = "mongodb://localhost/cast"
	}

	port, _ := strconv.Atoi(os.Getenv("API_PORT"))
	if port == 0 {
		port = 80
	}

	session, err := mgo.Dial(url)
	if err != nil {
		panic(err)
	}
	defer session.Close()
}
