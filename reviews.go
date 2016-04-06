package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const TableNameReviews = "reviews"

var dbcReviews *mgo.Collection

func init() {
	const tableName = TableNameReviews
	dbcReviews = dbSession.DB(dbName).C(tableName)
	dbCols[tableName] = dbSession.DB(dbName).C(tableName)

	index := mgo.Index{
		Key:        []string{"userid"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	if err := dbCols[tableName].EnsureIndex(index); err != nil {
		log.Printf("Error while ensuring index for '%s'. %s", tableName, err.Error())
		panic(err)
	}

	// TODO Ensure 2dsphere index
}

// Review for review
type Review struct {
	Id           bson.ObjectId   `bson:"_id" json:"id"`
	UserId       bson.ObjectId   `json:"userid"`
	PetId        bson.ObjectId   `json:"petid" binding:"required"`
	PetType      int             `json:"pet_type"`
	PetAge       int             `json:"pet_age"`
	PetSize      int             `json:"pet_size"`
	HospitalId   bson.ObjectId   `bson:"hospitalid,omitempty" json:"hospitalid,omitempty"`
	HospitalName string          `bson:"hospital,omitempty" json:"hospital_name"`
	Location     GeoJson         `bson:",omitempty" json:"location,omitempty"`
	LocationName string          `bson:",omitempty" json:"location_name"`
	VisitTime    time.Time       `bson:",omitempty" json:"visit_time,omitempty"`
	Category     string          `bson:",omitempty" json:"category,omitempty"`
	Categories   []string        `bson:",omitempty" json:"categories,omitempty"`
	Parts        []string        `bson:",omitempty" json:"parts,omitempty"`
	Cost         int             `bson:",omitempty" json:"cost"`
	ReviewBody   string          `json:"reviewbody" binding:"required"`
	Images       []bson.ObjectId `bson:"images" json:"images,omitempty"`
	Likes        []bson.ObjectId `bson:"likes" json:"likes,omitempty"`
	Comments     []bson.ObjectId `bson:"comments" json:"comments,omitempty"`
	IsDraft      bool            `bson:"isdraft" json:"isdraft"`
	IsSuspended  bool            `bson:",omitempty" json:"isreported"`
	SuspendNote  string          `bson:",omitempty" json:"suspend_note"`
	Suspended    time.Time       `bson:",omitempty" json:"-"`
	IsDeleted    bool            `bson:"isdeleted" json:"-"`
	Deleted      time.Time       `bson:",omitempty" json:"-"`
	Created      time.Time       `json:"created"`
}

////////////////////////      BASIC OPERATIONS     ///////////////////////////

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

	i.Created = time.Now()
	i.IsDeleted = false

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

func (i *Review) DeleteSoft() (err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Review Id")
		return
	}
	i.IsDeleted = true
	i.Deleted = time.Now()

	_, err = i.Update()
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
	if bson.IsObjectIdHex(idStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid id."})
		return
	}

	id := bson.ObjectIdHex(idStr)
	review := Review{
		Id: id,
	}

	if err := review.GetById(review.Id); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error get review from DB."})
		return
	}

	if review.IsDeleted || review.IsDraft {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Found ."})
		return
	} else if review.IsSuspended {
		gc.JSON(http.StatusOK, gin.H{"status": -2, "message": "Suspended."})
		return
	}

	pet := Pet{
		Id: review.PetId,
	}
	if err := pet.GetById(pet.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error get pet from DB"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successfully fetched Review.",
		"review":  review,
		"pet":     pet,
	})
	return
}

func getReviews(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	sort := gc.DefaultQuery("sort", "-created")

	skipStr := gc.DefaultQuery("skip", "0")
	limitStr := gc.DefaultQuery("limit", "20")
	skip, err := strconv.Atoi(skipStr)
	if err != nil {
		skip = 0
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	var reviews []Review
	if err := dbcReviews.Find(bson.M{"isdraft": false, "isdeleted": false}).Sort(sort).Skip(skip).Limit(limit).All(&reviews); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error get data from DB."})
		return
	}

	if len(reviews) == 0 {
		log.Print("No Reviews")
		gc.JSON(http.StatusOK, gin.H{"status": 1, "message": "No reviews."})
		return
	}

	log.Println("Fetched " + strconv.Itoa(len(reviews)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successfully fetched Reviews.",
		"reviews": reviews,
	})
	if buf, err := json.MarshalIndent(reviews, "", "  "); err == nil {
		log.Print(string(buf))
	}
	return
}

