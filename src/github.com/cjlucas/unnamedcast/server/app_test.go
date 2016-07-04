package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/cjlucas/unnamedcast/api"
	"github.com/cjlucas/unnamedcast/db"
	"github.com/cjlucas/unnamedcast/koda"
	"github.com/gin-gonic/gin"
)

var emptyID db.ID

type mockJobSubmitter struct{}

func (m mockJobSubmitter) Submit(queue string, priority int, payload interface{}) (*koda.Job, error) {
	return &koda.Job{
		ID:       1,
		Priority: priority,
		Payload:  payload}, nil
}

func init() {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter, _ = os.Open(os.DevNull)
}

func newTestApp() *App {
	// Initialize app with a clean database
	dbConn, err := db.New(db.Config{
		URL:                os.Getenv("DB_URL"),
		Clean:              true,
		ForceIndexCreation: true,
	})
	if err != nil {
		panic(err)
	}

	return NewApp(Config{
		DB:           dbConn,
		JobSubmitter: mockJobSubmitter{},
	})
}

func createFeed(t *testing.T, app *App, feed *db.Feed) *db.Feed {
	if err := app.DB.Feeds.Create(feed); err != nil {
		t.Fatal("Failed to create feed")
	}
	return feed
}

func createItem(t *testing.T, app *App, item *db.Item) *db.Item {
	if err := app.DB.Items.Create(item); err != nil {
		t.Fatal("Failed to create item")
	}
	return item
}

func createUser(t *testing.T, app *App, username, password string) *db.User {
	user, err := app.DB.Users.Create(username, password)
	if err != nil {
		t.Fatal("Failed to create user")
	}
	return user
}

func createJob(t *testing.T, app *App, job db.Job) db.Job {
	job, err := app.DB.Jobs.Create(job)
	if err != nil {
		t.Fatal("Failed to create user")
	}
	return job
}

func newRequest(method string, endpoint string, body interface{}) *http.Request {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		reqBody = bytes.NewReader(buf)
	} else {
		// Body cannot be nil because we are not sending request through a
		// transport and c.BindJSON/json.NewDecoder do not perform anil check
		reqBody = bytes.NewReader([]byte{})
	}

	// Use httptest.NewRequest instead (1.7+ only)
	r, err := http.NewRequest(method, endpoint, reqBody)
	if err != nil {
		panic(err)
	}

	return r
}

type endpointTestInfo struct {
	App          *App
	Request      *http.Request
	ExpectedCode int

	// Unmarshal given object from response body for further assertions
	ResponseBody interface{}
}

func testEndpoint(t *testing.T, info endpointTestInfo) {
	if info.App == nil {
		info.App = newTestApp()
	}

	w := httptest.NewRecorder()
	info.App.g.ServeHTTP(w, info.Request)

	if w.Code != info.ExpectedCode {
		t.Fatalf("Unexpected status code: %d != %d", w.Code, info.ExpectedCode)
	}

	if info.ResponseBody != nil {
		if err := json.Unmarshal(w.Body.Bytes(), info.ResponseBody); err != nil {
			t.Fatal("Unable to unmarshal response:", w.Body.String())
		}
	}
}

func TestSearchFeeds(t *testing.T) {
	t.Skip("skip until text search index is fixed")

	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("GET", "/search_feeds?q=test", nil),
		ExpectedCode: http.StatusOK,
	})
}

func TestSearchFeeds_WithoutQuery(t *testing.T) {
	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("GET", "/search_feeds", nil),
		ExpectedCode: http.StatusBadRequest,
	})
}

func TestLoginInvalidParameters(t *testing.T) {
	app := newTestApp()
	createUser(t, app, "chris", "hithere")

	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/login", nil),
		ExpectedCode: http.StatusBadRequest,
	})

	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/login?username=chris", nil),
		ExpectedCode: http.StatusBadRequest,
	})

	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/login?password=hithere", nil),
		ExpectedCode: http.StatusBadRequest,
	})
}

