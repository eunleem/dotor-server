package main

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/smtp"
	"time"

	valid "github.com/asaskevich/govalidator"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const TableNameUsers = "users"
const TableNameUserData = "user_data"

var dbcUsers *mgo.Collection
var dbcUserData *mgo.Collection

func init() {
	const tableName = TableNameUsers

	index := mgo.Index{
		Key:        []string{"username", "email"},
		Unique:     true,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	dbCols[tableName] = dbSession.DB(dbName).C(tableName)
	dbCols[TableNameUserData] = dbSession.DB(dbName).C(TableNameUserData)
	dbcUsers = dbCols[tableName]
	dbcUserData = dbCols[TableNameUserData]

	if err := dbCols[tableName].EnsureIndex(index); err != nil {
		log.Printf("Error while ensuring index for '%s'. %s", tableName, err.Error())
		panic(err)
	}

	// No need for ensure index for UserData collection
}

const UserGroupTemp = 0
const UserGroupNormal = 100
const UserGroupAdmin = 10000

type User struct {
	Id                    bson.ObjectId `bson:"_id,omitempty" json:"id"`
	Email                 string        `bson:",omitempty" json:"email" form:"email"`
	Username              string        `json:"username"`
	HashedPassword        []byte        `json:"-"`
	Salt                  string        `json:"-"`
	EmailVerificationCode string        `bson:",omitempty" json:"-"`
	IsTemp                bool          `json:"istemp"`
	IsEmailVerified       bool          `json:"-"`
	IsDeleted             bool          `json:"-"`
	IsSuspended           bool          `json:"-"`
	UserGroup             int           `json:"-"`
	LastLogin             time.Time     `json:"-"`
	LastPasswordChange    time.Time     `json:"-"`
	Created               time.Time     `json:"-"`
	Note                  string        `bson:",omitempty" json:"-"`
}

type UserData struct {
	UserId         bson.ObjectId `bson:"_id" json:"-"`
	Nickname       string        `bson:",omitempty" json:"nickname" form:"nickname"`
	ProfileImageId bson.ObjectId `bson:",omitempty" json:"imageid,omitempty"`
	Updated        time.Time     `json:"updated,omitempty"`
	//MyHospitalId        bson.ObjectId   `bson:",omitempty" json:"myhospitalid"`
	//FavoriteHospitalIds []bson.ObjectId `bson:",omitempty" json:"favorite_hospitals"`
	//HomeLocation        GeoJson         `bson:",omitempty" json:"home_location"`
	//HomeAddress         string          `bson:",omitempty" json:"home_address"`
}

////////////////////////    BASIC OPERATIONS    ///////////////////////////

func (i *User) Insert() (bson.ObjectId, error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}

	if len(i.Username) < 4 {
		errMsg := "Username must be longer than 4 chars."
		log.Print(errMsg)
		return i.Id, errors.New(errMsg)
	}

	if i.Created.IsZero() {
		i.Created = time.Now()
	}

	if err := dbcUsers.Insert(&i); err != nil {
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

func (i *User) GetById(id bson.ObjectId) (err error) {
	if err := dbcUsers.FindId(id).One(&i); err != nil {
		log.Println("Could not find User by Id.")
		return err
	}
	return
}

func (i *User) GetByUsername(username string) (err error) {
	if err := dbcUsers.Find(bson.M{"username": username}).One(&i); err != nil {
		log.Println("Could not find User by Username. Username: " + username)
		return err
	}
	return
}

func (i *User) GetByEmail(email string) (err error) {
	if err := dbcUsers.Find(bson.M{"email": email}).One(&i); err != nil {
		log.Println("Could not find User by Email. Email: " + email)
		return err
	}
	return
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
	if err := dbcUserData.FindId(id).One(&i); err != nil {
		log.Println("Could not find UserDataById. ErrorMsg: " + err.Error())
		return err
	}
	return
}

func (i *UserData) Upsert() (changeInfo *mgo.ChangeInfo, err error) {
	if i.UserId.Valid() == false {
		log.Println("Cannot upsert userdata without proper userid")
		return
	}
	i.Updated = time.Now()
	if changeInfo, err := dbcUserData.UpsertId(i.UserId, &i); err != nil {
		return changeInfo, err
	}
	return changeInfo, err
}

///////////////////    ADDITIONAL   /////////////////////

func isEmailUnique(email string) (isUnique bool, err error) {
	isUnique = true
	if valid.IsEmail(email) == false {
		log.Print("Not a valid email address!")
		return false, errors.New("Not a valid email address!")
	}

	n, err := dbcUsers.Find(bson.M{"email": email}).Count()
	if n > 0 {
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
	if len(nickname) > 2 {
		return true
	}
	return false
}

func (i *User) UpdateEmail(email string) (err error) {
	if i.Id.Valid() == false {
		log.Print("User not loaded.")
		return errors.New("User not loaded.")
	}

	if i.Email == email {
		log.Print("Same email .")
		return errors.New("Email is same.")
	}

	if isUnique, err := isEmailUnique(email); err != nil || isUnique == false {
		log.Print("Not a unique email.")
		return errors.New("Not a unique Email")
	}

	i.Email = email
	i.IsEmailVerified = false
	i.EmailVerificationCode = RandomNumberString(6)
	if _, err := i.Update(); err != nil {
		log.Print("Could not update User with new Verification Code.")
		return errors.New("Could not update user with new Verification Code.")
	}

	emailBody := "/user/verify/" + i.Email + "/" + i.EmailVerificationCode
	sendEmail(i.Email, emailBody)

	return nil
}

func sendEmail(to string, body string) {
	from := "team88.master@gmail.com"
	password := "go88ckaclcjfja"

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: Email Verification for Dotor\n\n" +
		body

	if err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, password, "smtp.gmail.com"),
		from, []string{to}, []byte(msg)); err != nil {

		log.Printf("smtp error: %s", err)
		return
	}
	log.Print("Email Sent")
}

////////////////    CONTROLLERS    /////////////////

func verifyEmail(gc *gin.Context) {

	email := gc.Param("email")
	code := gc.Param("code")

	var user User
	if err := user.GetByEmail(email); err != nil {
		log.Print("Could not get user by email. email: " + email)
		gc.File("./web/email_verify_failed.html")
		return
	}

	if user.IsDeleted || user.IsSuspended {
		log.Print("User Account is deleted or suspended.")
		gc.File("./web/email_verify_failed.html")
		return
	}

	if user.EmailVerificationCode != code {
		log.Print("Verification code does not match. email: " + email + " code: " + code)
		gc.File("./web/email_verify_failed.html")
		return
	}

	user.IsTemp = false
	user.IsEmailVerified = true
	user.EmailVerificationCode = ""

	if _, err := user.Update(); err != nil {
		log.Print("Internal Server Error")
		gc.File("./web/email_verify_failed.html")
		return
	}

	log.Print("Email Verified.")
	gc.File("./web/email_verify_success.html")

	return
}

func registerTemp(gc *gin.Context) {
	username := RandStringBytesMaskImprSrc(12)
	password := RandStringBytesMaskImprSrc(12)
	salt := RandStringBytesMaskImprSrc(6)

	password += salt

	user := User{
		Id:                 bson.NewObjectId(),
		Username:           username, // Temp Username
		HashedPassword:     HashString(password + salt),
		Salt:               salt,
		IsTemp:             true,
		IsEmailVerified:    false,
		IsDeleted:          false,
		IsSuspended:        false,
		UserGroup:          UserGroupTemp,
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

	// LOGIN
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
		"message":  "Registration for an temp account successful. ",
		"username": user.Username,
		"password": password,
	})
}

