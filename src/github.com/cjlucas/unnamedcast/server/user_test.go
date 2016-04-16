package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func newTestApp() *App {
	app, err := NewApp(os.Getenv("DB_URL"))
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
		defer info.App.DB.Drop()
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
