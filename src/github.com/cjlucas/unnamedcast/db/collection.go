package db

import (
	"fmt"

	"gopkg.in/mgo.v2"
)

type collection struct {
	// c can be nil if collection is really a subcollection in the db
	c         *mgo.Collection
	ModelInfo ModelInfo
}

func (c collection) Find(q *Query) Result {
	if q == nil {
		return &query{
			s: c.c.Database.Session,
			q: c.c.Find(nil),
		}
	}

	cur := &query{
		s: c.c.Database.Session,
		q: c.c.Find(q.Filter),
	}

	sel := make(map[string]int)
	for _, s := range q.SelectedFields {
		sel[s] = 1
	}
	for _, s := range q.OmittedFields {
		sel[s] = -1
	}
	if len(sel) > 0 {
		cur.Select(sel)
	}

	if q.SortField != "" {
		sortField := q.SortField
		if q.SortDesc {
			sortField = "-" + sortField
		}
		cur.Sort(sortField)
	}

	if q.Limit > 0 {
		cur.Limit(q.Limit)
	}

	return cur
}

func (c collection) FindByID(id ID) Result {
	return &query{
		s: c.c.Database.Session,
		q: c.c.FindId(id),
	}
}

func (c collection) createIndex(index Index, force bool) error {
	if force {
		c.c.DropIndexName(index.Name)
	}

	return c.c.EnsureIndex(mgoIndexForIndex(index))
}

// CreateIndexes creates all indexes in ModelInfo. If force is set, the existing
// index will be dropped prior to recreating the index. An error will
// be returned if an index already exists only if it's options differ.
// This is intentional as we don't want to recreate indexes on the fly.
func (c collection) CreateIndexes(force bool) error {
	for _, idx := range c.ModelInfo.Indexes {
		if err := c.createIndex(idx, force); err != nil {
			return err
		}
	}

	return nil
}

func (c collection) insert(model interface{}) error {
	return c.c.Insert(model)
}

// filterCond builds the "cond" value of a $filter operation from a Query.
// More specifically, it converts the specified query.Filter into the expression
// format required by the aggregation. varName is the variable name specified
// by the "as" option.
//
// Example: {"field": {"$ge": 5} would be transformed into {"$ge": ["field", 5]}
func (c collection) filterCond(query Query, varName string) M {
	// This code only handles the trivial case, as that is all that's needed currently
	cond := make(M)

	for field, expr := range query.Filter {
		if _, ok := c.ModelInfo.LookupDBName(field); !ok {
			panic(fmt.Errorf("\"%s\" is not a valid field", field))
		}

		if expr, ok := expr.(M); ok {
			for op, val := range expr {
				cond[op] = []interface{}{
					fmt.Sprintf("$$%s.%s", varName, field),
					val,
				}
			}
		} else {
			panic("unexpected expression")
		}

	}

	return cond
}

func (c collection) pipeline(pipeline interface{}) *Pipe {
	return &Pipe{
		s: c.c.Database.Session,
		p: c.c.Pipe(pipeline),
	}
}
