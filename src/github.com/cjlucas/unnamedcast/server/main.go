package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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

type Config struct {
	DBConfig db.Config
}

func NewApp(cfg Config) (*App, error) {
	app := App{}

	db, err := db.New(cfg.DBConfig)
	if err != nil {
		return nil, err
	}
	app.DB = db

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

func ensureQueryExists(c *gin.Context) *db.Query {
	if q, ok := c.Get("query"); ok {
		return q.(*db.Query)
	}

	q := &db.Query{}
	c.Set("query", q)
	return q
}

func parseSortParams(mi db.ModelInfo, sortableFields ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		params, ok := c.MustGet("params").(SortParams)
		if !ok {
			panic("parseSortParams called with invalid configuration")
		}

		field := params.SortField()
		if field == "" {
			return
		}

		found := false
		for _, f := range sortableFields {
			if field == f {
				found = true
				break
			}
		}

		if !found {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("\"%s\" is not a sortable field", field))
			return
		}

		info, ok := mi.LookupAPIName(field)
		if !ok {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("\"%s\" is not a known field", field))
			return
		}
		field = info.BSONName

		query := ensureQueryExists(c)
		query.SortField = field
		query.SortDesc = params.Desc()
	}
}

func parseLimitParams(c *gin.Context) {
	params, ok := c.MustGet("params").(LimitParams)
	if !ok {
		panic("parseLimitParams called with invalid configuration")
	}

	query := ensureQueryExists(c)
	query.Limit = params.Limit()
}

func parseQueryParams(spec interface{}) gin.HandlerFunc {
	info := NewQueryParamInfo(spec)

	return func(c *gin.Context) {
		params, err := info.Parse(c.Request.URL.Query())
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("failed to parse query params: %s", err))
			return
		}

		c.Set("params", params)
	}
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

	app.DB.Logs.Create(&db.Log{
		Method:        c.Request.Method,
		RequestHeader: c.Request.Header,
		RequestBody:   string(body),
		URL:           c.Request.URL.String(),
		StatusCode:    c.Writer.Status(),
		RemoteAddr:    c.ClientIP(),
		Errors:        c.Errors,
	})
}

func (app *App) requireModelID(f func(id bson.ObjectId) db.Cursor, paramName, boundName string) gin.HandlerFunc {
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
		app.requireModelID(app.DB.Users.FindByID, paramName, boundName)(c)
		if c.IsAborted() {
			return
		}

		id := c.MustGet(boundName).(bson.ObjectId)

		var user db.User
		if err := app.DB.Users.FindByID(id).One(&user); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Set("user", &user)
	}
}

func (app *App) requireUserID(paramName string) gin.HandlerFunc {
	return app.requireModelID(app.DB.Users.FindByID, paramName, "userID")
}

func (app *App) requireFeedID(paramName string) gin.HandlerFunc {
	return app.requireModelID(app.DB.Feeds.FindByID, paramName, "feedID")
}

func (app *App) requireItemID(paramName string) gin.HandlerFunc {
	return app.requireModelID(app.DB.Items.FindByID, paramName, "itemID")
}

