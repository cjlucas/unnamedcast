package main

import (
	"encoding/json"
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
	Position time.Duration `json:"position"` // 0 if item is unplayed
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
	// TODO: no error checking
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

func GetUserFeeds(c *gin.Context) {
	user := c.MustGet("user").(*User)

	const syncTokenKey = "X-Sync-Token"
	query := bson.M{
		"_id": bson.M{
			"$in": user.FeedIDs,
		},
	}

	token, err := DecodeSyncToken(c.Request.Header.Get(syncTokenKey))
	if err == nil {
		query["modification_time"] = bson.M{"$gt": token.SyncTime()}
	}

	var feedList []Feed
	if err := feeds().Find(query).All(&feedList); err != nil {
		c.JSON(404, gin.H{"error": "no results found"})
		return
	}

	if feedList == nil {
		feedList = make([]Feed, 0)
	}

	// Filter items by modification time
	// NOTE: This can be optimized by using an aggregate with $filter
	syncTime := token.SyncTime()
	for i := range feedList {
		feed := &feedList[i]

		var items []Item
		for i := range feed.Items {
			item := &feed.Items[i]
			if item.ModificationTime.After(syncTime) {
				items = append(items, *item)
			}
		}
		feed.Items = items
	}

	c.Header(syncTokenKey, GenerateSyncToken())
	c.JSON(200, &feedList)
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

func UpdateUserItemState(c *gin.Context) {
	user := c.MustGet("user").(*User)

	var body ItemState

	if err := c.Bind(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}

	found := false
	for i, s := range user.ItemStates {
		if s.ItemGUID == body.ItemGUID && s.FeedID == body.FeedID {
			user.ItemStates[i] = body
			found = true
			break
		}
	}

	if !found {
		user.ItemStates = append(user.ItemStates, body)
	}

	if err := user.Update(nil); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, &user)
}
