package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/cjlucas/koda-go"
	"github.com/cjlucas/unnamedcast/db"
	"github.com/cjlucas/unnamedcast/server/endpoint"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"
)

const (
	feedCtxKey   = "feed"
	userCtxKey   = "user"
	queryCtxKey  = "query"
	paramsCtxKey = "params"
	userIDCtxKey = "userID"
	feedIDCtxKey = "feedID"
	itemIDCtxKey = "itemID"
	jobIDCtxKey  = "jobID"
)

type JobCreator interface {
	CreateJob(payload interface{}) (koda.Job, error)
}

type JobSubmitter interface {
	SubmitJob(queue koda.Queue, priority int, job koda.Job) (koda.Job, error)
}

type defaultKodaClient struct{}

type App struct {
	DB           *db.DB
	g            *gin.Engine
	jobCreator   JobCreator
	jobSubmitter JobSubmitter
}

type Config struct {
	DB           *db.DB
	JobCreator   JobCreator
	JobSubmitter JobSubmitter
}

func NewApp(cfg Config) *App {
	app := App{
		DB:           cfg.DB,
		jobCreator:   cfg.JobCreator,
		jobSubmitter: cfg.JobSubmitter,
	}

	app.setupRoutes()
	return &app
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

	c.Set(feedCtxKey, &feed)
}

// TODO: Remove. Will be obsolete.
func ensureQueryExists(c *gin.Context) *db.Query {
	if q, ok := c.Get(queryCtxKey); ok {
		return q.(*db.Query)
	}

	q := &db.Query{}
	c.Set(queryCtxKey, q)
	return q
}

func parseSortParams(mi db.ModelInfo, sortableFields ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		params, ok := c.MustGet(paramsCtxKey).(SortParams)
		if !ok {
			panic("parseSortParams called with an invalid configuration")
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

// TODO: get rid of sortable fields and store sorting information in model tag
func parseSortParamsNew(mi db.ModelInfo, query *db.Query, params SortParams, sortableFields ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if params == nil {
			return
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

		query.SortField = field
		query.SortDesc = params.Desc()
	}
}

func parseLimitParams(c *gin.Context) {
	params, ok := c.MustGet(paramsCtxKey).(LimitParams)
	if !ok {
		panic("parseLimitParams called with an invalid configuration")
	}

	query := ensureQueryExists(c)
	query.Limit = params.Limit()
}

func parseLimitParamsNew(query *db.Query, params LimitParams) gin.HandlerFunc {
	return func(c *gin.Context) {
		if params != nil {
			query.Limit = params.Limit()
		}
	}
}

func parseQueryParams(spec interface{}) gin.HandlerFunc {
	info := NewQueryParamInfo(spec)

	return func(c *gin.Context) {
		params, err := info.Parse(c.Request.URL.Query())
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("failed to parse query params: %s", err))
			return
		}

		c.Set(paramsCtxKey, params)
	}
}

func parseQueryParamsNew(info *QueryParamInfo, params interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := info.ParsePtr(params, c.Request.URL.Query())
		if err != nil {
			fmt.Println(err)
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("failed to parse query params: %s", err))
			return
		}
		c.Set(paramsCtxKey, params)
	}
}

func (app *App) RegisterEndpoint(e endpoint.Interface) gin.HandlerFunc {
	queryParamInfo := NewQueryParamInfo(e)
	endpointType := reflect.TypeOf(e).Elem()

	return func(c *gin.Context) {
		// Create type
		v := reflect.New(endpointType)
		endpoint := v.Interface().(endpoint.Interface)

		// Inspect type for any special types to inject (*db.DB, *koda.Client)
		for i := 0; i < v.Elem().NumField(); i++ {
			f := v.Elem().Field(i)
			switch f.Interface().(type) {
			case *db.DB:
				f.Set(reflect.ValueOf(app.DB))
			}
		}

		// Bind query parameters
		parseQueryParamsNew(&queryParamInfo, endpoint)(c)

		// call binder
		// call each gin.HandlerFunc returned by the binder
		for _, middleware := range endpoint.Bind() {
			middleware(c)
			if c.IsAborted() {
				return
			}
		}

		endpoint.Handle(c)
	}
}

