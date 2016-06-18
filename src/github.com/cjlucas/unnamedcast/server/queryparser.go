package main

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
	for i := 0; i < v.NumField(); i++ {
		vf := v.Field(i)
		tf := t.Field(i)

		if vf.Kind() == reflect.Struct && tf.Anonymous {
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

func (info *QueryParamInfo) Parse(vals url.Values) (interface{}, error) {
	spec := reflect.New(reflect.TypeOf(info.spec)).Interface()
	v := reflect.ValueOf(spec).Elem()

	for i, f := range readFields(v) {
		p := info.Params[i]
		val := vals.Get(p.Name)
		if val == "" {
			if p.Required {
				return nil, fmt.Errorf("required param not found: %s", p.Name)
			}
			continue
		}

		switch f.V.Interface().(type) {
		case string:
			f.V.SetString(val)
		case int, int64:
			n, err := strconv.ParseInt(val, 10, 0)
			if err != nil {
				return nil, err
			}
			f.V.SetInt(n)
		case uint, uint64:
			n, err := strconv.ParseUint(val, 10, 0)
			if err != nil {
				return nil, err
			}
			f.V.SetUint(n)
		case time.Time:
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				return nil, err
			}
			f.V.Set(reflect.ValueOf(t))
		default:
			return nil, fmt.Errorf("unknown type for field: \"%s\"", f.F.Name)
		}
	}

	return spec, nil
}
