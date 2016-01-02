package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// ItemState represents the state of an unplayed/in progress items
// Played items will not have an associated state.
type ItemState struct {
	FeedID   bson.ObjectId `json:"feed_id"`
	ItemGUID string        `json:"item_guid"`
	Position time.Duration `json:"position"` // 0 if item not in progress
}

type User struct {
	ID         bson.ObjectId   `bson:"_id,omitempty" json:"id"`
	Username   string          `json:"name"`
	Password   string          `json:"-"` // encrypted
	FeedIDs    []bson.ObjectId `json:"feeds"`
	ItemStates []ItemState     `json:"states"`
}

func users() *mgo.Collection {
	return gSession.DB("test").C("users")
}

func loadUser(idHex string) *User {
	if !bson.IsObjectIdHex(idHex) {
		return nil
	}

	var user User
	users().FindId(bson.ObjectIdHex(idHex)).One(&user)

	return &user
}

func RequireValidUserID(c *gin.Context) {
	id := c.Param("id")
	user := loadUser(id)

	if user == nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		c.Abort()
	} else {
		c.Set("user", user)
	}
}

func CreateUser(c *gin.Context) {
	username := strings.TrimSpace(c.Query("name"))
	password := strings.TrimSpace(c.Query("pass"))

	if username == "" || password == "" {
		c.JSON(400, gin.H{"error": "missing required parameter(s)"})
		return
	}

	passwdEnc, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	user := User{
		ID:       bson.NewObjectId(),
		Username: username,
		Password: string(passwdEnc),
	}

	err = users().Insert(&user)

	fmt.Println("after insert", user.ID)

	if err != nil {
		c.JSON(400, gin.H{"error": err})
	} else {
		c.JSON(200, gin.H{"user": user})
	}
}

func ReadUser(c *gin.Context) {
	user := c.MustGet("user").(*User)
	c.JSON(200, user)
}

func UpdateUserFeeds(c *gin.Context) {
	user := c.MustGet("user").(*User)
	rawBody, _ := ioutil.ReadAll(c.Request.Body)

	var body struct {
		FeedsIDs []bson.ObjectId `json:"feeds"`
	}

	if err := json.Unmarshal(rawBody, &body); err != nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}

	user.FeedIDs = body.FeedsIDs
	if err := users().Update(bson.M{"_id": user.ID}, &user); err != nil {
		c.JSON(400, gin.H{"error": "could not update user"})
	} else {
		c.JSON(200, &user)
	}
}

func UpdateUserItemStates(c *gin.Context) {
	user := c.MustGet("user").(*User)
	rawBody, _ := ioutil.ReadAll(c.Request.Body)

	var body struct {
		States []ItemState `json:"states"`
	}

	if err := json.Unmarshal(rawBody, &body); err != nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}

	user.ItemStates = body.States

	if err := users().Update(bson.M{"_id": user.ID}, &user); err != nil {
		c.JSON(400, gin.H{"error": "could not update user"})
	} else {
		c.JSON(200, &user)
	}
}
