package api

import (
	"net/url"
	"testing"
)

var baseURL = url.URL{Scheme: "http", Host: "localhost:8080"}

func TestUrlf(t *testing.T) {
	api := API{BaseURL: &baseURL}

	cases := []struct {
		actual   string
		expected string
	}{
		{
			api.urlf("/api/feeds/%d", 123),
			"http://localhost:8080/api/feeds/123",
		},
	}

	for _, c := range cases {
		if c.actual != c.expected {
			t.Errorf("%s != %s", c.actual, c.expected)
		}
	}
}
