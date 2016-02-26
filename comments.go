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

var dbcComments *mgo.Collection

// Comment for reviews
type Comment struct {
	Id          bson.ObjectId   `bson:"_id" json:"id"`
	ReviewId    bson.ObjectId   `json:"-"`
	UserId      bson.ObjectId   `json:"userid"`
	CommentBody string          `json:"commentbody"`
	Likes       []bson.ObjectId `bson:",omitempty" json:"likes"`
	Created     time.Time       `json:"created"`
}

func ensureIndexesComments() (err error) {
	index := mgo.Index{
		Key:        []string{"reviewid"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	if err = dbcComments.EnsureIndex(index); err != nil {
		return errors.New("Could not ensure index for Comments.")
	}

	return
}

//////////////////////////      PET      /////////////////////////

func (i *Comment) Insert() (bson.ObjectId, error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}

	if i.Created.IsZero() {
		i.Created = time.Now()
	}

	err := dbcComments.Insert(&i)
	if err != nil {
		log.Println("Could not insert a comment.")
		return i.Id, err
	}

	log.Println("Inserted a comment. ReviewId: " + i.ReviewId.Hex())
	log.Println("CommentId: " + i.Id.Hex())

	return i.Id, nil
}

func (i *Comment) Update() (changeInfo *mgo.ChangeInfo, err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Comment Id")
		return
	}

	changeInfo, err = dbcComments.UpsertId(i.Id, &i)
	return
}

func (i *Comment) Delete() (err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Comment Id")
		return
	}

	err = dbcComments.RemoveId(i.Id)
	return
}

func (i *Comment) GetById(id bson.ObjectId) error {
	err := dbcComments.FindId(id).One(&i)
	if err != nil {
		log.Println("Could not find CommentById.")
		return err
	}

	return nil
}

/////////////////////////    CONTROLLERS    ////////////////////////

func getComments(gc *gin.Context) {
	isLoggedIn, _ := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	reviewIdStr := gc.Param("reviewid")
	// TODO Check if valid
	if bson.IsObjectIdHex(reviewIdStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Param id is invalid objectId."})
		return
	}

	reviewId := bson.ObjectIdHex(reviewIdStr)

	var review Review
	if err := review.GetById(reviewId); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "No Review found matching ObjectId."})
		return
	}

	var comments []Comment
	if err := dbcComments.Find(bson.M{"reviewid": review.Id}).Sort("created").All(&comments); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while retrieving data from DB."})
		return
	}

	if len(comments) == 0 {
		log.Print("No Comments")
		gc.JSON(http.StatusOK, gin.H{"status": -2, "message": "No comments."})
		return
	}

	log.Println("Fetched " + strconv.Itoa(len(comments)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Successfully fetched Comments.", "comments": comments})
	return
}

func insertComment(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	reviewIdStr := gc.Param("reviewid")
	// TODO Check if valid
	if bson.IsObjectIdHex(reviewIdStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Param id is invalid objectId."})
		return
	}

	reviewId := bson.ObjectIdHex(reviewIdStr)

	var review Review
	if err := review.GetById(reviewId); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "No Review found matching ObjectId."})
		return
	}

	var comment Comment
	if err := gc.BindJSON(&comment); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	comment.ReviewId = review.Id
	comment.UserId = user.Id
	comment.Created = time.Now()

	// #TODO Limit the number of Comments per user

	if _, err := comment.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Failed! Insert comment info.",
		})
		return
	}

	if err := review.AddComment(comment.Id); err != nil {
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Failed! Insert comment to Review.",
		})
		return
	}

	// Do not notify me on my review
	if review.UserId != user.Id {
		notification := Notification{
			Id:          bson.NewObjectId(), // Insert a new Notification
			UserId:      review.UserId,      // For user who owns the Review
			Type:        "review_comments",
			Message:     user.Nickname + " left a comment on your review!",
			RelatedType: "review",
			RelatedId:   review.Id,
			IsRead:      false, // It's new and not read.
			IsSent:      false,
		}

		if _, err := notification.Insert(); err != nil {
			log.Print(err)
		}
	}
	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successful! Insert comment info.",
	})
}

func updateComment(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	DumpRequestBody(gc)

	postedComment := Comment{}

	if err := gc.BindJSON(&postedComment); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	log.Println("commentid: " + postedComment.Id.Hex())

	comment := Comment{}

	if err := comment.GetById(postedComment.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid CommentId. Id: " + postedComment.Id.Hex()})
		return
	}

	if comment.UserId != user.Id {
		log.Println("User does not own this comment! commentId: " + comment.Id.Hex())
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Owner."})
		return
	}

	comment.CommentBody = postedComment.CommentBody

	if changeInfo, err := comment.Update(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed update comment."})
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Updated " + strconv.Itoa(changeInfo.Updated) + " field(s)."})
	}
	return
}

func deleteComment(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	postedComment := Comment{}

	if err := gc.BindJSON(&postedComment); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	comment := Comment{}

	if err := comment.GetById(postedComment.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid CommentId."})
		return
	}

	if comment.UserId != user.Id {
		log.Println("User does not own this comment!")
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Owner."})
		return
	}

	if err := comment.Delete(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed update comment."})
		return
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Removed comment."})
		return
	}
	return
}
