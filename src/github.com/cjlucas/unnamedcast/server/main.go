package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

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
			c.AbortWithError(500, err)
		} else if n < 0 {
			c.AbortWithStatus(404)
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

	// app.g.GET("/search_feeds", SearchFeeds)
	// app.g.GET("/login", UserLogin)

	api := app.g.Group("/api")

	// GET /api/users
	api.GET("/users", func(c *gin.Context) {
		var users []db.User
		if err := app.DB.FindUsers(nil).All(&users); err != nil {
			c.AbortWithError(500, err)
		}
		c.JSON(200, users)
	})

	// POST /api/users
	api.POST("/users", func(c *gin.Context) {
		username := strings.TrimSpace(c.Query("username"))
		password := strings.TrimSpace(c.Query("password"))

		if username == "" || password == "" {
			c.JSON(400, gin.H{"error": "missing required parameter(s)"})
			return
		}

		if user, err := app.DB.CreateUser(username, password); err != nil {
			c.AbortWithError(500, err)
		} else {
			c.JSON(200, user)
		}
	})

	userIDEndpoints := api.Group("/users/:id", app.requireValidUserID("id"), func(c *gin.Context) {
		id := c.MustGet("userID").(bson.ObjectId)
		var user db.User
		if err := app.DB.FindUserByID(id).One(&user); err != nil {
			c.AbortWithError(500, err)
			return
		}
		c.Set("user", &user)
	})

	// GET /api/users/:id
	userIDEndpoints.GET("/", func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		c.JSON(200, &user)
	})

	// GET /api/users/:id/feeds
	userIDEndpoints.GET("/feeds", func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		c.JSON(200, &user.FeedIDs)
	})

	// PUT /api/users/:id/feeds
	userIDEndpoints.PUT("/feeds", readRequestBody, func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		body := c.MustGet("body").([]byte)

		var ids []bson.ObjectId
		if err := json.Unmarshal(body, ids); err != nil {
			c.AbortWithError(400, err)
			return
		}

		user.FeedIDs = ids
		if err := app.DB.UpdateUser(user); err != nil {
			c.AbortWithError(500, err)
			return
		}
		c.JSON(200, &user)
	})

	// GET /api/users/:id/states
	api.GET("/states", func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		c.JSON(200, &user.ItemStates)
	})

	// PUT /api/users/:id/states
	api.PUT("/users/:id/states", func(c *gin.Context) {
		user := c.MustGet("user").(*db.User)
		body := c.MustGet("body").([]byte)

		var states []db.ItemState
		if err := json.Unmarshal(body, states); err != nil {
			c.AbortWithError(400, err)
			return
		}

		user.ItemStates = states
		if err := app.DB.UpdateUser(user); err != nil {
			c.AbortWithError(500, err)
			return
		}

		// TODO(clucas): Return user instead of user.ItemStates to be consistent
		// with PUT /api/users/:id/feeds
		c.JSON(200, &user.ItemStates)
	})

	// api.POST("/feeds", CreateFeed)
	// api.GET("/feeds/:id", RequireValidFeedID, ReadFeed)
	// api.GET("/feeds/:id/users", RequireValidFeedID, GetFeedsUsers)
	// api.GET("/feeds", FindFeed)
	// api.PUT("/feeds/:id", RequireValidFeedID, UpdateFeed)
}

func (app *App) Run(addr string) error {
	return app.g.Run(addr)
}

var gSession *mgo.Session

// func SearchFeeds(c *gin.Context) {
// 	var limit int
// 	if limitStr := c.Query("limit"); limitStr == "" {
// 		limit = 50
// 	} else if i, err := strconv.Atoi(limitStr); err != nil {
// 		c.AbortWithError(500, errors.New("Error parsing limit"))
// 		return
// 	} else {
// 		limit = i
// 	}
//
// 	query := c.Query("q")
// 	if query == "" {
// 		c.AbortWithError(400, errors.New("No query given"))
// 		return
// 	}
//
// 	q := feeds().Find(bson.M{
// 		"$text": bson.M{"$search": query},
// 	})
//
// 	q.Select(bson.M{
// 		"score":     bson.M{"$meta": "textScore"},
// 		"title":     1,
// 		"category":  1,
// 		"image_url": 1,
// 	}).Sort("$textScore:score").Limit(limit)
//
// 	var results []Feed
// 	if err := q.All(&results); err != nil {
// 		c.AbortWithError(500, err)
// 	}
//
// 	if results == nil {
// 		results = make([]Feed, 0)
// 	}
//
// 	c.JSON(200, results)
// }
//
// func UserLogin(c *gin.Context) {
// 	username := strings.TrimSpace(c.Query("username"))
// 	password := strings.TrimSpace(c.Query("password"))
//
// 	if username == "" || password == "" {
// 		c.JSON(400, gin.H{"error": "missing required parameter(s)"})
// 		return
// 	}
//
// 	var user db.User
// 	if err := user.FindByName(username); err != nil {
// 		c.JSON(400, gin.H{"error": "user not found"})
// 		return
// 	}
//
// 	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
// 		c.JSON(401, gin.H{"error": "incorrect password"})
// 		return
// 	}
//
// 	c.JSON(200, &user)
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