type LoginRequest struct {
	Username string `form:"username"`
	Email    string `form:"email"`
	Password string `form:"password" binding:"required"`
}

func login(gc *gin.Context) {
	DumpRequestBody(gc)
	isLoggedIn, myAccount := isLoggedInNoAutoResp(gc)
	if isLoggedIn == true {
		AlreadyLoggedInSlideExpiry(gc)
		return
	}

	var json LoginRequest
	if err := gc.Bind(&json); err != nil {
		ErrorBinding(gc)
		return
	}

	user := User{}
	if len(json.Password) < 4 {
		MissingRequiredValue(gc, "password")
		return
	}

	if len(json.Username) > 4 {
		if err := user.GetByUsername(json.Username); err != nil {
			gc.JSON(http.StatusOK, gin.H{
				"status":  -1,
				"message": "Login Failed.",
			})
			return
		}

	} else if len(json.Email) > 6 {
		if err := user.GetByEmail(json.Email); err != nil {
			gc.JSON(http.StatusOK, gin.H{
				"status":  -1,
				"message": "Login Failed.",
			})
			return
		}

	} else {
		MissingRequiredValue(gc, "username", "password")
		return
	}

	if authenticated := user.CheckPassword(json.Password); authenticated == false {
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Login Failed.",
		})
		return
	}

	session := sessions.Default(gc)
	session.Set("userid", user.Id.Hex())
	if err := session.Save(); err != nil {
		log.Println("Error saving session. " + err.Error())
		ServerError(gc)
		return
	}

	status := 0
	if myAccount.IsEmailVerified == true {
		status = 2
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  status,
		"message": "Welcome! Now you are logged in!",
	})

	user.LastLogin = time.Now()
	user.Update()
	return
}

