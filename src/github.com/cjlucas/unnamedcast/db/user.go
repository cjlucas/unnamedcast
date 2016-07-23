package db

import (
	"github.com/cjlucas/unnamedcast/db/utctime"

	"golang.org/x/crypto/bcrypt"

	"gopkg.in/mgo.v2/bson"
)

type itemState int

const (
	stateUnplayed itemState = 0
	stateInProgress
	statePlayed
)

// ItemState represents the state of an unplayed/in progress items
// Played items will not have an associated state.
type ItemState struct {
	ItemID           ID           `json:"item_id" bson:"item_id"`
	State            itemState    `json:"state" bson:"state"`
	Position         float64      `json:"position" bson:"position"` // 0 if item is unplayed
	ModificationTime utctime.Time `json:"modification_time" bson:"modification_time"`
}

type User struct {
	ID               ID           `bson:"_id,omitempty" json:"id"`
	Username         string       `json:"username" bson:"username" index:",unique"`
	Password         string       `json:"-" bson:"password"` // encrypted
	FeedIDs          []ID         `json:"feeds" bson:"feed_ids" index:"feed_ids"`
	ItemStates       []ItemState  `json:"states" bson:"states"`
	CreationTime     utctime.Time `json:"creation_time" bson:"creation_time"`
	ModificationTime utctime.Time `json:"modification_time" bson:"modification_time"`
}

type UserCollection struct {
	collection

	ItemStateCollection collection
}

func (c UserCollection) Create(username, password string) (*User, error) {
	pw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := utctime.Now()
	user := User{
		ID:               NewID(),
		Username:         username,
		Password:         string(pw),
		CreationTime:     now,
		ModificationTime: now,
	}

	if err := c.c.Insert(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c UserCollection) UpsertItemState(userID ID, state *ItemState) error {
	// New steps
	// 1. Find existing state. If exists, $pull it off (or pull after step 2 on reject)
	// 2. Compare modification time, if existing state is newer, reject
	// 3. Prepend state to array always

	pipeline := []bson.M{
		{"$match": bson.M{"_id": userID}},
		{"$project": bson.M{
			"states": bson.M{
				"$filter": bson.M{
					"input": "$states",
					"as":    "state",
					"cond": bson.M{
						"$eq": []interface{}{"$$state.item_id", state.ItemID},
					},
				},
			},
		}},
	}

	var user User
	if err := c.pipeline(pipeline).One(&user); err != nil {
		// TODO: don't assume ErrNotFound
		return ErrNotFound
	}

	if len(user.ItemStates) > 0 {
		if state.ModificationTime.Before(user.ItemStates[0].ModificationTime) {
			return ErrOutdatedResource
		}

		sel := bson.M{
			"_id":            userID,
			"states.item_id": state.ItemID,
		}

		return c.c.Update(sel, bson.M{
			"$set": bson.M{"states.$": &state},
		})
	}

	sel := bson.M{
		"_id": userID,
		// $ne operator ensures there will be no race condition
		"states.item_id": bson.M{
			"$ne": state.ItemID,
		},
	}

	// Push the state on the front of the array to keep the array
	// sorted by modification time (desc). In a normal use-case, this will
	// improve the speed of the initial update operation
	return c.c.Update(sel, bson.M{
		"$push": bson.M{
			"states": bson.M{
				"$each":     []ItemState{*state},
				"$position": 0,
			},
		},
	})
}

// Update updates an existing user. The User must already persist in the
// database (in other words, it must have a valid ID)
func (c UserCollection) Update(user *User) error {
	// NOTE: Fetching the user (for possibly a second time in a single context)
	// may turn out to be overkill. MongoDB supports updating fields selectivly
	// by using update operators instead of passing in a whole document.

	var origUser User
	if err := c.FindByID(user.ID).One(&origUser); err != nil {
		return err
	}

	if CopyModel(&origUser, user, "ID", "Username", "Password") {
		user.ModificationTime = utctime.Now()
	}

	return c.c.UpdateId(origUser.ID, &origUser)
}

func (c UserCollection) DeleteItemState(userID, itemID ID) error {
	return c.c.UpdateId(userID, bson.M{
		"$pull": bson.M{
			"states": bson.M{"item_id": itemID},
		},
	})
}

func (c UserCollection) FindItemStates(userID ID, query Query) ([]ItemState, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"_id": userID}},
		{"$project": bson.M{
			"states": bson.M{
				"$filter": bson.M{
					"input": "$states",
					"as":    "state",
					"cond":  c.ItemStateCollection.filterCond(query, "state"),
				},
			},
		},
		},
	}

	var user User
	if err := c.pipeline(pipeline).One(&user); err != nil {
		return nil, err
	}

	return user.ItemStates, nil
}
