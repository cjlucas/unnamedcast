package main

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func parseQueryParams(vals url.Values, spec interface{}) error {
	v := reflect.ValueOf(spec).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		vf := v.Field(i)
		tf := t.Field(i)
		opts := tf.Tag.Get("param")

		fieldName := strings.ToLower(t.Field(i).Name)
		if opts != "" {
			fieldName = opts
		}

		val := vals.Get(fieldName)
		if val == "" {
			continue
		}

		switch vf.Interface().(type) {
		case string:
			vf.SetString(val)
		case int, int64:
			n, err := strconv.ParseInt(val, 10, 0)
			if err != nil {
				return err
			}
			vf.SetInt(n)
		case uint, uint64:
			n, err := strconv.ParseUint(val, 10, 0)
			if err != nil {
				return err
			}
			vf.SetUint(n)
		case time.Time:
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				return err
			}
			vf.Set(reflect.ValueOf(t))
		default:
			return fmt.Errorf("unknown type for field: \"%s\"", fieldName)
		}
	}

	return nil
}
