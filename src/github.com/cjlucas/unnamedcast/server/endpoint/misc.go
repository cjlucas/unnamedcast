package endpoint

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/cjlucas/unnamedcast/db"
	"github.com/gin-gonic/gin"
)

type SearchFeeds struct {
	DB     *db.DB
	Params struct {
		limitParams
		Query string `param:"q,require"`
	}
}

func (e *SearchFeeds) Bind() []gin.HandlerFunc {
	return nil
}

func (e *SearchFeeds) Handle(c *gin.Context) {
	limit := e.Params.limitParams.Limit()
	if limit > 50 {
		limit = 50
	}

	query := db.Query{
		Filter: db.M{
			"$text": db.M{"$search": e.Params.Query},
		},
		SortField: "$textScore:score",
		SortDesc:  true,
		Limit:     limit,
	}

	var results []db.Feed
	if err := e.DB.Feeds.Find(&query).All(&results); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if results == nil {
		results = make([]db.Feed, 0)
	}

	c.JSON(http.StatusOK, results)
}

type Login struct {
	DB       *db.DB
	Username string `param:",require"`
	Password string `param:",require"`
}

func (e *Login) Bind() []gin.HandlerFunc {
	return nil
}

func (e *Login) Handle(c *gin.Context) {
	cur := e.DB.Users.Find(&db.Query{
		Filter: db.M{"username": e.Username},
		Limit:  1,
	})

	var user db.User
	if err := cur.One(&user); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(e.Password)); err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.JSON(http.StatusOK, &user)
}
