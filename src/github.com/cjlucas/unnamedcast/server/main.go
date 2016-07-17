package main

import (
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/cjlucas/koda-go"
	"github.com/cjlucas/unnamedcast/db"
	"github.com/cjlucas/unnamedcast/server/endpoint"
	"github.com/cjlucas/unnamedcast/server/middleware"
	"github.com/cjlucas/unnamedcast/server/queryparser"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"
)

type App struct {
	DB   *db.DB
	Koda *koda.Client
	g    *gin.Engine
}

type Config struct {
	DB   *db.DB
	Koda *koda.Client
}

// TODO: Make default App usable and remove Config.
// Perform setupRoutes in Run()
func NewApp(cfg Config) *App {
	app := App{
		DB:   cfg.DB,
		Koda: cfg.Koda,
	}

	app.setupRoutes()
	return &app
}

func (app *App) RegisterEndpoint(e endpoint.Interface) gin.HandlerFunc {
	queryParamInfo := queryparser.NewQueryParamInfo(e)
	endpointType := reflect.TypeOf(e).Elem()

	return func(c *gin.Context) {
		// Create type
		v := reflect.New(endpointType)
		endpoint := v.Interface().(endpoint.Interface)

		// Inspect type for any special properties to inject
		for i := 0; i < v.Elem().NumField(); i++ {
			f := v.Elem().Field(i)
			switch f.Interface().(type) {
			case *db.DB:
				f.Set(reflect.ValueOf(app.DB))
			case *koda.Client:
				f.Set(reflect.ValueOf(app.Koda))
			}
		}

		middleware.ParseQueryParams(&queryParamInfo, endpoint)(c)

		for _, middleware := range endpoint.Bind() {
			middleware(c)
			if c.IsAborted() {
				return
			}
		}

		endpoint.Handle(c)
	}
}

func (app *App) setupRoutes() {
	app.g = gin.Default()

	app.g.GET("/search_feeds", app.RegisterEndpoint(&endpoint.SearchFeeds{}))
	app.g.GET("/login", app.RegisterEndpoint(&endpoint.Login{}))

	api := app.g.Group("/api", middleware.LogErrors(app.DB.Logs))

	api.GET("/users", app.RegisterEndpoint(&endpoint.GetUsers{}))
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
	api.GET("/feeds/:id", app.RegisterEndpoint(&endpoint.GetFeed{}))
	api.PUT("/feeds/:id", app.RegisterEndpoint(&endpoint.UpdateFeed{}))
	api.GET("/feeds/:id/items", app.RegisterEndpoint(&endpoint.GetFeedItems{}))
	api.GET("/feeds/:id/users", app.RegisterEndpoint(&endpoint.GetFeedUsers{}))
	api.POST("/feeds/:id/items", app.RegisterEndpoint(&endpoint.CreateFeedItem{}))
	api.GET("/feeds/:id/items/:itemID", app.RegisterEndpoint(&endpoint.GetFeedItem{}))
	api.PUT("/feeds/:id/items/:itemID", app.RegisterEndpoint(&endpoint.UpdateFeedItem{}))

	api.GET("/jobs", app.RegisterEndpoint(&endpoint.GetJobs{}))
	api.GET("/jobs/:id", app.RegisterEndpoint(&endpoint.GetJob{}))
	api.POST("/jobs", app.RegisterEndpoint(&endpoint.CreateJob{}))

	api.GET("/stats/queues", app.RegisterEndpoint(&endpoint.GetQueueStats{}))
}

func (app *App) Run(addr string) error {
	return app.g.Run(addr)
}

func getenv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func main() {
	c := cron.New()

	kodaClient := koda.NewClient(&koda.Options{
		URL: getenv("REDIS_URL", "redis://localhost:6379"),
	})

	url, err := url.Parse(getenv("API_URL", "http://localhost:80"))
	if err != nil {
		panic(err)
	}

	port := 80
	if s := strings.Split(url.Host, ":"); len(s) == 2 {
		port, _ = strconv.Atoi(s[1])
	}

	dbConn, err := db.New(db.Config{
		URL: getenv("DB_URL", "mongodb://localhost/cast"),
	})
	if err != nil {
		panic(fmt.Errorf("Failed to connect to DB: %s", err))
	}

	app := NewApp(Config{
		DB:   dbConn,
		Koda: kodaClient,
	})

	c.AddFunc("0 */10 * * * *", func() {
		fmt.Println("Updating user feeds")
		ep := endpoint.CreateJob{
			DB:   dbConn,
			Koda: kodaClient,
			Job: db.Job{
				Queue:    "update-user-feeds",
				Priority: 0,
			},
		}

		if _, err := ep.Create(); err != nil {
			fmt.Println("Error updating user feeds:", err)
			return
		}
	})

	c.Start()

	app.Run(fmt.Sprintf("0.0.0.0:%d", port))
}
