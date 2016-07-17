package endpoint

import (
	"errors"
	"net/http"
	"time"

	"github.com/cjlucas/unnamedcast/db"
	"github.com/cjlucas/unnamedcast/server/middleware"
	"github.com/gin-gonic/gin"
)

func validateFeed(feed *db.Feed) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(feed.Items) > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"reason": "items is a read-only property",
			})
			c.Abort()
		}
	}
}

type Interface interface {
	Bind() []gin.HandlerFunc
	Handle(c *gin.Context)
}

type GetFeeds struct {
	DB     *db.DB
	Query  db.Query
	Params struct {
		sortParams
		limitParams
		ITunesID int    `param:"itunes_id"`
		URL      string `param:"url"`
	}
}

func (e *GetFeeds) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.AddQuerySortInfo(e.DB.Feeds.ModelInfo, &e.Query, &e.Params, "modification_time"),
		middleware.AddQueryLimitInfo(&e.Query, &e.Params),
	}
}

func (e *GetFeeds) Handle(c *gin.Context) {
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

type CreateFeed struct {
	DB   *db.DB
	Feed db.Feed
}

func (e *CreateFeed) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.UnmarshalBody(&e.Feed),
		validateFeed(&e.Feed),
	}
}

func (e *CreateFeed) Handle(c *gin.Context) {
	switch err := e.DB.Feeds.Create(&e.Feed); {
	case db.IsDup(err):
		c.JSON(http.StatusConflict, gin.H{"reason": "duplicate url found"})
		c.Abort()
		return
	case err != nil:
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	out, err := e.DB.Feeds.FeedByID(e.Feed.ID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

type FetchFeed struct {
	DB     *db.DB
	FeedID db.ID
}

func (e *FetchFeed) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModelWithID(e.DB.Feeds.FindByID, "id", &e.FeedID),
	}
}

func (e *FetchFeed) Handle(c *gin.Context) {
	feed, err := e.DB.Feeds.FeedByID(e.FeedID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, feed)
}

type UpdateFeed struct {
	DB     *db.DB
	Feed   db.Feed
	FeedID db.ID
}

func (e *UpdateFeed) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.UnmarshalBody(&e.Feed),
		validateFeed(&e.Feed),
		middleware.RequireExistingModelWithID(e.DB.Feeds.FindByID, "id", &e.FeedID),
	}
}

func (e *UpdateFeed) Handle(c *gin.Context) {
	existingFeed, err := e.DB.Feeds.FeedByID(e.FeedID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	// Persist existing items
	e.Feed.Items = existingFeed.Items

	if err := e.DB.Feeds.Update(&e.Feed); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, &e.Feed)
}

type GetFeedItems struct {
	DB     *db.DB
	FeedID db.ID
	Query  db.Query

	Params struct {
		sortParams
		limitParams
		ModifiedSince time.Time `param:"modified_since"`
	}
}

func (e *GetFeedItems) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.AddQuerySortInfo(e.DB.Items.ModelInfo, &e.Query, &e.Params, "modification_time"),
		middleware.AddQueryLimitInfo(&e.Query, &e.Params),
		middleware.RequireExistingModelWithID(e.DB.Feeds.FindByID, "id", &e.FeedID),
	}
}

func (e *GetFeedItems) Handle(c *gin.Context) {
	feed, err := e.DB.Feeds.FeedByID(e.FeedID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	e.Query.Filter = db.M{
		"_id": db.M{"$in": feed.Items},
	}

	if !e.Params.ModifiedSince.IsZero() {
		e.Query.Filter["modification_time"] = db.M{"$gt": e.Params.ModifiedSince}
	}

	var items []db.Item
	if err := e.DB.Items.Find(&e.Query).All(&items); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, &items)
}

type GetFeedUsers struct {
	DB     *db.DB
	FeedID db.ID
}

func (e *GetFeedUsers) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModelWithID(e.DB.Feeds.FindByID, "id", &e.FeedID),
	}
}

func (e *GetFeedUsers) Handle(c *gin.Context) {
	cur := e.DB.Users.Find(&db.Query{
		Filter: db.M{
			"feed_ids": db.M{
				"$in": []db.ID{e.FeedID},
			},
		},
	})

	var users []db.User
	if err := cur.All(&users); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, &users)
}

type CreateFeedItem struct {
	DB     *db.DB
	Item   db.Item
	FeedID db.ID
}

func (e *CreateFeedItem) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModelWithID(e.DB.Feeds.FindByID, "id", &e.FeedID),
		middleware.UnmarshalBody(&e.Item),
	}
}

func (e *CreateFeedItem) Handle(c *gin.Context) {
	if err := e.DB.Items.Create(&e.Item); err != nil {
		if db.IsDup(err) {
			c.JSON(http.StatusConflict, gin.H{
				"reason": "duplicate id",
			})
		} else {
			c.AbortWithError(http.StatusBadRequest, err)
		}
		return
	}

	feed, err := e.DB.Feeds.FeedByID(e.FeedID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	feed.Items = append(feed.Items, e.Item.ID)

	if err := e.DB.Feeds.Update(feed); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, &e.Item)
}

type GetFeedItem struct {
	DB     *db.DB
	FeedID db.ID
	ItemID db.ID
}

func (e *GetFeedItem) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModelWithID(e.DB.Feeds.FindByID, "id", &e.FeedID),
		middleware.RequireExistingModelWithID(e.DB.Items.FindByID, "itemID", &e.ItemID),
	}
}

func (e *GetFeedItem) Handle(c *gin.Context) {
	// TODO: RequireExistingModelWithID can be modified to do this boilerplate as well
	// (Just pass a ref to the function, if the ref is non-nil, store it at the provided ref)
	feed, err := e.DB.Feeds.FeedByID(e.FeedID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if !feed.HasItemWithID(e.ItemID) {
		c.AbortWithError(http.StatusNotFound, errors.New("item does not belong to feed"))
		return
	}

	var item db.Item
	if err := e.DB.Items.FindByID(e.ItemID).One(&item); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, &item)
}

type UpdateFeedItem struct {
	DB     *db.DB
	FeedID db.ID
	ItemID db.ID
	Item   db.Item
}

func (e *UpdateFeedItem) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModelWithID(e.DB.Feeds.FindByID, "id", &e.FeedID),
		middleware.RequireExistingModelWithID(e.DB.Items.FindByID, "itemID", &e.ItemID),
		middleware.UnmarshalBody(&e.Item),
	}
}

func (e *UpdateFeedItem) Handle(c *gin.Context) {
	e.Item.ID = e.ItemID

	feed, err := e.DB.Feeds.FeedByID(e.FeedID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if !feed.HasItemWithID(e.Item.ID) {
		c.AbortWithError(http.StatusNotFound, errors.New("item does not belong to feed"))
		return
	}

	if err := e.DB.Items.Update(&e.Item); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, &e.Item)
}
