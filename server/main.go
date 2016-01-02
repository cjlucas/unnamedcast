package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"

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

func AddPerson(w http.ResponseWriter, r *http.Request) {
	fmt.Println("AddPerson")
	fmt.Printf("%#v\n", mux.Vars(r))

	data, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(data))

	var p Person
	json.Unmarshal(data, &p)

	c := gSession.DB("test").C("people")

	p.CreationTime = time.Now()
	c.Insert(&p)

	fmt.Println("Added that shit")
}

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

	g := gin.Default()

	api := g.Group("/api")

	api.POST("/user", CreateUser)
	api.GET("/users/:id", RequireValidUserID, ReadUser)
	api.PUT("/users/:id/feeds", RequireValidUserID, UpdateUserFeeds)
	api.PUT("/users/:id/states", RequireValidUserID, UpdateUserItemStates)

	api.POST("/feed", CreateFeed)
	api.GET("/feeds/:id", RequireValidFeedID, ReadFeed)
	api.GET("/feeds", FindFeed)
	api.PUT("/feeds/:id/items", RequireValidFeedID, UpdateFeedItems)

	g.Run(":8081")
}
