package db

import (
	"reflect"
	"testing"
)

func TestFilterCond(t *testing.T) {
	cases := []struct {
		Collection collection
		Query      Query
		Var        string
		Expected   M
	}{
		{
			Collection: collection{
				ModelInfo: newModelInfo(struct {
					Field string `json:"field" bson:"field"`
				}{}),
			},
			Query: Query{
				Filter: M{
					"field": M{"$ge": 5},
				},
			},
			Var: "states",
			Expected: M{
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
