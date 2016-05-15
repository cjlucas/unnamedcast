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
	ItemID           bson.ObjectId `json:"item_id" bson:"item_id"`
	Position         float64       `json:"position" bson:"position"` // 0 if item is unplayed
	ModificationTime time.Time     `json:"modification_time" bson:"modification_time"`
}

type User struct {
	ID               bson.ObjectId   `bson:"_id,omitempty" json:"id"`
	Username         string          `json:"username" bson:"username"`
	Password         string          `json:"-" bson:"password"` // encrypted
	FeedIDs          []bson.ObjectId `json:"feeds" bson:"feed_ids"`
	ItemStates       []ItemState     `json:"states" bson:"states"`
	CreationTime     time.Time       `json:"creation_time" bson:"creation_time"`
	ModificationTime time.Time       `json:"modification_time" bson:"modification_time"`
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

	if CopyModel(&origUser, user, "ID", "Username", "Password") {
		user.ModificationTime = time.Now().UTC()
	}

	return db.users().UpdateId(origUser.ID, &origUser)
}

func (db *DB) UpsertUserState(userID bson.ObjectId, state *ItemState) error {
	state.ModificationTime = time.Now().UTC()

	sel := bson.M{
		"_id":            userID,
		"states.item_id": state.ItemID,
	}

	err := db.users().Update(sel, bson.M{
		"$set": bson.M{"states.$": &state},
	})

	if err == ErrNotFound {
		sel := bson.M{
			"_id": userID,
			"states.item_id": bson.M{
				"$ne": state.ItemID,
			},
		}

		// Push the state on the front of the array to keep the array
		// sorted by modification time (desc). In a normal use-case, this will
		// improve the speed of the initial update operation
		return db.users().Update(sel, bson.M{
			"$push": bson.M{
				"states": bson.M{
					"$each":     []ItemState{*state},
					"$position": 0,
				},
			},
		})
	}

	return err
}

func (db *DB) DeleteUserState(userID, itemID bson.ObjectId) error {
	return db.users().UpdateId(userID, bson.M{
		"$pull": bson.M{
			"states": bson.M{"item_id": itemID},
		},
	})
}

func (db *DB) EnsureUserIndex(idx Index) error {
	return db.users().EnsureIndex(mgoIndexForIndex(idx))
}
