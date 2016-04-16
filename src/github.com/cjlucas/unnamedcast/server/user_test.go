package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/cjlucas/unnamedcast/server/db"
)

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

func newRequest(method string, endpoint string, body io.Reader) *http.Request {
	r, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		panic(err)
	}
	return r
}

type endpointTestInfo struct {
	App          *App
	Request      *http.Request
	ExpectedCode int
}

func testEndpoint(t *testing.T, info endpointTestInfo) {
	if info.App == nil {
		info.App = newTestApp()
	}

	w := httptest.NewRecorder()
	info.App.g.ServeHTTP(w, info.Request)

	if w.Code != info.ExpectedCode {
		t.Errorf("Unexpected status code: %d != %d", w.Code, info.ExpectedCode)
	}
}

func TestCreateUserNoParams(t *testing.T) {
	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("POST", "/api/users", nil),
		ExpectedCode: http.StatusBadRequest,
	})
}

func TestCreateUserValidParams(t *testing.T) {
	testEndpoint(t, endpointTestInfo{
		Request:      newRequest("POST", "/api/users?username=chris&password=hi", nil),
		ExpectedCode: http.StatusOK,
	})
}

func TestCreateUserDuplicateUser(t *testing.T) {
	app := newTestApp()
	req := newRequest("POST", "/api/users?username=chris&password=hi", nil)

	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusOK,
	})

	testEndpoint(t, endpointTestInfo{
		App:          app,
		Request:      req,
		ExpectedCode: http.StatusInternalServerError,
	})
}