func TestLoginBadPassword(t *testing.T) {
	app := newTestApp()
	createUser(t, app, "chris", "hithere")

	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/login?username=chris&password=wrong", nil),
		ExpectedCode: http.StatusUnauthorized,
	})
}

func TestLogin(t *testing.T) {
	app := newTestApp()
	createUser(t, app, "chris", "hithere")

	var out db.User
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/login?username=chris&password=hithere", nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.Username != "chris" {
		t.Errorf("Username mismatch: %s != %s", out.Username, "chris")
	}
}

func TestGetUsers(t *testing.T) {
	app := newTestApp()
	createUser(t, app, "chris", "hithere")
	createUser(t, app, "john", "hithere")

	var out []db.User
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/users", nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) != 2 {
		t.Errorf("Unexpected # of users: %d != %d", len(out), 2)
	}
}

func TestCreateUserValidParams(t *testing.T) {
	app := newTestApp()
	req := newRequest("POST", "/api/users?username=chris&password=hi", nil)

	var user db.User
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &user,
	})

	expectedUsername := "chris"
	if user.Username != expectedUsername {
		t.Errorf("Username mismatch: %s != %s", user.Username, expectedUsername)
	}

	if user.ID == emptyID {
		t.Error("user.ID is invalid")
	}

	// Duplicate entry
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusConflict,
	})

	// No parameters
	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("POST", "/api/users", nil),
		ExpectedCode: http.StatusBadRequest,
	})
}

func TestGetUser(t *testing.T) {
	app := newTestApp()
	user := createUser(t, app, "chris", "hithere")

	var out db.User
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", fmt.Sprintf("/api/users/%s", user.ID.Hex()), nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.ID != user.ID {
		t.Errorf("ID mismatch: %s != %s", out.ID, user.ID)
	}

	// Non-existant ID
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", fmt.Sprintf("/api/users/%s", bson.NewObjectId().Hex()), nil),
		ExpectedCode: http.StatusNotFound,
	})
}

func TestGetUserFeeds(t *testing.T) {
	app := newTestApp()
	user := createUser(t, app, "chris", "hithere")
	user.FeedIDs = append(user.FeedIDs, db.NewID())

	if err := app.DB.Users.Update(user); err != nil {
		t.Fatal("Could not update user:", err)
	}

	var out []db.ID
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", fmt.Sprintf("/api/users/%s/feeds", user.ID.Hex()), nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) == len(user.FeedIDs) {
		if out[0] != user.FeedIDs[0] {
			t.Errorf("ID mismatch: %s != %s", out[0], user.FeedIDs[0])
		}
	} else {
		t.Errorf("Unexpected # of feed IDs: %d != %d", len(out), len(user.FeedIDs))
	}
}

func TestGetUserItemStates(t *testing.T) {
	app := newTestApp()
	user := createUser(t, app, "chris", "hithere")
	user.ItemStates = append(user.ItemStates, db.ItemState{
		ItemID:   db.NewID(),
		Position: 5,
	})

	if err := app.DB.Users.Update(user); err != nil {
		t.Fatal("Could not update user:", err)
	}

	var out []db.ItemState
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", fmt.Sprintf("/api/users/%s/states", user.ID.Hex()), nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) == len(user.ItemStates) {
		if out[0].ItemID != user.ItemStates[0].ItemID {
			t.Errorf("ID mismatch: %s != %s", out[0].ItemID, user.ItemStates[0].ItemID)
		}
	} else {
		t.Errorf("Unexpected # of feed IDs: %d != %d", len(out), len(user.ItemStates))
	}
}

func TestGetUserItemStates_WithModifiedSinceParam(t *testing.T) {
	app := newTestApp()
	user := createUser(t, app, "chris", "hithere")
	user.ItemStates = append(user.ItemStates, db.ItemState{
		ItemID:           db.NewID(),
		Position:         5,
		ModificationTime: time.Now(),
	})

	if err := app.DB.Users.Update(user); err != nil {
		t.Fatal("Could not update user:", err)
	}

	modTime := user.ItemStates[0].ModificationTime

	urlWithTime := func(modTime time.Time) string {
		return fmt.Sprintf("/api/users/%s/states?modified_since=%s", user.ID.Hex(), modTime.Format(time.RFC3339))
	}

	var out []db.ItemState
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", urlWithTime(modTime.Add(-1*time.Second)), nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) != 1 {
		t.Errorf("Unexpected response length: %d != 1", len(out))
	}

	out = make([]db.ItemState, 0)

	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", urlWithTime(modTime.Add(1*time.Second)), nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) != 0 {
		t.Errorf("Unexpected response length: %d != 0", len(out))
	}
}