type TempId struct {
	Id bson.ObjectId `bson:"_id"`
}

func getReviewsByLocation(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	sort := gc.DefaultQuery("sort", "-created")

	skipStr := gc.DefaultQuery("skip", "0")
	limitStr := gc.DefaultQuery("limit", "100")
	skip, err := strconv.Atoi(skipStr)
	if err != nil {
		skip = 0
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	var posted NearRequest
	if err := gc.Bind(&posted); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Could not Bind.",
		})
		return
	}

	locationStr := fmt.Sprintf("Lat %f Long: %f Dist: %f.", posted.Latitude, posted.Longitude, posted.Distance)
	//log.Printf("Posted lat %f long: %f dist: %f.", posted.Latitude, posted.Longitude, posted.Distance)
	log.Print(locationStr)

	var nearbyHospitalIds []TempId

	if err := dbcHospitals.Find(bson.M{
		"location": bson.M{
			"$nearSphere": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{posted.Longitude, posted.Latitude},
				},
				"$maxDistance": int(posted.Distance),
			},
		},
	}).Limit(20).Select(bson.M{"_id": true}).All(&nearbyHospitalIds); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Could not find hospitals from the location provided.",
		})
		return
	}

	//log.Print("Hospital " + nearbyHospitalIds[0].Id.Hex())

	var array []bson.ObjectId
	array = make([]bson.ObjectId, len(nearbyHospitalIds))
	for i, v := range nearbyHospitalIds {
		array[i] = v.Id
	}

	var reviews []Review
	if err := dbcReviews.Find(bson.M{
		"isdraft":    false,
		"isdeleted":  false,
		"hospitalid": bson.M{"$in": array},
	}).Sort(sort).Skip(skip).Limit(limit).All(&reviews); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error get data from DB."})
		return
	}

	status := 0

	if len(reviews) == 0 {
		log.Print("No Reviews found in that location")

		if err := dbcReviews.Find(bson.M{"isdraft": false, "isdeleted": false}).Sort(sort).Skip(skip).Limit(limit).All(&reviews); err != nil {
			log.Print(err)
			gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error get data from DB."})
			return
		}

		if len(reviews) == 0 {
			gc.JSON(http.StatusOK, gin.H{"status": 1, "message": "No reviews."})
			return
		}
		status = 2
	}

	log.Println("Fetched " + strconv.Itoa(len(reviews)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{
		"status":  status,
		"message": "Successfully fetched Reviews.",
		"reviews": reviews,
	})

	if buf, err := json.MarshalIndent(reviews, "", "  "); err == nil {
		log.Print(string(buf))
	}
	return
}

func getReviewsByCategory(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	sort := gc.DefaultQuery("sort", "-created")
	skipStr := gc.DefaultQuery("skip", "0")
	limitStr := gc.DefaultQuery("limit", "20")
	skip, err := strconv.Atoi(skipStr)
	if err != nil {
		skip = 0
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	//category := gc.Param("category")
	categories := gc.Param("categories")
	categoriesGood := strings.Split(categories, ",")

	var reviews []Review
	if err := dbcReviews.Find(bson.M{
		"isdraft":    false,
		"isdeleted":  false,
		"categories": bson.M{"$in": categoriesGood},
	}).Sort(sort).Skip(skip).Limit(limit).All(&reviews); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error get data from DB."})
		return
	}

	if len(reviews) == 0 {
		log.Print("No Reviews")
		gc.JSON(http.StatusOK, gin.H{"status": 1, "message": "No reviews."})
		return
	}

	log.Println("Fetched " + strconv.Itoa(len(reviews)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successfully fetched Reviews.",
		"reviews": reviews,
	})

	if buf, err := json.MarshalIndent(reviews, "", "  "); err == nil {
		log.Print(string(buf))
	}
	return
}

