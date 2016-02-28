package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"time"
)

var dbcReports *mgo.Collection

// Report stores
type Report struct {
	Id          bson.ObjectId `bson:"_id" json:"id"`
	UserId      bson.ObjectId `json:"userid"`
	RelatedType string        `json:"category"`
	RelatedId   bson.ObjectId `json:"reviewid"`
	Reason      string        `json:"reason"`
	Body        string        `bson:",omitempty" json:"body,omitempty"`
	IsRead      bool          `json:"isread"`
	Created     time.Time     `json:"created"`
}

func ensureIndexesReports() (err error) {
	index := mgo.Index{
		Key:        []string{"userid"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	if err = dbcReports.EnsureIndex(index); err != nil {
		return errors.New("Could not ensure index for Reports.")
	}

	return
}

//////////////////////////      PET      /////////////////////////

func (i *Report) Insert() (bson.ObjectId, error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}
	if i.Created.IsZero() {
		i.Created = time.Now()
	}
	err := dbcReports.Insert(&i)
	if err != nil {
		log.Println("Could not insert a report.")
		return i.Id, err
	}
	log.Println("Inserted a report.Id: " + i.Id.Hex())
	return i.Id, nil
}

func (i *Report) Update() (changeInfo *mgo.ChangeInfo, err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Report Id")
		return
	}

	changeInfo, err = dbcReports.UpsertId(i.Id, &i)
	return
}

func (i *Report) Delete() (err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Report Id")
		return
	}

	err = dbcReports.RemoveId(i.Id)
	return
}

func (i *Report) GetById(id bson.ObjectId) error {
	err := dbcReports.FindId(id).One(&i)
	if err != nil {
		log.Println("Could not find ReportById.")
		return err
	}

	return nil
}

/////////////////////////    CONTROLLERS    ////////////////////////

func insertReport(gc *gin.Context) {
	isLoggedIn, myAccount := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	var report Report
	if err := gc.BindJSON(&report); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	report.UserId = myAccount.Id
	report.Created = time.Now()

	// #TODO Limit the number of Reports per user
	// #TODO Check for Duplicate Reports

	if _, err := report.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Failed! Insert report info.",
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successful! Insert report info.",
	})
}