func TestPutUserFeeds(t *testing.T) {
	app := newTestApp()
	user := createUser(t, app, "chris", "hithere")

	ids := []db.ID{db.NewID()}
	req := newRequest("PUT", fmt.Sprintf("/api/users/%s/feeds", user.ID.Hex()), ids)
	var out db.User
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out.FeedIDs) == len(ids) {
		if out.FeedIDs[0] != ids[0] {
			t.Errorf("Feed ID mismatch: %s != %s", out.FeedIDs[0], ids[0])
		}
	} else {
		t.Errorf("Unexpected # of feed IDs: %d != %d", len(out.FeedIDs), len(ids))
	}
}

func TestPutUserItemState(t *testing.T) {
	app := newTestApp()
	user := createUser(t, app, "chris", "hithere")
	item := createItem(t, app, &db.Item{
		GUID: "http://google.com/1",
	})

	state := api.ItemState{
		ItemID:   item.ID.Hex(),
		Position: 5,
	}
	req := newRequest("PUT", fmt.Sprintf("/api/users/%s/states/%s", user.ID.Hex(), state.ItemID), state)
	var out api.ItemState
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.ItemID != state.ItemID {
		t.Errorf("Item ID mismatch: %s != %s", out.ItemID, state.ItemID)
	}
}

func TestPutUserItemState_WithOutdatedState(t *testing.T) {
	app := newTestApp()
	user := createUser(t, app, "chris", "hithere")
	item := createItem(t, app, &db.Item{
		GUID: "http://google.com/1",
	})

	state := api.ItemState{
		ItemID:   item.ID.Hex(),
		State:    api.StateInProgress,
		Position: 5,
	}

	req := newRequest("PUT", fmt.Sprintf("/api/users/%s/states/%s", user.ID.Hex(), state.ItemID), &state)
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &state,
	})

	state.ModificationTime = state.ModificationTime.Add(-1 * time.Second)

	req = newRequest("PUT", fmt.Sprintf("/api/users/%s/states/%s", user.ID.Hex(), state.ItemID), &state)
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusConflict,
	})
}

func TestDeleteUserItemState(t *testing.T) {
	app := newTestApp()
	user := createUser(t, app, "chris", "hithere")
	item := createItem(t, app, &db.Item{
		GUID: "http://google.com/1",
	})

	user.ItemStates = append(user.ItemStates, db.ItemState{
		ItemID:   item.ID,
		State:    api.StatePlayed,
		Position: 0,
	})

	if err := app.DB.Users.Update(user); err != nil {
		t.Fatal("Could not update user:", err)
	}

	req := newRequest("DELETE", fmt.Sprintf("/api/users/%s/states/%s", user.ID.Hex(), item.ID.Hex()), nil)
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
	})

	if err := app.DB.Users.FindByID(user.ID).One(&user); err != nil {
		t.Fatal("Could not find user:", err)
	}

	if len(user.ItemStates) != 0 {
		t.Errorf("# of item states mismatch: %d != %d", len(user.ItemStates), 0)
	}
}

func TestCreateFeed(t *testing.T) {
	in := db.Feed{URL: "http://google.com"}

	var out db.Feed
	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("POST", "/api/feeds", &in),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.ID == emptyID {
		t.Error("ID is invalid")
	}

	if out.URL != in.URL {
		t.Errorf("URL mismatch: %s != %s", out.URL, in.URL)
	}

	// Duplicate entry
	app := newTestApp()
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("POST", "/api/feeds", &in),
		ExpectedCode: http.StatusOK,
	})
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("POST", "/api/feeds", &in),
		ExpectedCode: http.StatusConflict,
	})
	app.DB.Drop()

	// No body given
	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("POST", "/api/feeds", nil),
		ExpectedCode: http.StatusBadRequest,
	})
}

