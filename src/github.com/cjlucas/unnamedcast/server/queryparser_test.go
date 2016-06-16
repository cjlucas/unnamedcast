package main

import (
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestParseQueryParams(t *testing.T) {
	cases := []struct {
		// URL encoded query
		Query       string
		ShouldError bool
		Expected    interface{}
	}{
		{
			Query: "a=5&b=hithere&c=2006-01-02T15:04:05.990Z",
			Expected: struct {
				A int
				B string
				C time.Time
			}{
				A: 5,
				B: "hithere",
				C: time.Date(2006, time.January, 2, 15, 4, 5, int(990*time.Millisecond), time.UTC),
			},
		},
		{
			Query:       "x=notanum",
			ShouldError: true,
			Expected: struct {
				X int
			}{},
		},
	}

	for _, c := range cases {
		vals, err := url.ParseQuery(c.Query)
		if err != nil {
			t.Fatal("Failed to parse query:", c.Query)
		}

		typ := reflect.TypeOf(c.Expected)
		out := reflect.New(typ).Interface() // out is a pointer type

		if err := parseQueryParams(vals, out); err != nil && !c.ShouldError {
			t.Error("parseQueryParams failed:", err)
		}

		// Use the concrete type of out when comparing against c.Expected
		actual := reflect.ValueOf(out).Elem().Interface()
		if !reflect.DeepEqual(actual, c.Expected) {
			t.Errorf("%#v != %#v", actual, c.Expected)
		}
	}
}

func BenchmarkParseQueryParams(b *testing.B) {
	vals, _ := url.ParseQuery("a=5&b=hithere&c=2006-01-02T15:04:05.990Z")
	spec := struct {
		A int
		B string
		C time.Time
	}{
		A: 5,
		B: "hithere",
		C: time.Date(2006, time.January, 2, 15, 4, 5, int(990*time.Millisecond), time.UTC),
	}

	for i := 0; i < b.N; i++ {
		parseQueryParams(vals, &spec)
	}
}
