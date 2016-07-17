package endpoint

import (
	"net/http"
	"strings"
	"time"

	"github.com/cjlucas/unnamedcast/db"
	"github.com/cjlucas/unnamedcast/server/middleware"
	"github.com/gin-gonic/gin"
)

type GetUsers struct {
	DB     *db.DB
	Query  db.Query
	Params struct {
		sortParams
		limitParams
		ModifiedSince time.Time `param:"modified_since"`
	}
}

func (e *GetUsers) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.AddQuerySortInfo(e.DB.Users.ModelInfo, &e.Query, &e.Params, "modification_time"),
		middleware.AddQueryLimitInfo(&e.Query, &e.Params),
	}
}

func (e *GetUsers) Handle(c *gin.Context) {
	var users []db.User
	if err := e.DB.Users.Find(&e.Query).All(&users); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, users)
}

type CreateUser struct {
	DB     *db.DB
	Params struct {
		Username string `param:",require"`
		Password string `param:",require"`
	}
}

func (e *CreateUser) Bind() []gin.HandlerFunc {
	return nil
}

func (e *CreateUser) Handle(c *gin.Context) {
	for _, s := range []*string{&e.Params.Username, &e.Params.Password} {
		*s = strings.TrimSpace(*s)
	}

	switch user, err := e.DB.Users.Create(e.Params.Username, e.Params.Password); {
	case err == nil:
		c.JSON(http.StatusOK, user)
	case db.IsDup(err):
		c.JSON(http.StatusConflict, gin.H{
			"reason": "user already exists",
		})
		c.Abort()
	default:
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

type GetUser struct {
	DB   *db.DB
	User db.User
}

func (e *GetUser) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModel(&middleware.RequireExistingModelOpts{
			Collection: e.DB.Users,
			BoundName:  "id",
			Result:     &e.User,
		}),
	}
}

func (e *GetUser) Handle(c *gin.Context) {
	c.JSON(http.StatusOK, &e.User)
}

type GetUserFeeds struct {
	DB   *db.DB
	User db.User
}

func (e *GetUserFeeds) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModel(&middleware.RequireExistingModelOpts{
			Collection: e.DB.Users,
			BoundName:  "id",
			Result:     &e.User,
		}),
	}
}

func (e *GetUserFeeds) Handle(c *gin.Context) {
	c.JSON(http.StatusOK, &e.User.FeedIDs)
}

type UpdateUserFeeds struct {
	DB      *db.DB
	User    db.User
	FeedIDs []db.ID
}

func (e *UpdateUserFeeds) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModel(&middleware.RequireExistingModelOpts{
			Collection: e.DB.Users,
			BoundName:  "id",
			Result:     &e.User,
		}),
		middleware.UnmarshalBody(&e.FeedIDs),
	}
}

func (e *UpdateUserFeeds) Handle(c *gin.Context) {
	// TODO: Add an UpdateFeeds method to DB.Users. There's no need to fetch
	// The entire user
	e.User.FeedIDs = e.FeedIDs
	if err := e.DB.Users.Update(&e.User); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, &e.User)
}

type GetUserItemStates struct {
	DB     *db.DB
	UserID db.ID
	Params struct {
		ModifiedSince time.Time `param:"modified_since"`
	}
}

func (e *GetUserItemStates) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModel(&middleware.RequireExistingModelOpts{
			Collection: e.DB.Users,
			BoundName:  "id",
			ID:         &e.UserID,
		}),
	}
}

func (e *GetUserItemStates) Handle(c *gin.Context) {
	var query db.Query
	if !e.Params.ModifiedSince.IsZero() {
		query.Filter = db.M{
			"modification_time": db.M{"$gt": e.Params.ModifiedSince},
		}
	}

	states, err := e.DB.Users.FindItemStates(e.UserID, query)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, states)
}

type UpdateUserItemState struct {
	DB        *db.DB
	ItemState db.ItemState
	UserID    db.ID
	ItemID    db.ID
}

func (e *UpdateUserItemState) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModel(&middleware.RequireExistingModelOpts{
			Collection: e.DB.Users,
			BoundName:  "id",
			ID:         &e.UserID,
		}),
		middleware.RequireExistingModel(&middleware.RequireExistingModelOpts{
			Collection: e.DB.Items,
			BoundName:  "itemID",
			ID:         &e.ItemID,
		}),
		middleware.UnmarshalBody(&e.ItemState),
	}
}

func (e *UpdateUserItemState) Handle(c *gin.Context) {
	e.ItemState.ItemID = e.ItemID

	switch err := e.DB.Users.UpsertItemState(e.UserID, &e.ItemState); err {
	case nil:
		c.JSON(http.StatusOK, &e.ItemState)
	case db.ErrOutdatedResource:
		c.JSON(http.StatusConflict, gin.H{"error": "resource is out of date"})
		c.Abort()
	default:
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

type DeleteUserItemState struct {
	DB     *db.DB
	UserID db.ID
	ItemID db.ID
}

func (e *DeleteUserItemState) Bind() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.RequireExistingModel(&middleware.RequireExistingModelOpts{
			Collection: e.DB.Users,
			BoundName:  "id",
			ID:         &e.UserID,
		}),
		middleware.RequireExistingModel(&middleware.RequireExistingModelOpts{
			Collection: e.DB.Items,
			BoundName:  "itemID",
			ID:         &e.ItemID,
		}),
	}
}

func (e *DeleteUserItemState) Handle(c *gin.Context) {
	if err := e.DB.Users.DeleteItemState(e.UserID, e.ItemID); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}