func TestCreateFeedWithItems(t *testing.T) {
	app := newTestApp()
	feed := &db.Feed{
		URL:   "http://google.com",
		Items: []db.ID{db.NewID()},
	}

	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("POST", "/api/feeds", feed),
		ExpectedCode: http.StatusConflict,
	})
}

func TestGetFeed(t *testing.T) {
	app := newTestApp()
	feed := createFeed(t, app, &db.Feed{URL: "http://google.com"})

	req := newRequest("GET", fmt.Sprintf("/api/feeds/%s", feed.ID.Hex()), nil)
	var out db.Feed
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.ID != feed.ID {
		t.Errorf("ID mismatch: %s != %s", out.ID, feed.ID)
	}
}

func TestGetFeedWithoutParams(t *testing.T) {
	app := newTestApp()
	createFeed(t, app, &db.Feed{URL: "http://google.com"})
	createFeed(t, app, &db.Feed{URL: "http://google2.com"})

	var out []db.Feed
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/feeds", nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) != 2 {
		t.Errorf("Feed count mismatch: %d != %d", len(out), 2)
	}
}

func TestGetFeedSortedByModificationTime(t *testing.T) {
	app := newTestApp()
	feed1 := createFeed(t, app, &db.Feed{
		URL:              "http://google.com",
		ModificationTime: time.Now(),
	})
	feed2 := createFeed(t, app, &db.Feed{
		URL:              "http://google2.com",
		ModificationTime: time.Now(),
	})

	var out []db.Feed
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/feeds?sort_by=modification_time&sort_order=asc", nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) != 2 || out[0].URL != feed1.URL || out[1].URL != feed2.URL {
		t.Error("feed order is incorrect sort_order=asc")
	}

	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/feeds?sort_by=modification_time&sort_order=desc", nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) != 2 || out[0].URL != feed2.URL || out[1].URL != feed1.URL {
		t.Error("feed order is incorrect sort_order=desc")
	}
}

func TestGetFeedByURL(t *testing.T) {
	app := newTestApp()
	feed := createFeed(t, app, &db.Feed{URL: "http://google.com"})

	req := newRequest("GET", fmt.Sprintf("/api/feeds?url=%s", feed.URL), nil)
	var out db.Feed
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.ID != feed.ID {
		t.Errorf("ID mismatch: %s != %s", out.ID, feed.ID)
	}

	// Non-existant URL
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/feeds?url=wrongurl", nil),
		ExpectedCode: http.StatusNotFound,
	})
}

func TestGetFeedByITunesID(t *testing.T) {
	app := newTestApp()
	feed := createFeed(t, app, &db.Feed{
		URL:      "http://google.com",
		ITunesID: 12345,
	})

	req := newRequest("GET", fmt.Sprintf("/api/feeds?itunes_id=%d", feed.ITunesID), nil)
	var out db.Feed
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.ID != feed.ID {
		t.Errorf("ID mismatch: %s != %s", out.ID, feed.ID)
	}

	// Non-existant iTunes ID
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/feeds?itunes_id=123", nil),
		ExpectedCode: http.StatusNotFound,
	})

	// Invalid parameter
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/feeds?itunes_id=notanum", nil),
		ExpectedCode: http.StatusBadRequest,
	})
}

func TestPutFeed(t *testing.T) {
	app := newTestApp()
	feed := createFeed(t, app, &db.Feed{
		URL:      "http://google.com",
		ITunesID: 12345,
	})
	feed.Title = "Something"

	url := fmt.Sprintf("/api/feeds/%s", feed.ID.Hex())
	var out db.Feed
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("PUT", url, feed),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.Title != feed.Title {
		t.Errorf("Unexpected title: %s != %s", out.Title, feed.Title)
	}

	// No body given
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("PUT", url, nil),
		ExpectedCode: http.StatusBadRequest,
	})
}

