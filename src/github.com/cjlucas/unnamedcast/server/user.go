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
	Position float64       `json:"position"` // 0 if item is unplayed
}

type User struct {
	ID               bson.ObjectId   `bson:"_id,omitempty" json:"id"`
	Username         string          `json:"username"`
	Password         string          `json:"-"` // encrypted
	FeedIDs          []bson.ObjectId `json:"feeds"`
	ItemStates       []ItemState     `json:"states"`
	CreationTime     time.Time       `json:"creation_time"`
	ModificationTime time.Time       `json:"modification_time"`
}

func (u *User) Create() error {
	u.ID = bson.NewObjectId()
	u.CreationTime = time.Now().UTC()
	u.ModificationTime = u.CreationTime
	return users().Insert(u)
}

func (u *User) Update(new *User) error {
	if new == nil || CopyModel(u, new, "ID", "Username", "Password") {
		u.ModificationTime = time.Now().UTC()
	}
	return users().Update(bson.M{"_id": u.ID}, u)
}

func (u *User) FindByName(username string) error {
	return users().Find(bson.M{"username": username}).One(u)
}

func users() *mgo.Collection {
	return gSession.DB("test").C("users")
}

func loadUser(idHex string) *User {
	if !bson.IsObjectIdHex(idHex) {
		return nil
	}

	var user User
	if err := users().FindId(bson.ObjectIdHex(idHex)).One(&user); err != nil {
		fmt.Println("Error loading user:", err)
	}

	return &user
}

func RequireValidUserID(c *gin.Context) {
	id := c.Param("id")
	user := loadUser(id)

	if user == nil {
		c.AbortWithStatus(404)
		return
	}

	c.Set("user", user)
}

func FindAllUsers(c *gin.Context) {
	var out []User
	if err := users().Find(nil).All(&out); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, &out)
}

func CreateUser(c *gin.Context) {
	username := strings.TrimSpace(c.Query("username"))
	password := strings.TrimSpace(c.Query("password"))

	if username == "" || password == "" {
		c.JSON(400, gin.H{"error": "missing required parameter(s)"})
		return
	}

	passwdEnc, _ := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	user := User{
		ID:       bson.NewObjectId(),
		Username: username,
		Password: string(passwdEnc),
	}

	if err := user.Create(); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, &user)
}

func ReadUser(c *gin.Context) {
	user := c.MustGet("user").(*User)
	c.JSON(200, user)
}

func GetUserItemStates(c *gin.Context) {
	user := c.MustGet("user").(*User)
	c.JSON(200, user.ItemStates)
}

func GetUserFeeds(c *gin.Context) {
	user := c.MustGet("user").(*User)
	c.JSON(200, &user.FeedIDs)
}

func UpdateUserFeeds(c *gin.Context) {
	user := c.MustGet("user").(*User)

	body, _ := ioutil.ReadAll(c.Request.Body)
	var ids []bson.ObjectId

	if err := json.Unmarshal(body, &ids); err != nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}

	user.FeedIDs = ids
	if err := user.Update(nil); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, user)
}

func UpdateUserItemStates(c *gin.Context) {
	user := c.MustGet("user").(*User)

	var states []ItemState
	body, _ := ioutil.ReadAll(c.Request.Body)

	if err := json.Unmarshal(body, &states); err != nil {
		c.AbortWithError(400, err)
		return
	}

	user.ItemStates = states

	if err := user.Update(nil); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, &user.ItemStates)
}
