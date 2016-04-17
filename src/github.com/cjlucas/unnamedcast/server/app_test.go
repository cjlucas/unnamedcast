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

func TestCreateUserNoParams(t *testing.T) {
	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("POST", "/api/users", nil),
		ExpectedCode: http.StatusBadRequest,
	})
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
}

func TestCreateFeed(t *testing.T) {
	in := db.Feed{
		URL: "http://google.com",
		Items: []db.Item{
			{
				GUID: "1",
				Link: "http://google.com/link",
			},
			{
				GUID: "2",
				Link: "http://google.com/link2",
			},
		},
	}
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

	if len(out.Items) == len(in.Items) {
		for i := range out.Items {
			outGUID := out.Items[i].GUID
			inGUID := out.Items[i].GUID
			if outGUID != inGUID {
				t.Errorf("GUID mismatch (item #%d): %s != %s", i, outGUID, inGUID)
			}
		}
	} else {
		t.Errorf("Unexpected # of items: %d != %d", len(out.Items), len(in.Items))
	}

	// No body given
	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("POST", "/api/feeds", nil),
		ExpectedCode: http.StatusBadRequest,
	})
}

func TestGetFeedWithoutParams(t *testing.T) {
	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("GET", "/api/feeds", nil),
		ExpectedCode: http.StatusBadRequest,
	})
}

func TestGetFeedByURL(t *testing.T) {
	app := newTestApp()
	feed := &db.Feed{
		URL: "http://google.com",
	}

	if err := app.DB.CreateFeed(feed); err != nil {
		t.Fatal("could not create feed", err)
	}

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
	feed := &db.Feed{
		URL:      "http://google.com",
		ITunesID: 12345,
	}

	if err := app.DB.CreateFeed(feed); err != nil {
		t.Fatal("could not create feed:", err)
	}

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
	feed := &db.Feed{
		URL:      "http://google.com",
		ITunesID: 12345,
	}

	if err := app.DB.CreateFeed(feed); err != nil {
		t.Fatal("could not create feed:", err)
	}

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
	feed := &db.Feed{
		URL:      "http://google.com",
		ITunesID: 12345,
	}

	if err := app.DB.CreateFeed(feed); err != nil {
		t.Fatal("could not create feed:", err)
	}

	user, err := app.DB.CreateUser("chris", "whatever")
	if err != nil {
		t.Fatal("Could not create user:", err)
	}

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
