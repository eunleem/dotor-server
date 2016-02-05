package main

import (
	"bytes"
	"errors"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

var dbSession *mgo.Session

//var dbcInvitations *mgo.Collection
var dbcUsers *mgo.Collection
var dbcUserData *mgo.Collection
var dbcReceipts *mgo.Collection

//var dbcContents *mgo.Collection
//var dbcEmailVerification *mgo.Collection

type User struct {
	Id                    bson.ObjectId `bson:"_id" json:"id"`
	Email                 string        `bson:",omitempty" json:"email"`
	Username              string        `json:"username"`
	HashedPassword        []byte        `json:"-"`
	Salt                  string        `json:"-"`
	Note                  string        `bson:",omitempty" json:"-"`
	Nickname              string        `bson:",omitempty" json:"nickname"`
	EmailVerificationCode string        `bson:",omitempty" json:"-"`
	IsTemp                bool          `json:"istemp"`
	IsEmailVerified       bool          `json:"-"`
	IsDeleted             bool          `json:"-"`
	IsSuspended           bool          `json:"-"`
	LastLogin             time.Time     `json:"-"`
	LastPasswordChange    time.Time     `json:"-"`
	Created               time.Time     `json:"-"`
}

type Pet struct {
	Name         string    `json:"name" form:"name" binding:"required"`
	Type         string    `json:"type" form:"type" binding:"required"`
	Gender       string    `json:"gender" form:"gender" binding:"required"`
	Age          int       `json:"age" form:"age" binding:"required"`
	Weight       string    `json:"weight" form:"weight" binding:"required"`
	ThumbnailUrl string    `json:"thumbnailurl"`
	PictureUrl   string    `json:"pictureurl"`
	Created      time.Time `json:"created"`
}

type Receipt struct {
	Id             bson.ObjectId `bson:"_id" json:"id"`
	UserId         bson.ObjectId `json:"-"`
	PetName        string        `json:"petname" form:"petname"`
	LocalImagePath string        `json:"imagepath" form:"imagepath"`
	ImageUrl       string        `json:"imageurl"`
	Location       string        `json:"location" form:"location"`
	Hospital       string        `json:"hospital" form:"hospital"`
	Symptoms       string        `json:"symtoms" form:"symptoms"`
	Cost           int           `json:"cost" form:"cost"`
	Review         string        `bson:",omitempty" json:"review" form:"reviewe"`
	Created        time.Time     `json:"created" form:"created"`
	Posted         time.Time     `json:"posted"`
}

type UserData struct {
	UserId     bson.ObjectId   `json:"-"`
	Locality   string          `json:"locality" form:"locality"`
	Hospital   string          `json:"hospital" form:"hospital"`
	Pets       []Pet           `json:"pets" form:"pets"`
	ReceiptIds []bson.ObjectId `json:"receiptids"`
	LastSynced time.Time       `json:"lastsynced" form:"lastsynced"`
}

type Hospital struct {
	Id       bson.ObjectId `bson:"_id" json:"id"`
	Name     string        `json:"name"`
	Locality string        `json:"locality"`
	Address  string        `json:"address"`
}

type Content struct {
	Id         bson.ObjectId   `bson:"_id" json:"id"`
	Title      string          `json:"title"`
	Body       string          `json:"body"`
	Likes      []bson.ObjectId `bson:",omitempty" json:"-"`
	ReceiptIds []bson.ObjectId `bson:",omitempty"`
	Created    time.Time       `json:"created"`
}

func init() {
	OpenCollections()
	err := EnsureIndexes()
	if err != nil {
		log.Fatal(err)
	}
}

func OpenCollections() error {
	const dbName = "Dotor"

	dbSession, err := mgo.Dial("localhost")
	if err != nil {
		fmt.Println("Could not connect to MongoDB!")
		panic(err)
	}

	dbSession.SetMode(mgo.Monotonic, true)

	dbcUsers = dbSession.DB(dbName).C("users")
	//dbcPets = dbSession.DB(dbName).C("Pets")
	dbcUserData = dbSession.DB(dbName).C("userdata")
	dbcReceipts = dbSession.DB(dbName).C("receipts")

	return nil
}

func CloseCollection() {
	dbSession.Close()
}

func EnsureIndexes() (err error) {
	if err = EnsureIndexesUsers(); err != nil {
		return
	}

	if err = EnsureIndexesUserData(); err != nil {
		return
	}

	if err = EnsureIndexesReceipts(); err != nil {
		return
	}
	return
}

func EnsureIndexesUsers() (err error) {
	index := mgo.Index{
		Key:        []string{"username", "email"},
		Unique:     true,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	if err = dbcUsers.EnsureIndex(index); err != nil {
		return errors.New("Could not ensure index for Users.")
	}

	return
}

func EnsureIndexesUserData() (err error) {
	index := mgo.Index{
		Key:        []string{"userid"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	if err = dbcUserData.EnsureIndex(index); err != nil {
		return errors.New("Could not ensure index for UserData.")
	}

	return
}

func EnsureIndexesReceipts() (err error) {
	index := mgo.Index{
		Key:        []string{"userid"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	if err = dbcReceipts.EnsureIndex(index); err != nil {
		return errors.New("Could not ensure index for Receipts.")
	}

	return
}

//////////////////////    USERS    ///////////////////////////

func (i *User) Insert() (bson.ObjectId, error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}

	if i.Created.IsZero() {
		i.Created = time.Now()
	}

	err := dbcUsers.Insert(&i)
	if err != nil {
		log.Println("Could not insert an user.")
		return i.Id, err
	}

	log.Println("Inserted an User. Username: " + i.Username)

	return i.Id, nil

}

func (i *User) GetById(id bson.ObjectId) error {
	err := dbcUsers.FindId(id).One(&i)
	if err != nil {
		log.Println("Could not find User by Id.")
		return err
	}

	return nil
}

func (i *User) GetByUsername(username string) error {
	err := dbcUsers.Find(bson.M{"username": username}).One(&i)
	if err != nil {
		log.Println("Could not find User by Username. Username: " + username)
		return err
	}

	return nil
}

func (i *User) GetByEmail(email string) error {
	err := dbcUsers.Find(bson.M{"email": email}).One(&i)
	if err != nil {
		log.Println("Could not find User by Email. Email: " + email)
		return err
	}

	return nil
}

func (i *User) CheckPassword(password string) bool {
	if i.Username == "" {
		log.Println("Cannot check password when User data is empty!")
		return false
	}

	hashedPassword := HashString(password + i.Salt)

	if bytes.Equal(i.HashedPassword, hashedPassword) == false {
		log.Println("Password does not match. Username: " + i.Username)
		return false
	}

	return true
}

////////////////////////     USER DATA  //////////////////////////

func (i *UserData) GetById(id bson.ObjectId) (err error) {
	if err = dbcUserData.FindId(id).One(&i); err != nil {
		log.Println("Could not find UserDataById. ErrorMsg: " + err.Error())
	}
	return
}

func (i *UserData) Upsert() (changeInfo *mgo.ChangeInfo, err error) {
	changeInfo, err = dbcUserData.UpsertId(i.UserId, &i)
	if err == nil {
		i.LastSynced = time.Now()
		dbcUserData.UpsertId(i.UserId, &i)
	}
	return changeInfo, err
}

////////////////////////     RECEIPT     ///////////////////////////

func (i *Receipt) Insert() (newId bson.ObjectId, err error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}

	if i.Created.IsZero() {
		i.Created = time.Now()
	}

	if err = dbcReceipts.Insert(&i); err != nil {
		log.Println("Could not insert a pet.")
		return i.Id, err
	}

	return i.Id, nil
}

func (i *Receipt) GetById(id bson.ObjectId) (err error) {
	if err = dbcReceipts.FindId(id).One(&i); err != nil {
		log.Println("Could not find ReceiptById.")
		return
	}

	return
}

//////////////////////////      PET      /////////////////////////

/*
func (i *Pet) Insert() (bson.ObjectId, error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}

	if i.Created.IsZero() {
		i.Created = time.Now()
	}

	err := dbcPets.Insert(&i)
	if err != nil {
		log.Println("Could not insert a pet.")
		return i.Id, err
	}

	log.Println("Inserted a pet. OwnerUserId: " + i.OwnerUserId.Hex())

	return i.Id, nil

}

func (i *Pet) GetById(id bson.ObjectId) error {
	err := dbcPets.FindId(id).One(&i)
	if err != nil {
		log.Println("Could not find PetById.")
		return err
	}

	return nil
}
*/
