package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"strconv"
	"time"
)

var dbcFeedbacks *mgo.Collection

// Feedback for reviews
type Feedback struct {
	Id           bson.ObjectId   `bson:"_id" json:"id"`
	UserId       bson.ObjectId   `json:"userid"`
	FeedbackBody string          `json:"feedbackbody"`
	Images       []bson.ObjectId `json:"images"`
	Response     string          `bson:",omitempty" json:"response"`
	IsRead       bool            `json:"isread"`
	Responded    time.Time       `json:"responded"`
	Created      time.Time       `json:"created"`
}

func ensureIndexesFeedbacks() (err error) {
	index := mgo.Index{
		Key:        []string{"userid"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	if err = dbcFeedbacks.EnsureIndex(index); err != nil {
		return errors.New("Could not ensure index for Feedbacks.")
	}

	return
}

//////////////////////////      PET      /////////////////////////

func (i *Feedback) Insert() (bson.ObjectId, error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}
	if i.Created.IsZero() {
		i.Created = time.Now()
	}
	err := dbcFeedbacks.Insert(&i)
	if err != nil {
		log.Println("Could not insert a feedback.")
		return i.Id, err
	}
	log.Println("Inserted a feedback.Id: " + i.Id.Hex())
	return i.Id, nil
}

func (i *Feedback) Update() (changeInfo *mgo.ChangeInfo, err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Feedback Id")
		return
	}

	changeInfo, err = dbcFeedbacks.UpsertId(i.Id, &i)
	return
}

func (i *Feedback) Delete() (err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Feedback Id")
		return
	}

	err = dbcFeedbacks.RemoveId(i.Id)
	return
}

func (i *Feedback) GetById(id bson.ObjectId) error {
	err := dbcFeedbacks.FindId(id).One(&i)
	if err != nil {
		log.Println("Could not find FeedbackById.")
		return err
	}

	return nil
}

/////////////////////////    CONTROLLERS    ////////////////////////

func getFeedbacks(gc *gin.Context) {
	isLoggedIn, myAccount := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	reviewIdStr := gc.Param("reviewid")
	// TODO Check if valid
	if bson.IsObjectIdHex(reviewIdStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Param id is invalid objectId."})
		return
	}

	var feedbacks []Feedback
	if err := dbcFeedbacks.Find(bson.M{"userid": myAccount.Id}).Sort("-created").All(&feedbacks); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while retrieving data from DB."})
		return
	}

	if len(feedbacks) == 0 {
		log.Print("No Feedbacks")
		gc.JSON(http.StatusOK, gin.H{"status": -2, "message": "No feedbacks."})
		return
	}

	log.Println("Fetched " + strconv.Itoa(len(feedbacks)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Successfully fetched Feedbacks.", "feedbacks": feedbacks})
	return
}

func insertFeedback(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	var feedback Feedback
	if err := gc.BindJSON(&feedback); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	feedback.UserId = user.Id
	feedback.Created = time.Now()

	// #TODO Limit the number of Feedbacks per user

	if _, err := feedback.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Failed! Insert feedback info.",
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successful! Insert feedback info.",
	})
}

func updateFeedback(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	DumpRequestBody(gc)

	postedFeedback := Feedback{}

	if err := gc.BindJSON(&postedFeedback); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	log.Println("feedbackid: " + postedFeedback.Id.Hex())

	feedback := Feedback{}

	if err := feedback.GetById(postedFeedback.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid FeedbackId. Id: " + postedFeedback.Id.Hex()})
		return
	}

	if feedback.UserId != user.Id {
		log.Println("User does not own this feedback! feedbackId: " + feedback.Id.Hex())
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Owner."})
		return
	}

	feedback.FeedbackBody = postedFeedback.FeedbackBody

	if changeInfo, err := feedback.Update(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed update feedback."})
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Updated " + strconv.Itoa(changeInfo.Updated) + " field(s)."})
	}
	return
}

func deleteFeedback(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	postedFeedback := Feedback{}

	if err := gc.BindJSON(&postedFeedback); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	feedback := Feedback{}

	if err := feedback.GetById(postedFeedback.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid FeedbackId."})
		return
	}

	if feedback.UserId != user.Id {
		log.Println("User does not own this feedback!")
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Owner."})
		return
	}

	if err := feedback.Delete(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed update feedback."})
		return
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Removed feedback."})
		return
	}
	return
}
