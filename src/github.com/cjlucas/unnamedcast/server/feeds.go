package main

import (
	"net/http"

	"github.com/cjlucas/unnamedcast/db"
	"github.com/gin-gonic/gin"
)

type Endpoint interface {
	Bind() []gin.HandlerFunc
	Handle(c *gin.Context)
}

type GetFeedsQueryEndpoint struct {
	// TODO: add support for injecting individual collections
	DB     *db.DB
	Query  db.Query
	Params struct {
		sortParams
		limitParams
		ITunesID int    `param:"itunes_id"`
		URL      string `param:"url"`
	}
}

func (e *GetFeedsQueryEndpoint) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		parseSortParamsNew(e.DB.Feeds.ModelInfo, &e.Query, "modification_time"),
		parseLimitParamsNew(&e.Query),
	}
}

func (e *GetFeedsQueryEndpoint) Handle(c *gin.Context) {
	if e.Params.URL != "" {
		e.Query.Filter = db.M{"url": e.Params.URL}
	} else if e.Params.ITunesID != 0 {
		e.Query.Filter = db.M{"itunes_id": e.Params.ITunesID}
	}

	if e.Query.Filter == nil {
		var feeds []db.Feed
		if err := e.DB.Feeds.Find(&e.Query).All(&feeds); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, feeds)
		}
		return
	}

	// TODO: use a switch here
	var feed db.Feed
	if err := e.DB.Feeds.Find(&e.Query).One(&feed); err != nil {
		if err == db.ErrNotFound {
			c.AbortWithStatus(http.StatusNotFound)
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	c.JSON(http.StatusOK, &feed)
}
