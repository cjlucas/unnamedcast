package queryparser

import (
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestNewQueryParamInfo(t *testing.T) {
	cases := []struct {
		In             interface{}
		ExpectedParams []QueryParam
	}{
		{
			In: struct {
				A int
				B string `param:"foo"`
				C string `param:"c,require"`
				D string `param:",require"`
			}{},
			ExpectedParams: []QueryParam{
				{
					Name:     "a",
					Required: false,
				},
				{
					Name:     "foo",
					Required: false,
				},
				{
					Name:     "c",
					Required: true,
				},
				{
					Name:     "d",
					Required: true,
				},
			},
		},
		{
			In: struct {
				A time.Time `param:"foo"`
			}{},
			ExpectedParams: []QueryParam{
				{
					Name:     "foo",
					Required: false,
				},
			},
		},
		{
			In: struct {
				Params struct {
					A string `param:"foo"`
				}
			}{},
			ExpectedParams: []QueryParam{
				{
					Name:     "foo",
					Required: false,
				},
			},
		},
	}

	for _, c := range cases {
		out := NewQueryParamInfo(c.In)
		out = NewQueryParamInfo(&c.In)
		if len(out.Params) != len(c.ExpectedParams) {
			t.Fatalf("Params length mismatch: %d != %d", len(out.Params), len(c.ExpectedParams))
		}

		for i, p1 := range out.Params {
			p2 := c.ExpectedParams[i]
			if p1 != p2 {
				t.Errorf("Param mismatch: %#v != %#v\n", p1, p2)
			}
		}
	}
}

func TestParseQueryParams(t *testing.T) {
	type embed struct {
		A string
		B string
	}

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
			Query: "custom=val",
			Expected: struct {
				X string `param:"custom"`
			}{X: "val"},
		},
		{
			Query:       "x=notanum",
			ShouldError: true,
			Expected: struct {
				X int
			}{},
		},
		{
			Query:       "a=something",
			ShouldError: true,
			Expected: struct {
				X int `param:",require"`
			}{},
		},
		{
			Query: "a=a&b=b&c=c",
			Expected: struct {
				embed
				C string
			}{
				embed: embed{
					A: "a",
					B: "b",
				},
				C: "c",
			},
		},
	}

	for _, c := range cases {
		vals, err := url.ParseQuery(c.Query)
		if err != nil {
			t.Fatal("Failed to parse query:", c.Query)
		}

		typ := reflect.TypeOf(c.Expected)
		info := NewQueryParamInfo(reflect.New(typ).Elem().Interface())

		out, err := info.Parse(vals)
		if err != nil && c.ShouldError {
			continue
		} else if err != nil && !c.ShouldError {
			t.Error("parseQueryParams failed:", err)
		} else if err == nil && c.ShouldError {
			t.Error("parseQueryParams unexpected succeeded")
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
	type spec struct {
		A int
		B string
		C time.Time
	}

	info := NewQueryParamInfo(spec{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		info.Parse(vals)
	}
}
