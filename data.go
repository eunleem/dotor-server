package main

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

// dbSession for MongoDB
var dbSession *mgo.Session

//var dbcContents *mgo.Collection
//var dbcEmailVerification *mgo.Collection

// Region for regions
type Region struct {
	Id        bson.ObjectId `bson:"_id" json:"id"`
	Region    string        `bson:"region" json:"region"`
	Latitude  float32       `json:"latitude"`
	Longitude float32       `json:"longitude"`
}

// Hospital Animal Hospital
type Hospital struct {
	Id       bson.ObjectId `bson:"_id" json:"id"`
	Name     string        `json:"name"`
	Locality string        `json:"locality"`
	Address  string        `json:"address"`
}

// Report stores
type Report struct {
	Id       bson.ObjectId `bson:"_id" json:"id"`
	UserId   bson.ObjectId `json:"userid"`
	ReviewId bson.ObjectId `json:"reviewid"`
	Category string        `json:"category" form:"category"`
	Body     string        `json:"body" form:"body"`
	Created  time.Time     `json:"created" form:"created"`
}

func init() {
	OpenDb()
	OpenCollections()
	err := ensureIndexes()
	if err != nil {
		log.Fatal(err)
	}
}

const dbName = "Dotor"

func DropDb() {
	err := dbSession.DB(dbName).DropDatabase()
	if err != nil {
		panic(err)
	}
}

func OpenDb() {
	var err error
	dbSession, err = mgo.Dial("localhost")
	if err != nil {
		fmt.Println("Could not connect to MongoDB!")
		panic(err)
	}

	dbSession.SetMode(mgo.Monotonic, true)
}

func OpenCollections() {
	if dbSession == nil {
		log.Fatalln("Open DB first!")
		return
	}
	dbcUsers = dbSession.DB(dbName).C("users")
	dbcUserData = dbSession.DB(dbName).C("userdata")
	dbcPets = dbSession.DB(dbName).C("pets")
	dbcReviews = dbSession.DB(dbName).C("reviews")
	dbcComments = dbSession.DB(dbName).C("comments")
	dbcImages = dbSession.DB(dbName).C("images")
	dbcNotifications = dbSession.DB(dbName).C("notifications")
	dbcPushSettings = dbSession.DB(dbName).C("push_settings")
}

// CloseDb closes DB connection
func CloseDb() {
	dbSession.Close()
}

func ensureIndexes() (err error) {
	if err = ensureIndexesUsers(); err != nil {
		return
	}
	//if err = ensureIndexesUserData(); err != nil {
	//return
	//}
	if err = ensureIndexesPets(); err != nil {
		return
	}
	if err = ensureIndexesReviews(); err != nil {
		return
	}
	return
}
