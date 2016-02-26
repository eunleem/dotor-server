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

var dbcPushSettings *mgo.Collection

// PushSetting for reviews
type PushSetting struct {
	Id          bson.ObjectId `bson:"_id" json:"-"`
	UserId      bson.ObjectId `bson:"userid" json:"-"`
	GcmToken    string        `json:"token"`
	IsPushOn    bool          `json:"ispushon"`
	GetLikes    bool          `json:"getlikes"`
	GetComments bool          `json:"getcomments"`
	Updated     time.Time     `json:"-"`
}

func ensureIndexesPushSettings() (err error) {
	index := mgo.Index{
		Key:        []string{"userid"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	if err = dbcPushSettings.EnsureIndex(index); err != nil {
		return errors.New("Could not ensure index for PushSettings.")
	}

	return
}

//////////////////////////      PET      /////////////////////////

func (i *PushSetting) Insert() (bson.ObjectId, error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}

	if i.Updated.IsZero() {
		i.Updated = time.Now()
	}

	err := dbcPushSettings.Insert(&i)
	if err != nil {
		log.Println("Could not insert a pushSetting.")
		return i.Id, err
	}

	log.Println("Inserted a pushSetting. UserId: " + i.UserId.Hex())
	log.Println("PushSettingId: " + i.Id.Hex())

	return i.Id, nil
}

func (i *PushSetting) UpsertByUserId(userId bson.ObjectId) (changeInfo *mgo.ChangeInfo, err error) {
	if userId.Valid() == false {
		return nil, errors.New("Invalid UserId")
	}

	i.Updated = time.Now()

	changeInfo, err = dbcPushSettings.Upsert(bson.M{"userid": userId}, &i)
	return

	//changeInfo, err = dbcPushSettings.UpsertId(i.Id, &i)
	//return
}

func (i *PushSetting) Upsert() (changeInfo *mgo.ChangeInfo, err error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}
	if i.UserId.Valid() == false {
		return nil, errors.New("Invalid UserId")
	}

	i.Updated = time.Now()
	log.Printf("Upsert UserId: %s", i.UserId.Hex())

	changeInfo, err = dbcPushSettings.Upsert(bson.M{"userid": i.UserId}, &i)
	return

	//changeInfo, err = dbcPushSettings.UpsertId(i.Id, &i)
	//return
}

func (i *PushSetting) Update() (changeInfo *mgo.ChangeInfo, err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid PushSetting Id")
		return
	}

	changeInfo, err = dbcPushSettings.UpsertId(i.Id, &i)
	return
}

func (i *PushSetting) Delete() (err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid PushSetting Id")
		return
	}

	err = dbcPushSettings.RemoveId(i.Id)
	return
}

func (i *PushSetting) GetByUserId(userId bson.ObjectId) error {
	err := dbcPushSettings.Find(bson.M{"userid": userId}).One(&i)
	if err != nil {
		log.Println("Could not find PushSetting By UserId.")
		return err
	}

	return nil
}

func (i *PushSetting) GetById(id bson.ObjectId) error {
	err := dbcPushSettings.FindId(id).One(&i)
	if err != nil {
		log.Println("Could not find PushSettingById.")
		return err
	}

	return nil
}

func GetTokenByUserId(userId bson.ObjectId) (token string, err error) {
	var push PushSetting
	if err := push.GetByUserId(userId); err != nil {
		return "", err
	}
	return push.GcmToken, nil
}

/////////////////////////    CONTROLLERS    ////////////////////////

func upsertPushSetting(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	DumpRequestBody(gc)

	var pushSetting PushSetting

	err := pushSetting.GetByUserId(user.Id)
	if err != nil {
		// New PushSettings
		pushSetting.Id = bson.NewObjectId()
		pushSetting.IsPushOn = true
		pushSetting.GetLikes = true
		pushSetting.GetComments = true
	}

	if err := gc.BindJSON(&pushSetting); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	pushSetting.UserId = user.Id

	log.Printf("UserId: %s PushSetting: %v %v %v", pushSetting.UserId.Hex(), pushSetting.IsPushOn, pushSetting.GetLikes, pushSetting.GetComments)

	if _, err := pushSetting.Upsert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Failed! Upsert pushSetting info.",
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successful! Upsert PushSetting info.",
	})
}

func insertPushSetting(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	var pushSetting PushSetting
	if err := gc.BindJSON(&pushSetting); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	pushSetting.UserId = user.Id
	pushSetting.IsPushOn = true
	pushSetting.GetLikes = true
	pushSetting.GetComments = true
	pushSetting.Updated = time.Now()

	// #TODO Limit the number of PushSettings per user

	if _, err := pushSetting.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Failed! Insert pushSetting info.",
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successful! Insert pushSetting info.",
	})
}

func updatePushSetting(gc *gin.Context) {
	isLoggedIn, myAccount := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	DumpRequestBody(gc)

	postedPushSetting := PushSetting{}

	if err := gc.BindJSON(&postedPushSetting); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	pushSetting := PushSetting{}

	if err := pushSetting.GetByUserId(myAccount.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "No PushSettings."})
		return
	}

	if pushSetting.UserId != myAccount.Id {
		log.Println("User does not own this pushSetting! pushSettingId: " + pushSetting.Id.Hex())
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Owner."})
		return
	}

	pushSetting.IsPushOn = postedPushSetting.IsPushOn
	pushSetting.GetLikes = postedPushSetting.GetLikes
	pushSetting.GetComments = postedPushSetting.GetComments

	if changeInfo, err := pushSetting.Update(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed update pushSetting."})
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Updated " + strconv.Itoa(changeInfo.Updated) + " field(s)."})
	}
	return
}

func deletePushSetting(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	postedPushSetting := PushSetting{}

	if err := gc.BindJSON(&postedPushSetting); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	pushSetting := PushSetting{}

	if err := pushSetting.GetById(postedPushSetting.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid PushSettingId."})
		return
	}

	if pushSetting.UserId != user.Id {
		log.Println("User does not own this pushSetting!")
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Owner."})
		return
	}

	if err := pushSetting.Delete(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed update pushSetting."})
		return

	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Removed pushSetting."})
	}

	return
}