func getReviewsByPet(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	sort := gc.DefaultQuery("sort", "-created")
	skipStr := gc.DefaultQuery("skip", "0")
	limitStr := gc.DefaultQuery("limit", "20")
	skip, err := strconv.Atoi(skipStr)
	if err != nil {
		skip = 0
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	var posted Pet
	if err := gc.Bind(&posted); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Could not Bind.",
		})
		return
	}

	var reviews []Review
	if err := dbcReviews.Find(bson.M{
		"isdraft":   false,
		"isdeleted": false,
		"pettype":   posted.Type,
		"petage":    posted.Age,
		"petsize":   posted.Size,
	}).Sort(sort).Skip(skip).Limit(limit).All(&reviews); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error get data from DB."})
		return
	}

	if len(reviews) == 0 {
		log.Print("No Reviews")
		gc.JSON(http.StatusOK, gin.H{"status": 1, "message": "No reviews."})
		return
	}

	log.Println("Fetched " + strconv.Itoa(len(reviews)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successfully fetched Reviews.",
		"reviews": reviews,
	})

	if buf, err := json.MarshalIndent(reviews, "", "  "); err == nil {
		log.Print(string(buf))
	}
	return
}

func getMyReviews(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	sort := gc.DefaultQuery("sort", "-created")
	skipStr := gc.DefaultQuery("skip", "0")
	limitStr := gc.DefaultQuery("limit", "20")
	skip, err := strconv.Atoi(skipStr)
	if err != nil {
		skip = 0
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	var reviews []Review
	if err := dbcReviews.Find(bson.M{
		"userid":    myAccount.Id,
		"isdraft":   false,
		"isdeleted": false,
	}).Sort(sort).Skip(skip).Limit(limit).All(&reviews); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while retrieving data from DB."})
		return
	}

	if len(reviews) == 0 {
		gc.JSON(http.StatusOK, gin.H{
			"status":  1,
			"message": "No Reviews.",
		})
		return
	}

	log.Println("Fetched " + strconv.Itoa(len(reviews)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successfully fetched My Reviews.",
		"reviews": reviews,
	})
}

func insertReview(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	var posted Review
	if err := gc.BindJSON(&posted); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Parsing posted JSON failed."})
		return
	}

	pet := Pet{}
	if err := dbcPets.FindId(posted.PetId).One(&pet); err != nil {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Pet."})
		return
	} else {
		if myAccount.Id != pet.UserId {
			gc.JSON(http.StatusOK, gin.H{
				"status":  -1,
				"message": "Cannot post a review for pet that is not yours.",
			})
			return
		}
	}

	posted.UserId = myAccount.Id

	// TODO Filter User Input

	if _, err := posted.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed to insert review to DB."})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Uploaded review!",
		"newid":   posted.Id.Hex(),
	})
}

func updateReview(gc *gin.Context) {

}

func deleteReview(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	var item Review
	if id, err := getIdFromParam(gc); err != nil {
		return

	} else {
		item.Id = id
	}

	if err := item.GetById(item.Id); err != nil {
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "No Reviews Found!",
		})
		return
	}

	if item.UserId != myAccount.Id {
		gc.JSON(http.StatusOK, gin.H{
			"status":  -2,
			"message": "Cannot delete other people's review.",
		})
		return
	}

	if err := item.DeleteSoft(); err != nil {
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Server Error!",
		})
		return
	}

	if changeInfo, err := dbcNotifications.RemoveAll(
		bson.M{
			"relatedtype": "review",
			"relatedid":   item.Id,
		}); err != nil {
		log.Print(err)
	} else {
		log.Printf("Removed %d notifications", changeInfo.Removed)
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successful",
	})
}

func likeReview(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	reviewIdStr := gc.Param("id")
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

	var userData UserData
	if err := userData.GetById(myAccount.Id); err != nil {
		userData.Nickname = "No nickname"
	}

	// TODO Rework Notification Message
	notification := Notification{
		Id:          bson.NewObjectId(), // Insert a new Notification
		UserId:      review.UserId,      // User who owns the Review and see this notification
		Type:        "review_like",
		Message:     userData.Nickname,
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
