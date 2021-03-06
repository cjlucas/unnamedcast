package queryparser

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type QueryParam struct {
	Name     string
	Required bool
}

type QueryParamInfo struct {
	spec   interface{}
	Params []QueryParam
}

type rawField struct {
	V reflect.Value
	F reflect.StructField
}

func readFields(v reflect.Value) []rawField {
	var fields []rawField

	t := v.Type()
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		v = v.Elem()
		t = v.Type()
	}

	for i := 0; i < v.NumField(); i++ {
		vf := v.Field(i)
		tf := t.Field(i)

		// HACK: hard coding struct types that we use in Parse
		// We don't want to recurse down into structs that arent ours
		if vf.Kind() == reflect.Struct && tf.Type != reflect.TypeOf(time.Time{}) {
			fields = append(fields, readFields(vf)...)
			continue
		}

		fields = append(fields, rawField{
			V: vf,
			F: tf,
		})
	}

	return fields
}

func NewQueryParamInfo(spec interface{}) QueryParamInfo {
	fields := readFields(reflect.ValueOf(spec))
	info := QueryParamInfo{
		spec:   spec,
		Params: make([]QueryParam, len(fields)),
	}

	for i, f := range fields {
		opts := strings.Split(f.F.Tag.Get("param"), ",")
		p := QueryParam{}

		if len(opts) > 0 && opts[0] != "" {
			p.Name = opts[0]
		} else {
			p.Name = strings.ToLower(f.F.Name)
		}

		if len(opts) > 1 {
			for _, opt := range opts[1:] {
				switch opt {
				case "require":
					p.Required = true
				default:
					panic(fmt.Errorf("unexpected option: %s", opt))
				}
			}
		}

		info.Params[i] = p
	}

	return info
}

func (info *QueryParamInfo) Parse(instance interface{}, vals url.Values) error {
	v := reflect.ValueOf(instance).Elem()

	for i, f := range readFields(v) {
		p := info.Params[i]
		val := vals.Get(p.Name)
		if val == "" {
			if p.Required {
				return fmt.Errorf("required param not found: %s", p.Name)
			}
			continue
		}

		switch f.V.Interface().(type) {
		case string:
			f.V.SetString(val)
		case int, int64:
			n, err := strconv.ParseInt(val, 10, 0)
			if err != nil {
				return err
			}
			f.V.SetInt(n)
		case uint, uint64:
			n, err := strconv.ParseUint(val, 10, 0)
			if err != nil {
				return err
			}
			f.V.SetUint(n)
		case time.Time:
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				return err
			}
			f.V.Set(reflect.ValueOf(t))
		default:
			return fmt.Errorf("unknown type for field: \"%s\"", f.F.Name)
		}
	}

	return nil
}
