package main

import (
	"fmt"
	"log"

	"gopkg.in/mgo.v2"
)

// dbSession for MongoDB
var dbSession = openSession()

var dbCols = make(map[string]*mgo.Collection)
var dbIndexes = make(map[string]mgo.Index)

func init() {

}

func openSession() *mgo.Session {
	dbSession, err := mgo.Dial("localhost")
	if err != nil {
		fmt.Println("Could not connect to MongoDB!")
		panic(err)
	}

	dbSession.SetMode(mgo.Monotonic, true)
	return dbSession
}

// CloseDb closes DB connection
func CloseDb() {
	dbSession.Close()
}

func DropDb() {
	if dbSession == nil {
		log.Fatal("Db is Not Open.")
		return
	}
	err := dbSession.DB(dbName).DropDatabase()
	if err != nil {
		panic(err)
	}
}

type Near struct {
	Coordinate  GeoJson `json:"$geometry"`
	MaxDistance float64 // in Meters
	MinDistance float64 // in Meters
}

type NearRequest struct {
	Longitude float64 `json:"longitude" binding:"required"`
	Latitude  float64 `json:"latitude" binding:"required"`
	Distance  float64 `json:"distance" binding:"required"` // in Meters
}

type GeoJson struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates,omitempty"`
}

func NewNear(lat float64, long float64, dist float64) (near Near, err error) {
	geo := GeoJson{
		Type:        "Point",
		Coordinates: make([]float64, 2, 2),
	}
	geo.Coordinates[0] = long
	geo.Coordinates[1] = lat

	near = Near{
		MinDistance: 0,
		MaxDistance: dist,
		Coordinate:  geo,
	}

	return near, nil
}

//func OpenCollections() {
//if dbSession == nil {
//log.Fatalln("Open DB first!")
////return
//}

//dbcUsers = dbSession.DB(dbName).C("users")
//dbcUserData = dbSession.DB(dbName).C("userdata")
//dbcPets = dbSession.DB(dbName).C("pets")
//dbcReviews = dbSession.DB(dbName).C("reviews")
//dbcComments = dbSession.DB(dbName).C("comments")
//dbcImages = dbSession.DB(dbName).C("images")
//dbcNotifications = dbSession.DB(dbName).C("notifications")
//dbcPushSettings = dbSession.DB(dbName).C("push_settings")
//dbcFeedbacks = dbSession.DB(dbName).C("feedbacks")
//dbcReports = dbSession.DB(dbName).C("reports")
//}

// CloseDb closes DB connection
