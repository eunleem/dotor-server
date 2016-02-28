package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"strconv"
	"time"
)

var dbcReviews *mgo.Collection

// Review for review
type Review struct {
	Id       bson.ObjectId   `bson:"_id" json:"id"`
	UserId   bson.ObjectId   `json:"-"`
	PetId    bson.ObjectId   `json:"petid" binding:"required"`
	Location string          `json:"location" binding:"required"`
	Hospital string          `bson:",omitempty" json:"hospital"`
	Category string          `bson:",omitempty" json:"category"`
	Cost     int             `bson:",omitempty" json:"cost"`
	Review   string          `json:"review" binding:"required"`
	Images   []bson.ObjectId `bson:"images" json:"images,omitempty"`
	Receipt  Receipt         `bson:",omitempty" json:"receipt"`
	Likes    []bson.ObjectId `bson:"likes" json:"likes,omitempty"`
	Comments []bson.ObjectId `bson:"comments" json:"comments,omitempty"`
	IsDraft  bool            `bson:"isdraft" json:"isdraft"`
	Created  time.Time       `json:"created"`
}

// Receipt for reviews
type Receipt struct {
	Id       bson.ObjectId   `bson:"_id" json:"id"`
	ReviewId bson.ObjectId   `json:"-"`
	Images   []bson.ObjectId `bson:",omitempty" json:"images,omitempty"`
	Category string          `json:"category"`
	Location string          `json:"location"`
	Hospital string          `json:"hospital"`
	Cost     int             `json:"cost"`
	Created  time.Time       `json:"created"`
	Posted   time.Time       `json:"posted"`
}

func ensureIndexesReviews() (err error) {
	index := mgo.Index{
		Key:        []string{"userid"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	if err = dbcReviews.EnsureIndex(index); err != nil {
		return errors.New("Could not ensure index for Reviews.")
	}

	return
}

////////////////////////     REVIEWS     ///////////////////////////

func (i *Review) Like(userId bson.ObjectId) (alreadyLiked bool, err error) {
	if err := i.GetById(i.Id); err != nil {
		return false, err
	}

	n, err := dbcReviews.Find(bson.M{"_id": i.Id, "likes": bson.M{"$in": []bson.ObjectId{userId}}}).Count()
	if err != nil {
		return false, err
	}

	if n > 0 {
		return true, nil
	}

	i.Likes = append(i.Likes, userId)
	_, err = i.Update()

	if err != nil {
		log.Println("Erro while liking review.")
		return false, err
	}
	return false, err
}

func (i *Review) AddComment(commentId bson.ObjectId) (err error) {
	if err := i.GetById(i.Id); err != nil {
		return err
	}

	i.Comments = append(i.Comments, commentId)
	_, err = i.Update()

	if err != nil {
		log.Println("Added a comment to the review.")
		return
	}
	return
}

func (i *Review) Insert() (newId bson.ObjectId, err error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}

	if i.Created.IsZero() {
		i.Created = time.Now()
	}

	if err = dbcReviews.Insert(&i); err != nil {
		log.Println("Could not insert a review.")
		return i.Id, err
	}

	return i.Id, nil
}

func (i *Review) Update() (changeInfo *mgo.ChangeInfo, err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Review Id")
		return
	}

	changeInfo, err = dbcReviews.UpsertId(i.Id, &i)
	return
}

func (i *Review) Delete() (err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Review Id")
		return
	}

	err = dbcReviews.RemoveId(i.Id)
	return
}

func (i *Review) GetById(id bson.ObjectId) (err error) {
	if err = dbcReviews.FindId(id).One(&i); err != nil {
		log.Println("Could not find ReviewById.")
		return
	}

	return
}

/////////////////////////    CONTROLLERS   ///////////////////////////

func getReview(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	idStr := gc.Param("id")

	var review Review
	if err := dbcReviews.FindId(bson.ObjectIdHex(idStr)).One(&review); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while retrieving review from DB."})
		return
	}

	var pet Pet
	if err := dbcPets.FindId(review.PetId).One(&pet); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while retrieving pet from DB"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Successfully fetched My Reviews.", "review": review, "pet": pet})
	return
}

func getReviews(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	skip := 0
	limit := 50

	var reviews []Review
	if err := dbcReviews.Find(bson.M{"isdraft": false}).Sort("-created").Skip(skip).Limit(limit).All(&reviews); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while retrieving data from DB."})
		return
	}

	if len(reviews) == 0 {
		log.Print("No Reviews")
		gc.JSON(http.StatusOK, gin.H{"status": -2, "message": "No reviews."})
		return
	}

	log.Println("Fetched " + strconv.Itoa(len(reviews)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Successfully fetched My Reviews.", "data": reviews})
	return
}

func getReviewsByRegion(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	//qType := gc.Query("type")
	qRegion := gc.Query("region")

	dbQuery := bson.M{"region": "Seoul"}

	if qRegion != "" {
		dbQuery = bson.M{"region": qRegion}
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "QueryString 'region' is required."})
		return
	}

	skip := 0
	limit := 20

	var reviews []Review
	if err := dbcReviews.Find(dbQuery).Sort("-created").Skip(skip).Limit(limit).All(&reviews); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while retrieving data from DB."})
		return
	}

	log.Println("Fetched " + strconv.Itoa(len(reviews)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Successfully fetched My Reviews.", "data": reviews})
}

func getMyReviews(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	skip := 0
	limit := 20

	var reviews []Review
	err := dbcReviews.Find(bson.M{"userid": myAccount.Id}).Sort("-created").Skip(skip).Limit(limit).All(&reviews)
	if err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while retrieving data from DB."})
		return
	}

	log.Println("Fetched " + strconv.Itoa(len(reviews)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Successfully fetched My Reviews.", "data": reviews})
}

func insertReview(gc *gin.Context) {
	loggedIn, user := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	var posted Review
	if err := gc.BindJSON(&posted); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Parsing posted JSON failed."})
		return
	}

	posted.UserId = user.Id

	if newId, err := posted.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed to insert review to DB."})
		return
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Uploaded review!", "newid": newId.Hex()})
		return
	}
}

func updateReview(gc *gin.Context) {

}

func deleteReview(gc *gin.Context) {

}

func likeReview(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	reviewIdStr := gc.Param("id")
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

	if alreadyLiked, err := review.Like(myAccount.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while updating DB!"})
		return

	} else {
		if alreadyLiked == true {
			gc.JSON(http.StatusOK, gin.H{"status": 1, "message": "Your already liked it!"})
			return
		}
	}

	if myAccount.Id == review.UserId {
		gc.JSON(http.StatusOK, gin.H{"status": 2, "message": "You liked your own review!"})
		return
	}

	notification := Notification{
		Id:          bson.NewObjectId(), // Insert a new Notification
		UserId:      review.UserId,      // User who owns the Review and see this notification
		Type:        "review_likes",
		Message:     fmt.Sprintf("'%s' likes your review!", myAccount.Nickname),
		RelatedType: "review",
		RelatedId:   review.Id,
		IsRead:      false, // It's new and not read.
		IsSent:      false,
	}

	if _, err := notification.Insert(); err != nil {
		log.Print(err)
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "You Liked the review!"})
	return
}
