package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

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
	FindByID(id db.ID) db.Cursor
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

func LogErrors(logs db.LogCollection) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		c.Request.Body.Close()

		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, errors.New("could not read body"))
			return
		}

		c.Request.Body = ioutil.NopCloser(bytes.NewReader(body))
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		logs.Create(&db.Log{
			Method:        c.Request.Method,
			RequestHeader: c.Request.Header,
			RequestBody:   string(body),
			URL:           c.Request.URL.String(),
			StatusCode:    c.Writer.Status(),
			RemoteAddr:    c.ClientIP(),
			Errors:        c.Errors,
		})
	}
}
