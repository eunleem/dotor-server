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

const TableNameReports = "reports"

var dbcReports *mgo.Collection

func init() {
	const tableName = TableNameReports
	dbcReports = dbSession.DB(dbName).C(tableName)
	dbCols[tableName] = dbcReports

	{
		index := mgo.Index{
			Key:        []string{"userid", "category", "relatedid"},
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

	{
		index := mgo.Index{
			Key:        []string{"category", "relatedid"},
			Unique:     false,
			DropDups:   false,
			Background: true,
			Sparse:     true,
		}

		if err := dbCols[tableName].EnsureIndex(index); err != nil {
			log.Printf("Error while ensuring index for '%s'. %s", tableName, err.Error())
			panic(err)
		}

	}
}

// Report stores
type Report struct {
	Id        bson.ObjectId `bson:"_id" json:"id"`
	UserId    bson.ObjectId `json:"userid"`
	Category  string        `json:"category"`  // Either Comment or Review
	RelatedId bson.ObjectId `json:"relatedid"` // Either CommentId or ReviewId
	Reason    string        `json:"reason"`
	Body      string        `bson:",omitempty" json:"body,omitempty"`
	IsRead    bool          `json:"isread"`
	Created   time.Time     `json:"created"`
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

	if relatedId, err := getIdFromParam(gc); err != nil {
		return
	} else {
		report.RelatedId = relatedId
	}

	report.Category = gc.Param("category")

	report.UserId = myAccount.Id
	report.Created = time.Now()

	// Check for Duplicate Reports
	if count, err := dbcReports.Find(bson.M{
		"userid":    myAccount.Id,
		"category":  report.Category,
		"relatedid": report.RelatedId,
	}).Count(); err != nil {

		gc.JSON(http.StatusInternalServerError, gin.H{
			"status":  -1,
			"message": "Internal Server Error .",
		})
		return
	} else {
		if count > 0 {
			gc.JSON(http.StatusOK, gin.H{
				"status":  1,
				"message": "Already Reported.",
			})
			return
		}
	}

	// #TODO Limit the number of Reports per user

	if _, err := report.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Failed! Insert report info.",
		})
		return
	}

	if count, err := dbcReports.Find(bson.M{
		"category":  report.Category,
		"relatedid": report.RelatedId,
	}).Count(); err != nil {

		gc.JSON(http.StatusInternalServerError, gin.H{
			"status":  -1,
			"message": "Internal Server Error .",
		})
		return
	} else {
		if count > 5 {
			// #TODO Suspend this post automatically.
		}
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successful! Insert report info.",
	})
}
