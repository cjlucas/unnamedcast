package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/cjlucas/unnamedcast/db"
	"github.com/cjlucas/unnamedcast/server/queryparser"
	"github.com/gin-gonic/gin"
)

func UnmarshalBody(data interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.BindJSON(data); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}
	}
}

type Collection interface {
	FindByID(id db.ID) *db.Result
}

type RequireExistingModelOpts struct {
	Collection Collection
	BoundName  string

	ID     *db.ID
	Result interface{}
}

func RequireExistingModel(opts *RequireExistingModelOpts) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := db.IDFromString(c.Param(opts.BoundName))
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		if opts.ID != nil {
			*opts.ID = id
		}

		cur := opts.Collection.FindByID(id)
		n, err := cur.Count()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		if n == 0 {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		if opts.Result != nil {
			if err := cur.One(opts.Result); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
			}
		}
	}
}

func ParseQueryParams(info *queryparser.QueryParamInfo, params interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := info.Parse(params, c.Request.URL.Query()); err != nil {
			fmt.Println(err)
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("failed to parse query params: %s", err))
			return
		}
	}
}

func LogRequest(logs db.LogCollection) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		c.Request.Body.Close()

		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, errors.New("could not read body"))
			return
		}

		c.Request.Body = ioutil.NopCloser(bytes.NewReader(body))

		start := time.Now()
		c.Next()
		executionTime := float32(time.Now().Sub(start)) / float32(time.Second)

		logs.Create(&db.Log{
			Method:        c.Request.Method,
			RequestHeader: c.Request.Header,
			RequestBody:   string(body),
			Endpoint:      c.Request.URL.Path,
			Query:         c.Request.URL.RawQuery,
			StatusCode:    c.Writer.Status(),
			RemoteAddr:    c.ClientIP(),
			ExecutionTime: executionTime,
			Errors:        c.Errors.Errors(),
		})
	}
}
