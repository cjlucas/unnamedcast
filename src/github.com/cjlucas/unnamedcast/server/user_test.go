package main

import (
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

func TestCreateUserNoParams(t *testing.T) {
	app := newTestApp()
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("POST", "/api/users", nil)
	app.g.ServeHTTP(w, r)

	if w.Code/100 != 4 {
		t.Fail()
	}
}
