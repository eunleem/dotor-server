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

const TableNameComments = "comments"

var dbcComments *mgo.Collection

func init() {
	const tableName = TableNameComments
	index := mgo.Index{
		Key:        []string{"category", "-relatedid"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}
	dbCols[tableName] = dbSession.DB(dbName).C(tableName)
	dbcComments = dbCols[tableName]
	if err := dbCols[tableName].EnsureIndex(index); err != nil {
		log.Printf("Error while ensuring index for '%s'. %s", tableName, err.Error())
	}
}

// Comment for reviews
type Comment struct {
	Id             bson.ObjectId   `bson:"_id" json:"id"`
	Category       string          `json:"-"` // Comment for a doc in which Collection
	RelatedId      bson.ObjectId   `json:"-"` // Comment for which document
	UserId         bson.ObjectId   `json:"userid"`
	Nickname       string          `json:"nickname"`
	CommentBody    string          `json:"commentbody"`
	ReplyTo        bson.ObjectId   `bson:",omitempty" json:"replyto_commentid"`
	UsersMentioned []bson.ObjectId `bson:",omitempty" json:"users_mentioned"`
	Likes          []bson.ObjectId `bson:",omitempty" json:"likes"`
	Created        time.Time       `json:"created"`
}

//////////////////////////      BASIC OPERATIONS      /////////////////////////

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

	log.Printf("Inserted a comment. Id: %s, Category: %s RelatedId: %s.",
		i.Id.Hex(), i.Category, i.RelatedId.Hex())

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
	if err := dbcComments.FindId(id).One(&i); err != nil {
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

	relatedIdStr := gc.Param("relatedid")
	if bson.IsObjectIdHex(relatedIdStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Param id is invalid objectId."})
		return
	}
	relatedId := bson.ObjectIdHex(relatedIdStr)

	//category := gc.Param("category")

	var review Review
	if err := review.GetById(relatedId); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "No Review found matching ObjectId."})
		return
	}

	sort := gc.DefaultQuery("sort", "created")
	skipStr := gc.DefaultQuery("skip", "0")
	limitStr := gc.DefaultQuery("limit", "20")

	skip, err := strconv.Atoi(skipStr)
	if err != nil {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Skip."})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid limit."})
		return
	}

	// TODO Check if sort is valid

	log.Printf("skip: %d, limit: %d, sort: %s.", skip, limit, sort)

	var comments []Comment
	if err := dbcComments.Find(bson.M{"relatedid": review.Id}).Sort(sort).Skip(skip).Limit(limit).All(&comments); err != nil {
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
	gc.JSON(http.StatusOK, gin.H{
		"status":   0,
		"message":  "Successfully fetched Comments.",
		"comments": comments,
	})
	return
}

func insertComment(gc *gin.Context) {
	isLoggedIn, myAccount := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	category := gc.Param("category")

	relatedIdStr := gc.Param("relatedid")
	if bson.IsObjectIdHex(relatedIdStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Param id is invalid objectId."})
		return
	}

	relatedId := bson.ObjectIdHex(relatedIdStr)

	var comment Comment
	if err := gc.BindJSON(&comment); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	if category == "" {
		category = "review" // Default category to review for now
	}

	var review Review

	if category == "review" {
		if err := review.GetById(relatedId); err != nil {
			log.Println(err)
			gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "No Review found matching ObjectId."})
			return
		}
		comment.Category = category
		comment.RelatedId = review.Id
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Unknown category."})
		return
	}

	var userData UserData
	if err := userData.GetById(myAccount.Id); err != nil {
		comment.Nickname = userData.UserId.Hex() // FIXME Replace this
	} else {
		comment.Nickname = userData.Nickname
	}

	comment.UserId = myAccount.Id
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

	if category == "review" {
		if err := review.AddComment(comment.Id); err != nil {
			gc.JSON(http.StatusOK, gin.H{
				"status":  -1,
				"message": "Failed! Insert comment to Review.",
			})
			return
		}
		// TODO Rework on Notification Message
		// Do not notify me commenting on my own review
		if review.UserId != myAccount.Id {
			notification := Notification{
				Id:          bson.NewObjectId(), // Insert a new Notification
				UserId:      review.UserId,      // For user who owns the Review
				Type:        "review_comment",
				Message:     userData.Nickname,
				RelatedType: "review",
				RelatedId:   review.Id,
				IsRead:      false, // It's new and not read.
				IsSent:      false,
			}

			if _, err := notification.Insert(); err != nil {
				log.Print(err)
			}
		}

	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successful! Insert comment info.",
	})
}

func updateComment(gc *gin.Context) {
	isLoggedIn, myAccount := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	//DumpRequestBody(gc)

	commentIdStr := gc.Param("id")
	if bson.IsObjectIdHex(commentIdStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid CommentId. Id: " + commentIdStr})
		return
	}

	commentId := bson.ObjectIdHex(commentIdStr)
	comment := Comment{}
	if err := comment.GetById(commentId); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid CommentId. Id: " + commentIdStr})
		return
	}

	if comment.UserId != myAccount.Id {
		log.Println("User does not own this comment! commentId: " + comment.Id.Hex())
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Owner."})
		return
	}

	var posted Comment
	if err := gc.BindJSON(&posted); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Could not bind."})
		return
	}

	comment.CommentBody = posted.CommentBody
	// TODO Handle ReplyTo and users_mentioned

	if changeInfo, err := comment.Update(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed update comment."})
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Updated " + strconv.Itoa(changeInfo.Updated) + " field(s)."})
	}
	return
}

func deleteComment(gc *gin.Context) {
	isLoggedIn, myAccount := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}
	commentIdStr := gc.Param("id")
	if bson.IsObjectIdHex(commentIdStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid CommentId. Id: " + commentIdStr})
		return
	}

	commentId := bson.ObjectIdHex(commentIdStr)
	comment := Comment{}
	if err := comment.GetById(commentId); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid CommentId. Id: " + commentIdStr})
		return
	}

	if comment.UserId != myAccount.Id {
		log.Println("User does not own this comment! commentId: " + comment.Id.Hex())
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Owner."})
		return
	}

	if err := comment.Delete(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed delete comment."})
		return
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Removed comment."})
		return
	}
	return
}
