package main

import (
	"flag"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

var doIntegrationTests = flag.Bool("integration", false, "perform integration tests in addition to unit tests")

func TestCreateUserNoParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", CreateUser)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, r)

	if w.Code/100 != 4 {
		t.Fail()
	}
}

func TestDoSomethingIntegrationRelated(t *testing.T) {
	t.Fail()
}
