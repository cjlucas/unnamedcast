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

	"gopkg.in/mgo.v2/bson"

	"github.com/cjlucas/unnamedcast/server/db"
	"github.com/gin-gonic/gin"
)

var emptyObjectID bson.ObjectId

func init() {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter, _ = os.Open(os.DevNull)
}

func newTestApp() *App {
	dbURL := os.Getenv("DB_URL")

	// Ensure a clean DB
	db, err := db.New(dbURL)
	if err := db.Drop(); err != nil {
		panic(err)
	}

	app, err := NewApp(dbURL)
	if err != nil {
		panic(err)
	}

	return app
}

func createFeed(t *testing.T, app *App, feed *db.Feed) *db.Feed {
	if err := app.DB.CreateFeed(feed); err != nil {
		t.Fatal("Failed to create feed")
	}
	return feed
}

func createItem(t *testing.T, app *App, item *db.Item) *db.Item {
	if err := app.DB.CreateItem(item); err != nil {
		t.Fatal("Failed to create item")
	}
	return item
}

func createUser(t *testing.T, app *App, username, password string) *db.User {
	user, err := app.DB.CreateUser(username, password)
	if err != nil {
		t.Fatal("Failed to create user")
	}
	return user
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

	if user.ID == emptyObjectID {
		t.Error("user.ID is invalid")
	}

	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusInternalServerError,
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
	user.FeedIDs = append(user.FeedIDs, bson.NewObjectId())

	if err := app.DB.UpdateUser(user); err != nil {
		t.Fatal("Could not update user:", err)
	}

	var out []bson.ObjectId
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
		FeedID:   bson.NewObjectId(),
		ItemGUID: "http://google.com",
		Position: 5,
	})

	if err := app.DB.UpdateUser(user); err != nil {
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
		if out[0] != user.ItemStates[0] {
			t.Errorf("ID mismatch: %+v != %+v", out[0], user.ItemStates[0])
		}
	} else {
		t.Errorf("Unexpected # of feed IDs: %d != %d", len(out), len(user.ItemStates))
	}
}

func TestPutUserFeeds(t *testing.T) {
	app := newTestApp()
	user := createUser(t, app, "chris", "hithere")

	ids := []bson.ObjectId{bson.NewObjectId()}
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

func TestPutUserItemStates(t *testing.T) {
	app := newTestApp()
	user := createUser(t, app, "chris", "hithere")

	states := []db.ItemState{
		{
			FeedID:   bson.NewObjectId(),
			ItemGUID: "http://google.com",
			Position: 5,
		},
	}
	req := newRequest("PUT", fmt.Sprintf("/api/users/%s/states", user.ID.Hex()), states)
	var out []db.ItemState
	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
		ResponseBody: &out,
	})

	if len(out) == len(states) {
		outFeedID := out[0].FeedID
		inFeedID := states[0].FeedID
		if outFeedID != inFeedID {
			t.Errorf("Feed ID mismatch: %s != %s", outFeedID, inFeedID)
		}
	} else {
		t.Errorf("Unexpected # of feed IDs: %d != %d", len(out), len(states))
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

	if out.ID == emptyObjectID {
		t.Error("ID is invalid")
	}

	if out.URL != in.URL {
		t.Errorf("URL mismatch: %s != %s", out.URL, in.URL)
	}

	// No body given
	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("POST", "/api/feeds", nil),
		ExpectedCode: http.StatusBadRequest,
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
	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("GET", "/api/feeds", nil),
		ExpectedCode: http.StatusBadRequest,
	})
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

func TestGetFeedsUsers(t *testing.T) {
	app := newTestApp()
	feed := createFeed(t, app, &db.Feed{
		URL:      "http://google.com",
		ITunesID: 12345,
	})
	user := createUser(t, app, "chris", "whatever")

	user.FeedIDs = append(user.FeedIDs, feed.ID)
	if err := app.DB.UpdateUser(user); err != nil {
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

	feed, err := app.DB.FeedByID(feedID)
	if err != nil {
		t.Fatal("Could not fetch feed")
	}

	if len(feed.Items) != 1 {
		t.Errorf("feed.Items len mismatch: %d != %d", len(feed.Items), 1)
	} else if feed.Items[0] != out.ID {
		t.Errorf("item.ID mismatch: %s != %s", feed.Items[0], out.ID)
	}
}

func TestPutFeedItem(t *testing.T) {
	app := newTestApp()
	item := createItem(t, app, &db.Item{GUID: "http://google.com/item"})
	feed := createFeed(t, app, &db.Feed{
		URL:   "http://google.com",
		Items: []bson.ObjectId{item.ID},
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
