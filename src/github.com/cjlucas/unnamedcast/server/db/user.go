package db

import (
	"time"

	"golang.org/x/crypto/bcrypt"

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

func (db *DB) users() *mgo.Collection {
	return db.db().C("users")
}

func (db *DB) FindUsers(q interface{}) Query {
	return &query{
		s: db.s,
		q: db.users().Find(q),
	}
}

func (db *DB) FindUserByID(id bson.ObjectId) Query {
	return db.FindUsers(bson.M{"_id": id})
}

func (db *DB) CreateUser(username, password string) (*User, error) {
	pw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user := User{
		ID:               bson.NewObjectId(),
		Username:         username,
		Password:         string(pw),
		CreationTime:     now,
		ModificationTime: now,
	}

	if err := db.users().Insert(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUser updates and existing user. The User must already persist in the
// database (in other words, it must have a valid ID)
func (db *DB) UpdateUser(user *User) error {
	// NOTE: Fetching the user (for possibly a second time in a single context)
	// may turn out to be overkill. MongoDB supports updating fields selectivly
	// by using update operators instead of passing in a whole document.

	var origUser User
	if err := db.FindUserByID(user.ID).One(&origUser); err != nil {
		return err
	}

	if CopyModel(origUser, user, "ID", "Username", "Password") {
		user.ModificationTime = time.Now().UTC()
	}

	return db.users().UpdateId(user.ID, user)
}

func (db *DB) EnsureUserIndex(idx Index) error {
	return db.users().EnsureIndex(mgoIndexForIndex(idx))
}
