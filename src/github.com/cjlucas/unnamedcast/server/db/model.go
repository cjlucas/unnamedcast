package db

import (
	"fmt"
	"reflect"
	"strings"
)

type FieldInfo struct {
	JSONName      string
	JSONOmitEmpty bool

	BSONName      string
	BSONOmitEmpty bool

	IndexName   string
	IndexUnique bool
	IndexText   bool
}

func parseFieldTag(tag reflect.StructTag) FieldInfo {
	type option struct {
		Key  string
		Flag *bool
	}

	// Parse encoding/json styled tags where the format is the following:
	// [<name>][,<opt1>][,<opt2>]
	// Name can be omitted like so: ",opt1,opt2"
	parseTag := func(tagName string, name *string, opts []option) {
		info := tag.Get(tagName)
		if info == "-" || info == "" {
			return
		}

		split := strings.Split(info, ",")
		*name = split[0]
		if len(split) == 1 {
			return
		}

		for _, s := range split[1:] {
			for _, opt := range opts {
				if s == opt.Key {
					*opt.Flag = true
				}
			}
		}
	}

	tagInfo := FieldInfo{}

	parseTag("json", &tagInfo.JSONName, []option{
		{Key: "omitempty", Flag: &tagInfo.JSONOmitEmpty},
	})

	parseTag("bson", &tagInfo.BSONName, []option{
		{Key: "omitempty", Flag: &tagInfo.JSONOmitEmpty},
	})

	parseTag("index", &tagInfo.IndexName, []option{
		{Key: "unique", Flag: &tagInfo.IndexUnique},
		{Key: "text", Flag: &tagInfo.IndexText},
	})

	// If index name is omitted, default to bson name
	if tag.Get("index") != "" && tagInfo.IndexName == "" {
		tagInfo.IndexName = tagInfo.BSONName
	}

	return tagInfo
}

// ModelInfo describes the API-DB relationship.
// This struct is not designed to be initialized directly.
type ModelInfo struct {
	// Get Field names
	Fields []FieldInfo

	jsonNameMap map[string]int
	bsonNameMap map[string]int

	// Indexed colums? (could allow index creation to be moved to)
	// Add ability to delegate index creation/rebuilding/deleting to collection
	// Instead of in app setup
	Indexes map[string]Index
}

func (info *ModelInfo) addField(field FieldInfo) {
	info.Fields = append(info.Fields, field)
	info.jsonNameMap[field.JSONName] = len(info.Fields) - 1
	info.bsonNameMap[field.BSONName] = len(info.Fields) - 1
}

func (info *ModelInfo) lookupName(name string, nameMap map[string]int) (FieldInfo, bool) {
	if i, ok := nameMap[name]; ok {
		return info.Fields[i], true
	}
	return FieldInfo{}, false
}

func (info *ModelInfo) LookupAPIName(name string) (FieldInfo, bool) {
	return info.lookupName(name, info.jsonNameMap)
}

func (info *ModelInfo) LookupDBName(name string) (FieldInfo, bool) {
	return info.lookupName(name, info.bsonNameMap)
}

// Build a Model Info from a given struct
func newModelInfo(m interface{}) ModelInfo {
	info := ModelInfo{
		jsonNameMap: make(map[string]int),
		bsonNameMap: make(map[string]int),
		Indexes:     make(map[string]Index),
	}
	model := reflect.TypeOf(m)

	for i := 0; i < model.NumField(); i++ {
		f := model.Field(i)
		tag := parseFieldTag(f.Tag)
		if tag.JSONName == "" || tag.BSONName == "" {
			continue
		}

		info.addField(tag)

		if tag.IndexName != "" {
			if idx, ok := info.Indexes[tag.IndexName]; ok {
				idx.Key = append(idx.Key, tag.BSONName)
			} else {
				info.Indexes[tag.IndexName] = Index{
					Name:   tag.IndexName,
					Key:    []string{tag.BSONName},
					Unique: tag.IndexUnique,
				}
			}
		}
	}

	return info
}

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