// Regression test to ensure items array is not modified
func TestPutFeedWithExistingItems(t *testing.T) {
	app := newTestApp()
	item := createItem(t, app, &db.Item{
		GUID: "http://google.com/item",
	})
	feed := createFeed(t, app, &db.Feed{
		URL:   "http://google.com",
		Items: []db.ID{item.ID},
	})
	feed.Items = []db.ID{}

	url := fmt.Sprintf("/api/feeds/%s", feed.ID.Hex())
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("PUT", url, feed),
		ExpectedCode: http.StatusOK,
	})

	feed, err := app.DB.Feeds.FeedByID(feed.ID)
	if err != nil {
		t.Fatal("Could not find feed")
	}

	if len(feed.Items) != 1 {
		t.Error("Items list was emptied")
	}
}

func TestPutFeedWithItems(t *testing.T) {
	app := newTestApp()
	item := createItem(t, app, &db.Item{
		GUID: "http://google.com/item",
	})
	feed := createFeed(t, app, &db.Feed{
		URL: "http://google.com",
	})
	feed.Items = append(feed.Items, item.ID)

	url := fmt.Sprintf("/api/feeds/%s", feed.ID.Hex())
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("PUT", url, feed),
		ExpectedCode: http.StatusConflict,
	})
}

func TestGetUserFeedItems(t *testing.T) {
	app := newTestApp()
	item := createItem(t, app, &db.Item{
		GUID: "http://google.com/item",
	})
	feed := createFeed(t, app, &db.Feed{
		URL:   "http://google.com",
		Items: []db.ID{item.ID},
	})

	req := newRequest("GET", fmt.Sprintf("/api/feeds/%s/items", feed.ID.Hex()), nil)
	var items []db.Item
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &items,
	})

	if len(items) != 1 {
		t.Errorf("items len mismatch: %d != %d", len(items), 1)
		if items[0].ID != feed.Items[0] {
			t.Errorf("item id mismatch: %s != %s", items[0].ID, feed.Items[0])
		}
	}
}

func TestGetUserFeedItemsWithModTime(t *testing.T) {
	app := newTestApp()
	item := createItem(t, app, &db.Item{
		GUID: "http://google.com/item",
	})
	feed := createFeed(t, app, &db.Feed{
		URL:   "http://google.com",
		Items: []db.ID{item.ID},
	})

	modTime := item.ModificationTime.Add(1 * time.Second)
	url := fmt.Sprintf("/api/feeds/%s/items?modified_since=%s", feed.ID.Hex(), modTime.Format(time.RFC3339))
	req := newRequest("GET", url, nil)
	var items []db.Item
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &items,
	})

	if len(items) != 0 {
		t.Errorf("items len mismatch: %d != %d", len(items), 0)
	}

	modTime = item.ModificationTime.Add(-2 * time.Second)
	url = fmt.Sprintf("/api/feeds/%s/items?modified_since=%s", feed.ID.Hex(), modTime.Format(time.RFC3339))
	req = newRequest("GET", url, nil)
	items = []db.Item{}
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &items,
	})

	if len(items) != 1 {
		t.Errorf("items len mismatch: %d != %d", len(items), 1)
	}
}

func TestGetFeedsUsers(t *testing.T) {
	app := newTestApp()
	feed := createFeed(t, app, &db.Feed{
		URL:      "http://google.com",
		ITunesID: 12345,
	})
	user := createUser(t, app, "chris", "whatever")

	user.FeedIDs = append(user.FeedIDs, feed.ID)
	if err := app.DB.Users.Update(user); err != nil {
		t.Fatal("Failed to update user:", err)
	}

	req := newRequest("GET", fmt.Sprintf("/api/feeds/%s/users", feed.ID.Hex()), nil)
	var out []db.User
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) == 1 {
		if out[0].ID != user.ID {
			t.Errorf("User ID mismatch: %s != %s", out[0].ID, user.ID)
		}
	} else {
		t.Errorf("Unexpected number of users: %d != %d", len(out), 1)
	}
}

