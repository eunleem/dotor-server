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

const TableNamePushSettings = "push_settings"

var dbcPushSettings *mgo.Collection

func init() {
	const tableName = TableNamePushSettings

	index := mgo.Index{
		Key:        []string{"userid"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	dbCols[tableName] = dbSession.DB(dbName).C(tableName)
	dbcPushSettings = dbCols[tableName]

	if err := dbCols[tableName].EnsureIndex(index); err != nil {
		log.Printf("Error while ensuring index for '%s'. %s", tableName, err.Error())
		panic(err)
	}
}

// PushSetting for reviews
type PushSetting struct {
	UserId      bson.ObjectId `bson:"_id" json:"-"`
	GcmToken    string        `json:"token"`
	IsPushOn    bool          `json:"ispushon"`
	GetLikes    bool          `json:"getlikes"`
	GetComments bool          `json:"getcomments"`
	Updated     time.Time     `json:"-"`
}

//////////////////////////      PET      /////////////////////////

func (i *PushSetting) Insert() error {
	if i.UserId.Valid() == false {
		log.Println("UserId is required.")
		return errors.New("UserId is required.")
	}

	if i.Updated.IsZero() {
		i.Updated = time.Now()
	}

	if err := dbcPushSettings.Insert(&i); err != nil {
		log.Println("Could not insert a pushSetting.")
		return err
	}

	log.Println("Inserted a pushSetting. UserId: " + i.UserId.Hex())

	return nil
}

func (i *PushSetting) Upsert() (changeInfo *mgo.ChangeInfo, err error) {
	if i.UserId.Valid() == false {
		return nil, errors.New("Invalid UserId")
	}

	i.Updated = time.Now()

	changeInfo, err = dbcPushSettings.UpsertId(i.UserId, &i)
	return

	//changeInfo, err = dbcPushSettings.UpsertId(i.Id, &i)
	//return
}

func (i *PushSetting) Delete() (err error) {
	if i.UserId.Valid() == false {
		err = errors.New("Invalid PushSetting Id")
		return
	}

	err = dbcPushSettings.RemoveId(i.UserId)
	return
}

func (i *PushSetting) GetById(id bson.ObjectId) error {
	err := dbcPushSettings.FindId(id).One(&i)
	if err != nil {
		log.Println("Could not find PushSetting By UserId.")
		return err
	}

	return nil
}

func GetTokenById(id bson.ObjectId) (token string, err error) {
	var push PushSetting
	if err := push.GetById(id); err != nil {
		return "", err
	}
	return push.GcmToken, nil
}

/////////////////////////    CONTROLLERS    ////////////////////////

func upsertPushSetting(gc *gin.Context) {
	isLoggedIn, myAccount := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	pushSetting := PushSetting{
		IsPushOn:    true,
		GetLikes:    true,
		GetComments: true,
	}

	if err := gc.BindJSON(&pushSetting); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	pushSetting.UserId = myAccount.Id

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

func deletePushSetting(gc *gin.Context) {
	isLoggedIn, myAccount := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	pushSetting := PushSetting{
		UserId: myAccount.Id,
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
