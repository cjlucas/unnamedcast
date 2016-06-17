package db

import (
	"reflect"
	"testing"

	"gopkg.in/mgo.v2/bson"
)

func TestFilterCond(t *testing.T) {
	cases := []struct {
		Collection collection
		Query      Query
		Var        string
		Expected   bson.M
	}{
		{
			Collection: collection{
				ModelInfo: newModelInfo(struct {
					Field string `json:"field" bson:"field"`
				}{}),
			},
			Query: Query{
				Filter: bson.M{
					"field": bson.M{"$ge": 5},
				},
			},
			Var: "states",
			Expected: bson.M{
				"$ge": []interface{}{"$$states.field", 5},
			},
		},
	}

	for _, c := range cases {
		out := c.Collection.filterCond(c.Query, c.Var)
		if !reflect.DeepEqual(out, c.Expected) {
			t.Errorf("cond mismatch %s != %s", out, c.Expected)
		}
	}
}