func (app *App) submitJob(job db.Job) (db.Job, error) {
	j, err := app.jobCreator.CreateJob(job.Payload)
	if err != nil {
		return db.Job{}, err
	}

	job.KodaID = j.ID
	job.State = "initial"
	job, err = app.DB.Jobs.Create(job)

	j, err = app.jobSubmitter.SubmitJob(koda.Queue{Name: job.Queue}, job.Priority, j)
	if err != nil {
		return db.Job{}, err
	}

	if err := app.DB.Jobs.UpdateState(job.ID, "queued"); err != nil {
		return db.Job{}, err
	}

	return job, nil
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

func (app *App) requireModelID(f func(id db.ID) db.Cursor, paramName, boundName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := db.IDFromString(c.Param(paramName))
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
		boundName := userIDCtxKey
		app.requireModelID(app.DB.Users.FindByID, paramName, boundName)(c)
		if c.IsAborted() {
			return
		}

		id := c.MustGet(boundName).(db.ID)

		var user db.User
		if err := app.DB.Users.FindByID(id).One(&user); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Set(userCtxKey, &user)
	}
}

func (app *App) requireUserID(paramName string) gin.HandlerFunc {
	return app.requireModelID(app.DB.Users.FindByID, paramName, userIDCtxKey)
}

func (app *App) requireFeedID(paramName string) gin.HandlerFunc {
	return app.requireModelID(app.DB.Feeds.FindByID, paramName, feedIDCtxKey)
}

func (app *App) requireItemID(paramName string) gin.HandlerFunc {
	return app.requireModelID(app.DB.Items.FindByID, paramName, itemIDCtxKey)
}

func (app *App) requireJobID(paramName string) gin.HandlerFunc {
	return app.requireModelID(app.DB.Jobs.FindByID, paramName, jobIDCtxKey)
}

