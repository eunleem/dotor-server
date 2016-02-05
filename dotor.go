package main

import (
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"
)

func main() {

	router := gin.Default()

	OpenCollections()
	defer CloseCollection()

	store, _ := sessions.NewRedisStore(10, "tcp", "localhost:6379", "xlaVkfVkfgo", []byte("secret"))
	router.Use(sessions.Sessions("dotor_session", store))

	router.Static("/img", "./file/img")
	router.Static("/file", "./file")
	router.StaticFile("/favicon.ico", "./file/favicon.ico")
	router.StaticFile("/", "./file/index.html")

	router.StaticFile("/test", "./file/test.html")

	router.GET("/test/json", testjson)
	router.GET("/test/session", testsession)
	//router.GET("/project/:id", getProject)

	router.POST("/register", register)
	router.POST("/login", login)

	//router.POST("/addpet", insertPet)

	router.POST("/sync", sync)
	//router.POST("/project/update", updateProject)

	//router.GET("/journals", getJournalsHandler)
	//router.GET("/journal/:id", getJournalHandler)
	//router.POST("/journal/insert", postJournalHandler)
	//router.POST("/journal/update", postJournalHandler)

	//router.GET("/content/:id", getContentHandler)

	router.Run(":8080")
}

func register(gc *gin.Context) {
	username := RandStringBytesMaskImprSrc(12)
	password := RandStringBytesMaskImprSrc(12)
	salt := RandStringBytesMaskImprSrc(6)

	password += salt

	i := User{
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

	if _, err := i.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Could not insert User.",
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0,
		"message":  "Successfully added an user. ",
		"username": i.Username,
		"password": password,
	})
}

type LoginForm struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func login(gc *gin.Context) {
	req, er := httputil.DumpRequest(gc.Request, true)
	if er != nil {
		log.Println("Error")
		return
	}

	log.Print(string(req[:]))

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
	return
}

func sync(gc *gin.Context) {
	session := sessions.Default(gc)
	useridStr := session.Get("userid")
	if useridStr == nil {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged in!"})
		return
	}

	log.Println("User Session is found! userid:" + useridStr.(string))

	user := User{}
	if err := user.GetById(bson.ObjectIdHex(useridStr.(string))); err != nil {
		log.Println("Session exists but User data is not in DB! Session cleared! " + err.Error())
		session.Clear()
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Server Error!"})
		return
	}

	var posted UserData
	if err := gc.BindJSON(&posted); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Parsing posted JSON failed."})
		return
	}

	log.Println(posted.LastSynced)

	var userData UserData
	if err := userData.GetById(user.Id); err != nil {
		// Assume it's a new row
		userData.UserId = user.Id
		userData.Locality = posted.Locality
		userData.Hospital = posted.Hospital
		userData.Pets = posted.Pets
		userData.LastSynced = time.Now()

	} else {

		log.Println(posted.LastSynced.Unix())
		log.Println(userData.LastSynced.Unix())

		if posted.LastSynced.Unix() == userData.LastSynced.Unix() {
			gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Already Synced! Good!"})
			return

		} else if posted.LastSynced.Unix() > userData.LastSynced.Unix() {
			userData.Locality = posted.Locality
			userData.Hospital = posted.Hospital
			userData.Pets = posted.Pets
			userData.LastSynced = time.Now()

		} else {
			gc.JSON(http.StatusOK, gin.H{"status": 1, "message": "Overwriting Client Data.", "data": userData})
			return

		}
	}

	log.Print(userData)

	log.Println("About to upsert")
	if changeInfo, err := userData.Upsert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed to upsert userdate to db."})
		return

	} else {
		log.Print("Updated: " + strconv.Itoa(changeInfo.Updated))
		log.Print("Removed: " + strconv.Itoa(changeInfo.Removed))
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Uploaded client data to server. Good!", "lastsynced": userData.LastSynced})
	return
}

func addReceipt(gc *gin.Context) {
	session := sessions.Default(gc)
	useridStr := session.Get("userid")
	if useridStr == nil {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged in!"})
		return
	}

	log.Println("User Session is found! userid:" + useridStr.(string))

	user := User{}
	if err := user.GetById(bson.ObjectIdHex(useridStr.(string))); err != nil {
		log.Println("Session exists but User data is not in DB! Session cleared! " + err.Error())
		session.Clear()
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Server Error!"})
		return
	}

	var posted Receipt
	if err := gc.BindJSON(&posted); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Parsing posted JSON failed."})
		return
	}

	posted.UserId = user.Id

	if newId, err := posted.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed to upsert receipt to db."})
		return

	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Uploaded receipt!", "newid": newId.Hex()})
		return

	}

}

/*
func insertPet(gc *gin.Context) {
	session := sessions.Default(gc)
	useridStr := session.Get("userid")
	if useridStr == nil {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged in!"})
		return
	}

	log.Println("User Session is found! userid:" + useridStr.(string))

	user := User{}
	if err := user.GetById(bson.ObjectIdHex(useridStr.(string))); err != nil {
		log.Println("Session exists but User data is not in DB! Session cleared! " + err.Error())
		session.Clear()
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Server Error!"})
		return
	}


	if err := gc.BindJSON(&pet); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	var err error

	newId, err := pet.Insert()
	if err != nil {
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Failed! Insert pet info.",
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successful! Insert pet info.",
		"petid":   newId.Hex(),
	})
}
*/

func updateUser(gc *gin.Context) {

}
