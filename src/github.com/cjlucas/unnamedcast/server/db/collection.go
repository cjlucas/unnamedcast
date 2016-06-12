package db

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type collection struct {
	c         *mgo.Collection
	ModelInfo ModelInfo
}

func (c collection) Find(q *Query) Cursor {
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

func (c collection) FindByID(id bson.ObjectId) Cursor {
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

func (c collection) pipeline(pipeline interface{}) *Pipe {
	return &Pipe{
		s: c.c.Database.Session,
		p: c.c.Pipe(pipeline),
	}
}
