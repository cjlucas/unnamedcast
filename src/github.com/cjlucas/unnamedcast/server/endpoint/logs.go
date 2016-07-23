package endpoint

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/cjlucas/unnamedcast/db"
	"github.com/cjlucas/unnamedcast/server/middleware"
	"github.com/gin-gonic/gin"
)

type getLogsParams struct {
	sortParams
	limitParams
	Code string `param:"code"`
}

type GetLogs struct {
	DB     *db.DB
	Query  db.Query
	Params getLogsParams
}

func (e *GetLogs) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.AddQuerySortInfo(e.DB.Jobs.ModelInfo, &e.Query, &e.Params, "creation_time", "execution_time"),
		middleware.AddQueryLimitInfo(&e.Query, &e.Params),
		e.buildQuery,
	}
}

func (e *GetLogs) buildQuery(c *gin.Context) {
	if e.Query.Filter == nil {
		e.Query.Filter = make(db.M)
	}

	var split []int
	for _, s := range strings.Split(e.Params.Code, "-") {
		n, err := strconv.Atoi(s)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("bad status code: %s", s))
		}
		split = append(split, n)
	}

	switch l := len(split); {
	case l == 1:
		e.Query.Filter["status_code"] = split[0]
	case l > 1:
		e.Query.Filter["status_code"] = db.M{
			"$gte": split[0],
			"$lte": split[1],
		}
	}
}

func (e *GetLogs) Handle(c *gin.Context) {
	var logs []db.Log
	if err := e.DB.Logs.Find(&e.Query).All(&logs); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, logs)
}
