package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

var dbcNotifications *mgo.Collection

// Notification for notification
type Notification struct {
	Id          bson.ObjectId `bson:"_id" json:"id"`
	UserId      bson.ObjectId `json:"-"`
	Type        string        `json:"type"`
	Message     string        `json:"message"`
	RelatedType string        `bson:",omitempty" json:"relatedtype"`
	RelatedId   bson.ObjectId `bson:",omitempty" json:"relatedid"`
	IsRead      bool          `json:"isread"`
	IsSent      bool          `json:"-"`
	ReadTime    time.Time     `json:"-"`
	Created     time.Time     `json:"created"`
}

func ensureIndexesNotifications() (err error) {
	index := mgo.Index{
		Key:        []string{"userid"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	if err = dbcNotifications.EnsureIndex(index); err != nil {
		return errors.New("Could not ensure index for Notifications.")
	}

	return
}

////////////////////////     Notifications     ///////////////////////////

//gcmSender := &gcm.Sender(ApiKey: "AIzaSyAOVFr4rSYW0bCt2ISjyBkl-kiQYV1t7S4")

type GcmMessage struct {
	To   string                 `json:"to,omitempty"`
	Data map[string]interface{} `json:"data,omitempty"`
}

func (i *Notification) Insert() (newId bson.ObjectId, err error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}

	if i.Created.IsZero() {
		i.Created = time.Now()
	}

	i.IsRead = false
	//i.IsSent = false
	if len(i.Message) == 0 {
		return i.Id, errors.New("Message cannot be empty")
	}

	if i.Type == "" {
		i.Type = "general"
	}

	if i.RelatedType == "" {
		// Maybe This field is required... FIXME
		i.RelatedType = "general"
	}

	if err = dbcNotifications.Insert(&i); err != nil {
		log.Println("Could not insert a notification.")
		return i.Id, err
	}

	// Check User Push Setting
	pushSetting := PushSetting{
		IsPushOn:    false,
		GetLikes:    false,
		GetComments: false,
	}

	if err := pushSetting.GetByUserId(i.UserId); err != nil {
		log.Println(err)
	}

	log.Printf("UserId: %v PushSetting: %v %v %v",
		i.UserId.Hex(),
		pushSetting.IsPushOn,
		pushSetting.GetLikes,
		pushSetting.GetComments,
	)

	//  Check if this type of push is allowed
	if pushSetting.IsPushOn == false {
		return i.Id, nil
	}

	if pushSetting.GetLikes == false {
		if i.Type == "review_likes" {
			return i.Id, nil
		}
	}

	if pushSetting.GetComments == false {
		if i.Type == "review_comments" {
			return i.Id, nil
		}
	}

	// Prepare to send Push Noti
	token, err := GetTokenByUserId(i.UserId)
	if err != nil {
		log.Print(err)
	}

	log.Println("Token: " + token)

	pushData := map[string]interface{}{"message": i.Type + " " + i.Message}
	gcmMessage := GcmMessage{
		To:   token,
		Data: pushData,
	}

	gcmData, err := json.Marshal(gcmMessage)
	if err != nil {
		log.Print(err)
	}
	log.Printf("ReqBody: %s", gcmData)

	req, err := http.NewRequest("POST", "https://gcm-http.googleapis.com/gcm/send", bytes.NewBuffer(gcmData))
	if err != nil {
		log.Print(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("key=%s", "AIzaSyAOVFr4rSYW0bCt2ISjyBkl-kiQYV1t7S4"))
	req.Header.Add("Content-Type", "application/json")

	httpClient := new(http.Client)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Print(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Println("respStatusCode != OK")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
	}

	log.Printf("ResBody: %s", body)

	//msg := gcm.NewMessage(data)

	log.Println("Inserted a Notification!")
	return i.Id, nil
}

func (i *Notification) Update() (changeInfo *mgo.ChangeInfo, err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Notification Id")
		return
	}

	changeInfo, err = dbcNotifications.UpsertId(i.Id, &i)
	return
}

func (i *Notification) Delete() (err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Notification Id")
		return
	}

	err = dbcNotifications.RemoveId(i.Id)
	return
}

func (i *Notification) GetById(id bson.ObjectId) (err error) {
	if err = dbcNotifications.FindId(id).One(&i); err != nil {
		log.Println("Could not find NotificationById.")
		return
	}

	return
}

func (i *Notification) MarkRead() (err error) {
	if err = dbcNotifications.FindId(i.Id).One(&i); err != nil {
		log.Println("Could not find NotificationById.")
		return
	}

	i.IsRead = true
	i.ReadTime = time.Now()

	if _, err := i.Update(); err != nil {
		return err
	}

	return
}

/////////////////////////    CONTROLLERS   ///////////////////////////

func getNotification(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	idStr := gc.Param("id")

	var notification Notification
	if err := dbcNotifications.FindId(bson.ObjectIdHex(idStr)).One(&notification); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while retrieving notification from DB."})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Successfully fetched Notifications.", "notification": notification})
	return
}

func getMyNotifications(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	skip := 0
	limit := 20

	var notifications []Notification
	err := dbcNotifications.Find(bson.M{"userid": myAccount.Id}).Sort("-created").Skip(skip).Limit(limit).All(&notifications)
	if err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while retrieving data from DB."})
		return
	}

	log.Println("Fetched " + strconv.Itoa(len(notifications)) + " rows.")
	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Successfully fetched My Notifications.", "notifications": notifications})
}

func deleteNotification(gc *gin.Context) {
	// TODO Implement this

}

func readAllNotification(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	if changeInfo, err := dbcNotifications.UpdateAll(bson.M{"userid": myAccount.Id, "isread": false}, bson.M{"isread": true, "readtime": time.Now()}); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed mark all notifications read!"})
		return
	} else {
		log.Println(strconv.Itoa(changeInfo.Updated) + " records updated")
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Marked all notifications read!"})
	return
}

func readNotification(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	notificationIdStr := gc.Param("id")
	// TODO Check if valid
	if bson.IsObjectIdHex(notificationIdStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Param id is invalid objectId."})
		return
	}

	notificationId := bson.ObjectIdHex(notificationIdStr)

	var notification Notification
	if err := notification.GetById(notificationId); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "No Notification found matching ObjectId."})
		return
	}

	if err := notification.MarkRead(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while marking it as read."})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "You Read the notification!"})
	return
}
