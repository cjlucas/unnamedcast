package db

import (
	"reflect"
	"testing"
)

type foo struct {
	A string
}

func TestCopyModel_NoIgnoredFields(t *testing.T) {
	f1 := foo{A: "Something"}
	f2 := foo{}

	if !CopyModel(&f2, &f1) {
		t.Errorf("Expected CopyModel to return true")
	}

	if f2.A != f1.A {
		t.Errorf("Mismatch: %s != %s", f2.A, f1.A)
	}
}

func TestCopyModel_IgnoredFields(t *testing.T) {
	f1 := foo{A: "Something"}
	f2 := foo{}

	if CopyModel(&f2, &f1, "A") {
		t.Errorf("Expected CopyModel to return false")
	}

	if f2.A != "" {
		t.Errorf("Mismatch: %s != \"\"", f2.A)
	}
}

func TestNewModelInfo(t *testing.T) {
	cases := []struct {
		In              interface{}
		ExpectedFeeds   []string
		ExpectedNameMap map[string]string
	}{
		{
			In: struct {
				A int    `json:"a" bson:"a"`
				B string `json:"b" bson:"b"`
				C string `json:"-"`
				D int
			}{},
			ExpectedFeeds: []string{"a", "b"},
			ExpectedNameMap: map[string]string{
				"a": "a",
				"b": "b",
			},
		},
	}

	for _, c := range cases {
		out := newModelInfo(c.In)
		if !reflect.DeepEqual(out.Fields, c.ExpectedFeeds) {
			t.Errorf("Fields mismatch %#v != %#v", out.Fields, c.ExpectedFeeds)
		}

		if !reflect.DeepEqual(out.APIToDBNameMap, c.ExpectedNameMap) {
			t.Errorf("Fields mismatch %#v != %#v", out.APIToDBNameMap, c.ExpectedNameMap)
		}
	}
}