func getUser(gc *gin.Context) {
	if isLoggedIn, _ := isLoggedIn(gc); isLoggedIn == false {
		return
	}

	idStr := gc.Param("id")
	if bson.IsObjectIdHex(idStr) == false {
		log.Println("Invalid id.")
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Id."})
		return
	}
	id := bson.ObjectIdHex(idStr)
	i := User{}
	if err := i.GetById(id); err != nil {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while getting item."})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Success.", "user": i})
	return
}

func softDeleteUser(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	myAccount.IsDeleted = true

	if _, err := myAccount.Update(); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Error while getting item.",
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successfully soft deleted user.",
	})
}

func updateEmail(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	var posted User
	posted.Email = gc.Param("email")
	log.Println("Email: " + posted.Email)

	// TODO Checking only if it is unique is not enough.
	// Needs to check if that email address is verified.
	// If not verified, Give user to resend verification code.
	// or claim it theirs.

	if posted.Email != "" {
		if myAccount.Email != posted.Email {
			if err := myAccount.UpdateEmail(posted.Email); err == nil {
				log.Println("Update Email to " + posted.Email)
			} else if err != nil {
				gc.JSON(http.StatusOK, gin.H{
					"status":  -2,
					"message": "Email is already registered.",
				})
				return

			}
		}
	} else {
		log.Println("Posted Email is null.")
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Posted Email is null.",
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "User Information Updated"})
	return
}

func updateNickname(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	existing := UserData{}
	existing.UserId = myAccount.Id

	if err := existing.GetById(myAccount.Id); err != nil {
		if _, err := existing.Upsert(); err != nil {
			log.Println("Error while UserData Upserting. " + err.Error())
			gc.JSON(http.StatusOK, gin.H{
				"status":  -1,
				"message": "Error while upserting data.",
			})
			return
		} else {
			log.Println("Inserted userdata.")
		}
	}

	DumpRequestBody(gc)

	posted := UserData{}
	posted.Nickname = gc.Param("nickname")

	if posted.Nickname != "" {
		if existing.Nickname != posted.Nickname {
			if isUnique, err := isNicknameUnique(posted.Nickname); err == nil && isUnique == true {
				log.Println("Update Nickname  to " + posted.Nickname)
				existing.Nickname = posted.Nickname

			} else if err != nil {
				log.Print("Error Checking isUnique")
			} else if isUnique == false {
				gc.JSON(http.StatusOK, gin.H{
					"status":  -1,
					"message": "Nickname is already in use.",
				})
				return
			}
		}

	} else {
		log.Println("Posted Nickname is null.")
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Nothing to update.",
		})
		return
	}

	/*if posted.MyHospitalId != nil {
		// Make sure HospitalId exists...
		if dbcHospitals.FindId(bson.M{"_id": posted.MyHospitalId}).Count() > 0 {
			existing.MyHospitalId = posted.MyHospitalId
		} else {
				gc.JSON(http.StatusOK, gin.H{
					"status":  -1,
					"message": "Invalid hospitalid.",
				})
				return
		}
	}*/

	if _, err := existing.Upsert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Error updating User Information. err: " + err.Error(),
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "User Information Updated"})
	return
}

// Use this only for before registration. NOT FOR UPDATE.
func checkEmail(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	email := gc.Query("email")

	if isUnique, err := isEmailUnique(email); err != nil || isUnique == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not a unique or valid email."})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Email is usable."})
}

func checkNickname(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	nickname := gc.Query("nickname")

	if isUnique, err := isNicknameUnique(nickname); err != nil || isUnique == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not a unique or valid Nickname."})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Nickname is usable."})
}