func TestCreateFeedItem(t *testing.T) {
	app := newTestApp()
	feedID := createFeed(t, app, &db.Feed{URL: "http://google.com"}).ID

	item := db.Item{GUID: "http://google.com/items/1"}
	req := newRequest("POST", fmt.Sprintf("/api/feeds/%s/items", feedID.Hex()), &item)
	var out db.Item
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.GUID != item.GUID {
		t.Errorf("GUID mismatch: %s != %s", out.GUID, item.GUID)
	}

	feed, err := app.DB.Feeds.FeedByID(feedID)
	if err != nil {
		t.Fatal("Could not fetch feed")
	}

	if len(feed.Items) != 1 {
		t.Errorf("feed.Items len mismatch: %d != %d", len(feed.Items), 1)
	} else if feed.Items[0] != out.ID {
		t.Errorf("item.ID mismatch: %s != %s", feed.Items[0], out.ID)
	}
}

func TestGetFeedItem(t *testing.T) {
	app := newTestApp()
	item := createItem(t, app, &db.Item{GUID: "http://google.com/item"})
	feed := createFeed(t, app, &db.Feed{
		URL:   "http://google.com",
		Items: []db.ID{item.ID},
	})

	url := fmt.Sprintf("/api/feeds/%s/items/%s", feed.ID.Hex(), item.ID.Hex())
	req := newRequest("GET", url, item)
	var out db.Item
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.GUID != item.GUID {
		t.Errorf("GUID mismatch: %s != %s", out.GUID, item.GUID)
	}
}

func TestPutFeedItem(t *testing.T) {
	app := newTestApp()
	item := createItem(t, app, &db.Item{GUID: "http://google.com/item"})
	feed := createFeed(t, app, &db.Feed{
		URL:   "http://google.com",
		Items: []db.ID{item.ID},
	})

	item.URL = "http://google.com/item.mp3"

	url := fmt.Sprintf("/api/feeds/%s/items/%s", feed.ID.Hex(), item.ID.Hex())
	req := newRequest("PUT", url, item)
	var out db.Item
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.URL != item.URL {
		t.Errorf("URL mismatch: %s != %s", out.URL, item.URL)
	}
}

func TestGetJob(t *testing.T) {
	app := newTestApp()
	job := createJob(t, app, db.Job{
		KodaID: 1,
	})

	var out db.Job
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", fmt.Sprintf("/api/jobs/%s", job.ID.Hex()), nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.ID != job.ID {
		t.Errorf("id mismatch %s != %s", out.ID, job.ID)
	}
}

func TestCreateJob(t *testing.T) {
	app := newTestApp()

	in := db.Job{
		Queue:    "queue",
		Priority: 100,
		Payload:  map[string]string{"data": "stuff"},
	}

	var out db.Job
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("POST", "/api/jobs", &in),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if out.ID == emptyID {
		t.Error("id is invalid")
	}

	if out.KodaID == 0 {
		t.Error("KodaID was not set")
	}
}

func TestGetJobs(t *testing.T) {
	app := newTestApp()
	job := createJob(t, app, db.Job{
		KodaID: 1,
	})

	var out []db.Job
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/jobs", nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) != 1 {
		t.Fatalf("output len mismatch: %d != 1", len(out))
		return
	}

	if out[0].ID != job.ID {
		t.Error("unexpected job id")
	}
}

func TestGetJobs_Filter(t *testing.T) {
	app := newTestApp()
	createJob(t, app, db.Job{
		KodaID: 1,
		Queue:  "some-queue",
		State:  "done",
	})

	var out []db.Job
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/jobs?queue=some-queue", nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) != 1 {
		t.Error("job not found")
	}

	out = make([]db.Job, 0)
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/jobs?queue=fake-queue", nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) != 0 {
		t.Errorf("job count mismatch: %d != 0", len(out))
	}

	out = make([]db.Job, 0)
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/jobs?state=done", nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) != 1 {
		t.Error("job not found")
	}

	out = make([]db.Job, 0)
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      newRequest("GET", "/api/jobs?queue=working", nil),
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) != 0 {
		t.Errorf("job count mismatch: %d != 0", len(out))
	}
}
