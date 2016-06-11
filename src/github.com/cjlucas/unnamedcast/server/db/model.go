package db

import (
	"fmt"
	"reflect"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// CopyModel copies all the fields from m2 into m1 excluding any fields
// specified by ignoredFields. A boolean is returned representing whether
// any data has changed in m1 as a result of the copy.
func CopyModel(m1, m2 interface{}, ignoredFields ...string) bool {
	isIgnoredField := func(name string) bool {
		for _, s := range ignoredFields {
			if s == name {
				return true
			}
		}
		return false
	}

	// Get the underlying struct as a Value
	s1 := reflect.ValueOf(m1).Elem()
	s2 := reflect.ValueOf(m2).Elem()
	t := s1.Type()

	for _, v := range []reflect.Value{s1, s2} {
		if v.Kind() == reflect.Ptr {
			panic("CopyModel was called with a double pointer type")
		}
	}

	changed := false
	for i := 0; i < s1.NumField(); i++ {
		f := s1.Field(i)
		f2 := s2.Field(i)
		fieldName := t.Field(i).Name

		// CanInterface tells us if a field is unexported
		if !f.CanInterface() || isIgnoredField(fieldName) {
			continue
		}

		if !reflect.DeepEqual(f.Interface(), f2.Interface()) {
			changed = true
			if !f.CanSet() {
				panic(fmt.Sprintf("Cannot set field %s", fieldName))
			}
			f.Set(f2)
		}
	}

	return changed
}

type collection struct {
	DB *DB
	c  *mgo.Collection
}

func (c collection) Find(q interface{}) Query {
	return &query{
		s: c.c.Database.Session,
		q: c.c.Find(q),
	}
}

func (c collection) FindByID(id bson.ObjectId) Query {
	return &query{
		s: c.c.Database.Session,
		q: c.c.FindId(id),
	}
}

func (c collection) EnsureIndex(idx Index) error {
	return c.c.EnsureIndex(mgoIndexForIndex(idx))
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
