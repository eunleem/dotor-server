package main

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"strconv"
	"time"
)

const TableNameHospitals = "hospitals"

var dbcHospitals *mgo.Collection

func init() {
	const tableName = TableNameHospitals
	dbcHospitals = dbSession.DB(dbName).C(tableName)
	dbCols[tableName] = dbSession.DB(dbName).C(tableName)

	index := mgo.Index{
		Key:        []string{"$2dsphere:location"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	if err := dbCols[tableName].EnsureIndex(index); err != nil {
		log.Printf("Error while ensuring index for '%s'. %s", tableName, err.Error())
		panic(err)
	}

	index = mgo.Index{
		Key:        []string{"googleplaceid"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	if err := dbCols[tableName].EnsureIndex(index); err != nil {
		log.Printf("Error while ensuring index for '%s'. %s", tableName, err.Error())
		panic(err)
	}
}

// Address similar as in android.location.Address Java Package
// Hospital for hospital
type Hospital struct {
	Id            bson.ObjectId     `bson:"_id" json:"id"`
	Name          string            `bson:"name" json:"name"`
	Location      GeoJson           `bson:"location" json:"location"`
	Address       string            `bson:",omitempty" json:"address,omitempty"`
	PhoneNumber   string            `bson:",omitempty" json:"phone_number,omitempty"`
	ContactInfo   map[string]string `bson:",omitempty" json:"contact_info,omitempty"`
	ExtraInfo     map[string]string `bson:",omitempty" json:"extra_info,omitempty"`
	GooglePlaceId string            `bson:"googleplaceid,omitempty" json:"placeid,omitempty"`
	Likes         []bson.ObjectId   `json:"likes,omitempty"`
	Updated       time.Time         `json:"updated"`
	Created       time.Time         `json:"created"`
}

////////////////////////     Basic Ops     ///////////////////////////

func (i *Hospital) Insert() (err error) {
	i.Id = bson.NewObjectId()
	i.Updated = time.Now()
	i.Created = time.Now()

	if err = dbcHospitals.Insert(&i); err != nil {
		log.Println("Could not insert a hospital.")
		return err
	}

	return nil
}

func (i *Hospital) Update() (err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Hospital Id")
		return
	}

	err = dbcHospitals.UpdateId(i.Id, &i)
	return
}

func (i *Hospital) Delete() (err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Hospital Id")
		return
	}

	err = dbcHospitals.RemoveId(i.Id)
	return
}

func (i *Hospital) GetById(id bson.ObjectId) (err error) {
	if err = dbcHospitals.FindId(id).One(&i); err != nil {
		log.Println("Could not find HospitalById.")
		return
	}

	return
}

func (i *Hospital) Like(userId bson.ObjectId) (alreadyLiked bool, err error) {
	if err := i.GetById(i.Id); err != nil {
		return false, err
	}

	n, err := dbcHospitals.Find(bson.M{"_id": i.Id, "likes": bson.M{"$in": []bson.ObjectId{userId}}}).Count()
	if err != nil {
		return false, err
	}

	if n > 0 {
		return true, nil
	}

	i.Likes = append(i.Likes, userId)
	err = i.Update()

	if err != nil {
		log.Println("Erro while liking hospital.")
		return false, err
	}
	return false, err
}

/////////////////////////    CONTROLLERS   ///////////////////////////

func getHospital(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	idStr := gc.Param("id")
	if bson.IsObjectIdHex(idStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid id."})
		return
	}

	id := bson.ObjectIdHex(idStr)
	var hospital Hospital
	if err := hospital.GetById(id); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Error while retrieving hospital from DB.",
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":   0,
		"message":  "Successfully fetched Hospital.",
		"hospital": hospital,
	})
	return
}

func getHospitals(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	skip := 0
	limit := 50

	// TODO Get Hospitals by Filter

	var hospitals []Hospital
	if err := dbcHospitals.Find(nil).Sort("created").Skip(skip).Limit(limit).All(&hospitals); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while retrieving data from DB."})
		return
	}

	if len(hospitals) == 0 {
		log.Print("No Hospitals")
		gc.JSON(http.StatusOK, gin.H{"status": -2, "message": "No hospitals."})
		return
	}

	log.Println("Fetched " + strconv.Itoa(len(hospitals)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Successfully fetched My Hospitals.", "data": hospitals})
	return
}

func getHospitalsNearby(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	DumpRequestBody(gc)

	var posted NearRequest
	if err := gc.BindJSON(&posted); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Could not bind.",
		})
		return
	}

	var results []Hospital

	if err := dbcHospitals.Find(bson.M{
		"location": bson.M{
			"$nearSphere": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{posted.Longitude, posted.Latitude},
				},
				"$maxDistance": posted.Distance,
			},
		},
	}).Limit(20).All(&results); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Could not find.",
		})
		return
	}

	if len(results) == 0 {
		log.Print("No hospital found nearby.")
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "No Hospitals found nearby.",
		})
		return
	}

	if str, err := json.MarshalIndent(results, "", "  "); err == nil {
		log.Printf("%s", str)
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":    0,
		"message":   "Fetched Hospitals nearby.",
		"hospitals": results,
	})
}

func insertHospital(gc *gin.Context) {
	//loggedIn, _ := isLoggedIn(gc)
	//if loggedIn == false {
	//return
	//}

	DumpRequestBody(gc)

	var posted Hospital
	if err := gc.BindJSON(&posted); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Parsing posted JSON failed."})
		return
	}

	posted.Location.Type = "Point"

	if err := posted.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed to insert hospital to DB."})
		return
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Uploaded hospital!", "newid": posted.Id.Hex()})
		return
	}
}

func updateHospital(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	var posted Hospital
	if err := gc.BindJSON(&posted); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Parsing posted JSON failed."})
		return
	}

	if err := posted.Update(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed to insert hospital to DB."})
		return
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Uploaded hospital!", "newid": posted.Id.Hex()})
		return
	}

}

func likeHospital(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	idStr := gc.Param("id")
	// TODO Check if valid
	if bson.IsObjectIdHex(idStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Param id is invalid objectId."})
		return
	}

	id := bson.ObjectIdHex(idStr)

	var hospital Hospital
	if err := hospital.GetById(id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "No Hospital found matching ObjectId."})
		return
	}

	if alreadyLiked, err := hospital.Like(myAccount.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while updating DB!"})
		return

	} else {
		if alreadyLiked == true {
			gc.JSON(http.StatusOK, gin.H{"status": 1, "message": "Your already liked it!"})
			return
		}
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "You Liked the hospital!"})
	return
}
