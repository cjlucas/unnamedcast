package middleware

import (
	"fmt"
	"net/http"

	"github.com/cjlucas/unnamedcast/db"
	"github.com/gin-gonic/gin"
)

type SortParams interface {
	SortField() string
	Desc() bool
}

// TODO: get rid of sortable fields and store sorting information in model tag
func AddQuerySortInfo(mi db.ModelInfo, query *db.Query, params SortParams, sortableFields ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if params == nil {
			return
		}

		field := params.SortField()
		if field == "" {
			return
		}

		found := false
		for _, f := range sortableFields {
			if field == f {
				found = true
				break
			}
		}

		if !found {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("\"%s\" is not a sortable field", field))
			return
		}

		info, ok := mi.LookupAPIName(field)
		if !ok {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("\"%s\" is not a known field", field))
			return
		}
		field = info.BSONName

		query.SortField = field
		query.SortDesc = params.Desc()
	}
}

type LimitParams interface {
	Limit() int
}

func AddQueryLimitInfo(query *db.Query, params LimitParams) gin.HandlerFunc {
	return func(c *gin.Context) {
		if params != nil {
			query.Limit = params.Limit()
		}
	}
}
