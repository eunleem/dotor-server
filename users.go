package main

import (
	"bytes"
	"errors"
	valid "github.com/asaskevich/govalidator"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"strconv"
	"time"
)

var dbcUsers *mgo.Collection
var dbcUserData *mgo.Collection

//var dbcUserPref *mgo.Collection

// User sctur
type User struct {
	Id                    bson.ObjectId `bson:"_id" json:"id"`
	Email                 string        `bson:",omitempty" json:"email" form:"email"`
	Username              string        `json:"username"`
	HashedPassword        []byte        `json:"-"`
	Salt                  string        `json:"-"`
	EmailVerificationCode string        `bson:",omitempty" json:"-"`
	Nickname              string        `bson:",omitempty" json:"nickname" form:"nickname"`
	IsTemp                bool          `json:"istemp"`
	IsEmailVerified       bool          `json:"-"`
	IsDeleted             bool          `json:"-"`
	IsSuspended           bool          `json:"-"`
	AccountType           string        `bson:",omitempty" json:"-"`
	LastLogin             time.Time     `json:"-"`
	LastPasswordChange    time.Time     `json:"-"`
	Created               time.Time     `json:"-"`
	Note                  string        `bson:",omitempty" json:"-"`
}

type UserData struct {
	UserId     bson.ObjectId `bson:"_id" json:"-"`
	Locality   string        `json:"locality" form:"locality"`
	Hospital   string        `json:"hospital" form:"hospital"`
	LastSynced time.Time     `json:"lastsynced" form:"lastsynced"`
}

// Maybe Unnecessary
type RegistrationInfo struct {
	UserId    bson.ObjectId `bson:"_id" json:"-"`
	IpAddress string        `bson:",omitempty"`
	Created   time.Time     `json:"-"`
}

func ensureIndexesUsers() (err error) {
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

func (i *User) Update() (changeInfo *mgo.ChangeInfo, err error) {
	changeInfo, err = dbcUsers.UpsertId(i.Id, &i)
	if err != nil {
		log.Println("Could not update an user. " + err.Error())
	}
	return changeInfo, err
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

	hashed := HashString(password + i.Salt)

	if bytes.Equal(i.HashedPassword, hashed) == false {
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

/////////////////// ETC /////////////////////

func isEmailUnique(email string) (isUnique bool, err error) {
	isUnique = true
	if valid.IsEmail(email) == false {
		log.Print("Not a valid email address!")
		return false, errors.New("Not a valid email address!")
	}

	count, err := dbcUsers.Find(bson.M{"email": email}).Count()
	if count > 0 {
		isUnique = false
	}
	return
}

func isNicknameUnique(nickname string) (isUnique bool, err error) {
	isUnique = true

	if isValidNickname(nickname) == false {
		log.Print("Not a valid nickname!")
		return false, errors.New("Not a valid nickname!")
	}

	count, err := dbcUsers.Find(bson.M{"nickname": nickname}).Count()
	if count > 0 {
		isUnique = false
	}
	return
}

func isValidNickname(nickname string) bool {
	// #TODO Implement this function in detail
	if len(nickname) > 4 {
		return true
	}
	return false
}

////////////////    CONTROLLERS    /////////////////

func register(gc *gin.Context) {
	username := RandStringBytesMaskImprSrc(12)
	password := RandStringBytesMaskImprSrc(12)
	salt := RandStringBytesMaskImprSrc(6)

	password += salt

	user := User{
		Username:           username, // Temp Username
		HashedPassword:     HashString(password + salt),
		Salt:               salt,
		IsTemp:             true,
		IsEmailVerified:    false,
		IsDeleted:          false,
		IsSuspended:        false,
		LastLogin:          time.Now(),
		LastPasswordChange: time.Now(),
		Created:            time.Now(),
	}

	if _, err := user.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Could not insert User.",
		})
		return
	}

	session := sessions.Default(gc)
	session.Set("userid", user.Id.Hex())
	if err := session.Save(); err != nil {
		log.Println("Error saving session. " + err.Error())
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Server Error!",
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0,
		"message":  "Successfully added an user. ",
		"username": user.Username,
		"password": password,
	})
}

type LoginForm struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func login(gc *gin.Context) {
	session := sessions.Default(gc)
	userid := session.Get("userid")
	if userid != nil {
		gc.JSON(http.StatusOK, gin.H{
			"status":  1,
			"message": "Already Logged in!",
		})
		return
	}

	var json LoginForm
	gc.Bind(&json)

	username := json.Username //gc.PostForm("username")
	password := json.Password //gc.PostForm("password")

	user := User{}

	if err := user.GetByUsername(username); err != nil {
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Login Failed.",
		})
		return
	}

	authenticated := user.CheckPassword(password)
	if authenticated == false {
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Login Failed.",
		})
		return
	}

	session.Set("userid", user.Id.Hex())
	if err := session.Save(); err != nil {
		log.Println("Error saving session. " + err.Error())
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Server Error!",
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Welcome! Now you are logged in!",
	})

	user.LastLogin = time.Now()
	user.Update()
	return
}

type CheckUserInfoForm struct {
	Email    string `form:"email"`
	Nickname string `form:"nickname"`
	Locality string `form:"locality"`
	Hospital string `form:"hospital"`
}

func updateUser(gc *gin.Context) {
	loggedIn, user := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	var posted CheckUserInfoForm
	gc.Bind(&posted)

	if posted.Email != "" {
		if user.Email != posted.Email {
			if isUnique, err := isEmailUnique(posted.Email); err == nil && isUnique == true {
				log.Println("Update Email from " + user.Email + " to " + posted.Email)
				user.Email = posted.Email
			}
		}
	} else {
		log.Println("Posted Email is null.")
	}

	if posted.Nickname != "" {
		if user.Nickname != posted.Nickname {
			if isUnique, err := isNicknameUnique(posted.Nickname); err == nil && isUnique == true {
				log.Println("Update Nickname from " + user.Nickname + " to " + posted.Nickname)
				user.Nickname = posted.Nickname
			}
		}
	} else {
		log.Println("Posted Nickname is null.")
	}

	//#TODO Update Locality and Hospital

	if changeInfo, err := user.Update(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error updating User Information. err: " + err.Error()})
	} else {
		log.Println(strconv.Itoa(changeInfo.Updated) + " field(s) have been updated.")
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "User Information Updated"})
}

// Use this only for before registration. NOT FOR UPDATE.
func checkUserInfo(gc *gin.Context) {
	loggedIn, user := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	var posted CheckUserInfoForm
	gc.Bind(&posted)

	if isUnique, err := isEmailUnique(posted.Email); err != nil || isUnique == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not a unique or valid email."})
		return
	}

	user.Email = posted.Email

	if isUnique, err := isNicknameUnique(posted.Nickname); err != nil || isUnique == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not a unique or valid email."})
		return
	}
	user.Nickname = posted.Nickname

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Requested information is valid."})
}