func (app *App) setupRoutes() {
	app.g = gin.Default()

	// GET /search_feeds
	type SearchFeedsParams struct {
		limitParams
		Query string `param:"q,require"`
	}

	app.g.GET("/search_feeds",
		parseQueryParams(SearchFeedsParams{}),
		func(c *gin.Context) {
			params := c.MustGet(paramsCtxKey).(*SearchFeedsParams)
			query := &db.Query{
				Filter: db.M{
					"$text": db.M{"$search": params.Query},
				},
				SortField: "$textScore:score",
				SortDesc:  true,
				Limit:     params.Limit(),
			}

			if query.Limit > 50 {
				query.Limit = 50
			}

			var results []db.Feed
			if err := app.DB.Feeds.Find(query).All(&results); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			if results == nil {
				results = make([]db.Feed, 0)
			}

			c.JSON(http.StatusOK, results)
		})

	// GET /login

	type LoginParams struct {
		Username string `param:",require"`
		Password string `param:",require"`
	}

	app.g.GET("/login",
		parseQueryParams(LoginParams{}),
		func(c *gin.Context) {
			params := c.MustGet(paramsCtxKey).(*LoginParams)

			cur := app.DB.Users.Find(&db.Query{
				Filter: db.M{"username": params.Username},
				Limit:  1,
			})

			var user db.User
			if err := cur.One(&user); err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password)); err != nil {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			c.JSON(http.StatusOK, &user)
		})

	api := app.g.Group("/api", app.logErrors)

	api.GET("/users", app.RegisterEndpoint(&endpoint.GetUsers{}))

	// POST /api/users (TODO: this endpoint needs to be refactored)
	api.POST("/users", app.RegisterEndpoint(&endpoint.CreateUser{}))
	api.GET("/users/:id", app.RegisterEndpoint(&endpoint.GetUser{}))
	api.GET("/users/:id/feeds", app.RegisterEndpoint(&endpoint.GetUserFeeds{}))
	api.PUT("/users/:id/feeds", app.RegisterEndpoint(&endpoint.UpdateUserFeeds{}))

	api.GET("/users/:id/states", app.RegisterEndpoint(&endpoint.GetUserItemStates{}))
	api.PUT("/users/:id/states/:itemID", app.RegisterEndpoint(&endpoint.UpdateUserItemState{}))
	api.DELETE("/users/:id/states/:itemID", app.RegisterEndpoint(&endpoint.DeleteUserItemState{}))

	// GET /api/feeds
	// GET /api/feeds?url=http://url.com
	// GET /api/feeds?itunes_id=43912431
	//
	// TODO: modify the ?url and ?itunes_id variants to return a list for consistency
	api.GET("/feeds", app.RegisterEndpoint(&endpoint.GetFeeds{}))
	api.POST("/feeds", app.RegisterEndpoint(&endpoint.CreateFeed{}))

	// GET /api/feeds/:id
	api.GET("/feeds/:id", app.RegisterEndpoint(&endpoint.FetchFeed{}))

	api.PUT("/feeds/:id", app.RegisterEndpoint(&endpoint.UpdateFeed{}))

	// GET /api/feeds/:id/items[?modified_since=2006-01-02T15:04:05Z07:00]
	// NOTE: modified_since date check is strictly greater than

	api.GET("/feeds/:id/items", app.RegisterEndpoint(&endpoint.GetFeedItems{}))
	api.GET("/feeds/:id/users", app.RegisterEndpoint(&endpoint.GetFeedUsers{}))
	api.POST("/feeds/:id/items", app.RegisterEndpoint(&endpoint.CreateFeedItem{}))

	// GET /api/feeds/:id/items/:itemID
	// NOTE: Placeholder for feed id MUST be :id due to a limitation in gin's router
	// that is not expected to be resolved. See: https://github.com/gin-gonic/gin/issues/388
	api.GET("/feeds/:id/items/:itemID", app.RegisterEndpoint(&endpoint.GetFeedItem{}))

	api.PUT("/feeds/:id/items/:itemID", app.RegisterEndpoint(&endpoint.UpdateFeedItem{}))

	type GetJobsParams struct {
		limitParams
		sortParams

		Queue string
		State string
	}

	api.GET("/jobs",
		parseQueryParams(GetJobsParams{}),
		parseLimitParams,
		parseSortParams(app.DB.Jobs.ModelInfo, "completion_time"),
		func(c *gin.Context) {
			params := c.MustGet(paramsCtxKey).(*GetJobsParams)
			query := ensureQueryExists(c)

			filter := db.M{}
			if params.Queue != "" {
				filter["queue"] = params.Queue
			}
			if params.State != "" {
				filter["state"] = params.State
			}
			query.Filter = filter

			var jobs []db.Job
			if err := app.DB.Jobs.Find(query).All(&jobs); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			c.JSON(http.StatusOK, jobs)
		})

	api.GET("/jobs/:id",
		app.requireJobID("id"),
		func(c *gin.Context) {
			id := c.MustGet(jobIDCtxKey).(db.ID)

			var job db.Job
			if err := app.DB.Jobs.FindByID(id).One(&job); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			c.JSON(http.StatusOK, &job)
		},
	)

	api.POST("/jobs", func(c *gin.Context) {
		var job db.Job
		if err := c.BindJSON(&job); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if job.Queue == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		job, err := app.submitJob(job)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, &job)
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

	kodaClient := koda.NewClient(&koda.Options{
		URL: rdbURL,
	})

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

	dbConn, err := db.New(db.Config{
		URL: dbURL,
	})
	if err != nil {
		panic(fmt.Errorf("Failed to connect to DB: %s (%s)", err, dbURL))
	}

	app := NewApp(Config{
		DB:           dbConn,
		JobCreator:   kodaClient,
		JobSubmitter: kodaClient,
	})

	c.AddFunc("0 */10 * * * *", func() {
		fmt.Println("Updating user feeds")
		_, err := app.submitJob(db.Job{
			Queue:    "update-user-feeds",
			Priority: 0,
		})
		if err != nil {
			fmt.Println("Error updating user feeds:", err)
			return
		}
	})

	c.Start()

	app.Run(fmt.Sprintf("0.0.0.0:%d", port))
}
