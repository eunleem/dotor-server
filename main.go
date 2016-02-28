package main

import (
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {

	router := gin.Default()

	defer CloseDb()

	store, _ := sessions.NewRedisStore(10, "tcp", "localhost:6379", "xlaVkfVkfgo", []byte("secret"))
	router.Use(sessions.Sessions("dotor_session", store))

	newReditClient()

	rootDir := "./web"

	router.Static("/img", (rootDir + "/img"))

	//router.StaticFile("/", (rootDir + "/index.html"))
	//router.StaticFile("/favicon.ico", (rootDir + "/favicon.ico"))

	router.POST("/register", register) // Deprecated TODO
	router.POST("/login", login)       // Deprecated
	router.POST("/sync", sync)         // Deprecated

	router.POST("/status", checkServerStatus) // Deprecated

	router.GET("/server/status", checkServerStatus)

	router.POST("/user/check", checkUserInfo) // are Nickname and email valid and unique?

	router.POST("/user/register", register)
	router.POST("/user/login", login)

	router.POST("/user/update", updateUser)

	router.POST("/pet/get/:id", getPet)

	router.POST("/pet/insert", insertPet)
	router.POST("/pet/update", updatePet)
	router.POST("/pet/delete", deletePet)

	router.GET("/review/:id", getReview)

	router.POST("/review/insert", insertReview)
	router.POST("/review/update", updateReview)
	router.POST("/review/delete", deleteReview)

	router.POST("/review/like/:id", likeReview)

	router.GET("/reviews/all", getReviews)
	router.GET("/reviews/my", getMyReviews)
	router.GET("/reviews/region/:region", getReviewsByRegion)

	router.POST("/image/insert", insertImage)
	router.GET("/image/:id", getImage)

	router.GET("/notification/:id", getNotification)

	router.GET("/notifications", getMyNotifications)

	router.POST("/notification/readall", readAllNotification)
	router.POST("/notification/read/:id", readNotification)
	router.POST("/notification/received/:id", receivedNotification)

	router.POST("/comment/insert/:reviewid", insertComment)
	router.GET("/comments/:reviewid", getComments)
	//router.POST("/comment/like/:id", likeComment)

	router.POST("/settings/push/insert", insertPushSetting)
	router.POST("/settings/push/update", updatePushSetting)
	router.POST("/settings/push/upsert", upsertPushSetting)

	router.POST("/feedback/insert", insertFeedback)
	//router.POST("/feedback/update", updateFeedback)
	//router.POST("/feedback/delete", deleteFeedback)

	router.POST("/reset", reset)

	router.Run(":8088")
}

func reset(gc *gin.Context) {
	DropDb()
	resetRedis()
	os.RemoveAll("./web/img")
	os.Mkdir("./web/img", 0775)
	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "RESET DB!"})
}

func checkServerStatus(gc *gin.Context) {
	// #TODO Report finer result
	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Server is good and running!"})
}

func isLoggedIn(gc *gin.Context) (bool, User) {
	session := sessions.Default(gc)
	useridStr := session.Get("userid")
	if useridStr == nil {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged in!"})
		return false, User{}
	}

	log.Println("User Session is found! userid:" + useridStr.(string))

	user := User{}
	if err := user.GetById(bson.ObjectIdHex(useridStr.(string))); err != nil {
		log.Println("Session exists but User data is not in DB! Session cleared! " + err.Error())
		session.Clear()
		gc.JSON(http.StatusOK, gin.H{"status": -2, "message": "Server Error!"})
		return false, User{}
	}

	return true, user
}

func sync(gc *gin.Context) {
	loggedIn, user := isLoggedIn(gc)
	if loggedIn == false {
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
		//userData.Pets = posted.Pets
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
			//userData.Pets = posted.Pets
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
