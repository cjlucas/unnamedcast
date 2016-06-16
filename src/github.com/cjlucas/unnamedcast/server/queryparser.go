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

func NewQueryParamInfo(spec interface{}) QueryParamInfo {
	info := QueryParamInfo{
		spec: spec,
	}

	v := reflect.ValueOf(spec)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		paramName := t.Field(i).Tag.Get("param")
		if paramName == "" {
			paramName = strings.ToLower(t.Field(i).Name)
		}
		info.Params = append(info.Params, QueryParam{
			Name: paramName,
		})
	}

	return info
}

func (info *QueryParamInfo) newSpec() interface{} {
	return reflect.New(reflect.TypeOf(info.spec)).Interface()
}

func (info *QueryParamInfo) Parse(vals url.Values) (interface{}, error) {
	spec := info.newSpec()
	v := reflect.ValueOf(spec).Elem()

	for i := 0; i < v.NumField(); i++ {
		vf := v.Field(i)
		param := info.Params[i]
		val := vals.Get(param.Name)
		if val == "" {
			continue
		}

		switch vf.Interface().(type) {
		case string:
			vf.SetString(val)
		case int, int64:
			n, err := strconv.ParseInt(val, 10, 0)
			if err != nil {
				return nil, err
			}
			vf.SetInt(n)
		case uint, uint64:
			n, err := strconv.ParseUint(val, 10, 0)
			if err != nil {
				return nil, err
			}
			vf.SetUint(n)
		case time.Time:
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				return nil, err
			}
			vf.Set(reflect.ValueOf(t))
		default:
			return nil, fmt.Errorf("unknown type for field: \"%s\"", param.Name)
		}
	}

	return spec, nil
}
