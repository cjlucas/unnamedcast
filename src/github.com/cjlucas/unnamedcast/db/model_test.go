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
		ExpectedFields  []FieldInfo
		ExpectedIndexes map[string]Index
	}{
		{
			In: struct {
				A int    `json:"a" bson:"a"`
				B string `json:"b" bson:"b"`
				C string `json:"-"`
				D int
			}{},
			ExpectedFields: []FieldInfo{
				{
					JSONName: "a",
					BSONName: "a",
				},
				{
					JSONName: "b",
					BSONName: "b",
				},
			},
		},
		{
			In: struct {
				A int `json:"a" bson:"a" index:"a,unique"`
				B int `json:"b" bson:"b" index:",unique"`
			}{},
			ExpectedFields: []FieldInfo{
				{
					JSONName:    "a",
					BSONName:    "a",
					IndexName:   "a",
					IndexUnique: true,
				},
				{
					JSONName:    "b",
					BSONName:    "b",
					IndexName:   "b",
					IndexUnique: true,
				},
			},
			ExpectedIndexes: map[string]Index{
				"a": {
					Name:   "a",
					Key:    []string{"a"},
					Unique: true,
				},
				"b": {
					Name:   "b",
					Key:    []string{"b"},
					Unique: true,
				},
			},
		},
		{
			In: struct {
				A int    `json:"a" bson:"a" index:"idx"`
				B string `json:"b" bson:"b" index:"idx"`
			}{},
			ExpectedIndexes: map[string]Index{
				"idx": {
					Name:   "idx",
					Key:    []string{"a", "b"},
					Unique: false,
				},
			},
		},
	}

	for _, c := range cases {
		out := newModelInfo(c.In)
		if len(c.ExpectedFields) > 0 && !reflect.DeepEqual(out.Fields, c.ExpectedFields) {
			t.Errorf("Fields mismatch %#v != %#v", out.Fields, c.ExpectedFields)
		}
		if len(c.ExpectedIndexes) > 0 && !reflect.DeepEqual(out.Indexes, c.ExpectedIndexes) {
			t.Errorf("Indexes mismatch %#v != %#v", out.Indexes, c.ExpectedIndexes)
		}
	}
}

func TestModelInfo_LookupAPIName(t *testing.T) {
	info := newModelInfo(struct {
		A int `json:"a" bson:"a"`
	}{})
	if _, ok := info.LookupAPIName("a"); !ok {
		t.Error("field not found")
	}
}

func TestModelInfo_LookupDBName(t *testing.T) {
	info := newModelInfo(struct {
		A int `json:"a" bson:"a"`
	}{})
	if _, ok := info.LookupDBName("a"); !ok {
		t.Error("field not found")
	}
}
