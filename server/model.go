package main

import (
	"fmt"
	"reflect"
)

// CopyModel copies all the fields from m2 into m1 excluding any fields
// specified by ignoredFields. A boolean is returned representing whether
// any data has changed in m1 as a result of the copy. If either m1 or m2
// is nil, No merge will be attempted and false will be returned
func CopyModel(m1, m2 interface{}, ignoredFields ...string) bool {
	changed := false

	if m1 == nil || m2 == nil {
		return false
	}

	isIgnoredField := func(name string) bool {
		for _, s := range ignoredFields {
			if s == name {
				return true
			}
		}
		return false
	}

	s1 := reflect.ValueOf(m1).Elem()
	s2 := reflect.ValueOf(m2).Elem()
	t := s1.Type()

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
