package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Person struct {
	ID               bson.ObjectId `bson:"_id,omitempty" json:"id"`
	Name             string        `json:"name"`
	Title            string        `json:"title"`
	CreationTime     time.Time
	ModificationTime time.Time
}

// type Thing struct {
// 	Balls []map[string]int
// }
//
var gSession *mgo.Session

func FetchPersons(w http.ResponseWriter, r *http.Request) {
	c := gSession.DB("test").C("people")

	var people []Person
	c.Find(nil).All(&people)

	p, _ := json.Marshal(people)

	w.Write(p)

	// fmt.Println("hi")
	//
	// c := gSession.DB("test").C("blah")
	//
	// var p []Thing
	//
	// c.Find(nil).All(&p)
	//
	// fmt.Printf("%#v\n", p)
	//
	// pipeData := bson.M{
	// 	"$project": bson.M{
	// 		"balls": bson.M{
	// 			"$filter": bson.M{
	// 				"input": "$balls",
	// 				"as":    "item",
	// 				"cond":  bson.M{"$gte": []interface{}{"$$item.count", 2}},
	// 			},
	// 		},
	// 	},
	// }
	//
	// var people []Thing
	//
	// c.Pipe([]bson.M{pipeData}).All(&people)
	// fmt.Printf("%#v\n", people)
}

func main() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	gSession = session

	err = feeds().EnsureIndex(mgo.Index{
		Key:      []string{"url"},
		Unique:   true,
		DropDups: true,
	})

	if err != nil {
		panic(err)
	}

	err = feeds().EnsureIndex(mgo.Index{
		Key: []string{"itunes_id"},
	})
	if err != nil {
		panic(err)
	}

	g := gin.Default()

	api := g.Group("/api")

	api.POST("/users", CreateUser)
	api.GET("/users/:id", RequireValidUserID, ReadUser)
	api.PUT("/users/:id/feeds", RequireValidUserID, UpdateUserFeeds)
	api.PUT("/users/:id/states", RequireValidUserID, UpdateUserItemStates)

	api.POST("/feeds", CreateFeed)
	api.GET("/feeds/:id", RequireValidFeedID, ReadFeed)
	api.GET("/feeds", FindFeed)
	api.PUT("/feeds/:id/items", RequireValidFeedID, UpdateFeedItems)

	g.Run(":8081")
}