func (app *App) setupRoutes() {
	app.g = gin.Default()

	// GET /search_feeds
	// TODO: use parseLimitLimit
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

		cur := app.DB.Feeds.Find(&db.Query{
			Filter: bson.M{
				"$text": bson.M{"$search": query},
			},
			SelectedFields: []string{"title", "category", "image_url"},
			SortField:      "$textScore:score",
			Limit:          limit,
		})

		var results []db.Feed
		if err := cur.All(&results); err != nil {
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

		cur := app.DB.Users.Find(&db.Query{
			Filter: bson.M{"username": username},
			Limit:  1,
		})

		var user db.User
		if err := cur.One(&user); err != nil {
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

	type GetUsersParams struct {
		sortParams
		limitParams
	}

	// GET /api/users
	api.GET("/users",
		parseQueryParams(GetUsersParams{}),
		parseSortParams(app.DB.Users.ModelInfo, "modification_time"),
		parseLimitParams,
		func(c *gin.Context) {
			query := ensureQueryExists(c)
			var users []db.User
			if err := app.DB.Users.Find(query).All(&users); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
			}
			c.JSON(http.StatusOK, users)
		},
	)

	// POST /api/users
	api.POST("/users", func(c *gin.Context) {
		username := strings.TrimSpace(c.Query("username"))
		password := strings.TrimSpace(c.Query("password"))

		if username == "" || password == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		switch user, err := app.DB.Users.Create(username, password); {
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
		if err := app.DB.Users.Update(user); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, &user)
	})

	// GET /api/users/:id/states
	type GetUserItemStatesParams struct {
		ModifiedSince time.Time `param:"modified_since"`
	}

	api.GET("/users/:id/states",
		parseQueryParams(GetUserItemStatesParams{}),
		app.requireUserID("id"),
		func(c *gin.Context) {
			params := c.MustGet("params").(*GetUserItemStatesParams)
			userID := c.MustGet("userID").(bson.ObjectId)

			query := db.Query{Filter: make(bson.M)}
			if !params.ModifiedSince.IsZero() {
				query.Filter["modification_time"] = bson.M{"$gt": params.ModifiedSince}
			}

			states, err := app.DB.Users.FindItemStates(userID, query)
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			c.JSON(http.StatusOK, states)
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

		switch err := app.DB.Users.UpsertItemState(userID, &state); err {
		case nil:
			c.JSON(http.StatusOK, &state)
		case db.ErrOutdatedResource:
			c.JSON(http.StatusConflict, gin.H{"error": "resource is out of date"})
			c.Abort()
		default:
			c.AbortWithError(http.StatusInternalServerError, err)
		}
	})

	api.DELETE("/users/:id/states/:itemID",
		app.requireUserID("id"),
		app.requireItemID("itemID"),
		func(c *gin.Context) {
			userID := c.MustGet("userID").(bson.ObjectId)
			itemID := c.MustGet("itemID").(bson.ObjectId)

			if err := app.DB.Users.DeleteItemState(userID, itemID); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			c.Status(http.StatusOK)
		},
	)

	// GET /api/feeds
	// GET /api/feeds?url=http://url.com
	// GET /api/feeds?itunes_id=43912431
	//
	// TODO: modify the ?url and ?itunes_id variants to return a list for consistency

	type GetFeedsQueryParams struct {
		sortParams
		limitParams
		URL      string
		ITunesID int `param:"itunes_id"`
	}

	api.GET("/feeds",
		parseQueryParams(GetFeedsQueryParams{}),
		parseSortParams(app.DB.Feeds.ModelInfo, "modification_time"),
		parseLimitParams,
		func(c *gin.Context) {
			params := c.MustGet("params").(*GetFeedsQueryParams)
			query := ensureQueryExists(c)

			if params.URL != "" {
				query.Filter = bson.M{"url": params.URL}
			} else if params.ITunesID != 0 {
				query.Filter = bson.M{"itunes_id": params.ITunesID}
			}

			if query.Filter == nil {
				var feeds []db.Feed
				if err := app.DB.Feeds.Find(query).All(&feeds); err != nil {
					c.AbortWithError(http.StatusInternalServerError, err)
				} else {
					c.JSON(http.StatusOK, feeds)
				}
				return
			}

			// TODO: use a switch here
			var feed db.Feed
			if err := app.DB.Feeds.Find(query).One(&feed); err != nil {
				if err == db.ErrNotFound {
					c.AbortWithStatus(http.StatusNotFound)
				} else {
					c.AbortWithError(http.StatusInternalServerError, err)
				}
				return
			}

			c.JSON(http.StatusOK, &feed)
		},
	)

	// POST /api/feeds
	api.POST("/feeds", unmarshalFeed, func(c *gin.Context) {
		feed := c.MustGet("feed").(*db.Feed)

		switch err := app.DB.Feeds.Create(feed); {
		case db.IsDup(err):
			c.JSON(http.StatusConflict, gin.H{"reason": "duplicate url found"})
			c.Abort()
			return
		case err != nil:
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		out, err := app.DB.Feeds.FeedByID(feed.ID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, out)
	})

	// GET /api/feeds/:id
	api.GET("/feeds/:id", app.requireFeedID("id"), func(c *gin.Context) {
		id := c.MustGet("feedID").(bson.ObjectId)
		feed, err := app.DB.Feeds.FeedByID(id)
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

		existingFeed, err := app.DB.Feeds.FeedByID(feed.ID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		// Persist existing items
		feed.Items = existingFeed.Items

		if err := app.DB.Feeds.Update(feed); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &feed)
	})

	// GET /api/feeds/:id/items[?modified_since=2006-01-02T15:04:05Z07:00]
	// NOTE: modified_since date check is strictly greater than

	type GetFeedItemsParams struct {
		sortParams
		limitParams

		ModifiedSince time.Time `param:"modified_since"`
	}

	api.GET("/feeds/:id/items",
		parseQueryParams(GetFeedItemsParams{}),
		parseSortParams(app.DB.Items.ModelInfo, "modification_time"),
		parseLimitParams,
		app.requireFeedID("id"),
		func(c *gin.Context) {
			query := ensureQueryExists(c)
			params := c.MustGet("params").(*GetFeedItemsParams)
			feedID := c.MustGet("feedID").(bson.ObjectId)

			feed, err := app.DB.Feeds.FeedByID(feedID)
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			query.Filter = bson.M{
				"_id": bson.M{"$in": feed.Items},
			}
			if !params.ModifiedSince.IsZero() {
				query.Filter["modification_time"] = bson.M{"$gt": params.ModifiedSince}
			}

			var items []db.Item
			if err := app.DB.Items.Find(query).All(&items); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			c.JSON(http.StatusOK, &items)
		})

	// GET /api/feeds/:id/users
	api.GET("/feeds/:id/users", app.requireFeedID("id"), func(c *gin.Context) {
		id := c.MustGet("feedID").(bson.ObjectId)

		cur := app.DB.Users.Find(&db.Query{
			Filter: bson.M{
				"feed_ids": bson.M{
					"$in": []bson.ObjectId{id},
				},
			},
		})

		var users []db.User
		if err := cur.All(&users); err != nil {
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

		if err := app.DB.Items.Create(&item); err != nil {
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
		feed, err := app.DB.Feeds.FeedByID(id)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		feed.Items = append(feed.Items, item.ID)

		if err := app.DB.Feeds.Update(feed); err != nil {
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

		feed, err := app.DB.Feeds.FeedByID(feedID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if !feed.HasItemWithID(itemID) {
			c.AbortWithError(http.StatusNotFound, errors.New("item does not belong to feed"))
			return
		}

		var item db.Item
		if err := app.DB.Items.FindByID(itemID).One(&item); err != nil {
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

		feed, err := app.DB.Feeds.FeedByID(feedID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if !feed.HasItemWithID(itemID) {
			c.AbortWithError(http.StatusNotFound, errors.New("item does not belong to feed"))
			return
		}

		if err := app.DB.Items.Update(&item); err != nil {
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

	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:80"
	}

	url, err := url.Parse(apiURL)
	if err != nil {
		panic(err)
	}

	port := 80
	if s := strings.Split(url.Host, ":"); len(s) == 2 {
		port, _ = strconv.Atoi(s[1])
	}

	app, err := NewApp(Config{
		DBConfig: db.Config{
			URL: dbURL,
		},
	})
	if err != nil {
		panic(fmt.Errorf("Failed to connect to DB: %s (%s)", err, dbURL))
	}

	app.Run(fmt.Sprintf("0.0.0.0:%d", port))
}
