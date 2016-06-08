package main

import (
	"errors"
	"flag"
	"log"
	"net/http"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type HostSwitch map[string]http.Handler

// Implement the ServerHTTP method on our new type
func (hs HostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if a http.Handler is registered for the given host.
	// If yes, use it to handle the request.
	if handler := hs[r.Host]; handler != nil {
		handler.ServeHTTP(w, r)
	} else {
		// Handle host names for wich no handler is registered
		http.Error(w, "Forbidden", 403) // Or Redirect?
	}
}

const rootDir = "./web"

var devMode = isDevMode()
var requestLogginEnabled = isRequestLoggingEnabled()
var dbName = setDbName()

const SessionTimeSecs = 60 * 60 * 24 * 7 // an hour

func isDevMode() bool {
	devModePtr := flag.Bool("dev", false, "When dev mode is on, it uses port 8088")
	flag.Parse()
	return *devModePtr
}

func isRequestLoggingEnabled() bool {
	flagOnPtr := flag.Bool("reqlog", false, "When reqlog is set, it prints requests")
	flag.Parse()
	return *flagOnPtr
}

func setDbName() string {
	if devMode == true {
		return "dotor_dev"
	}

	return "dotor"
}

func main() {
	// DB and Collections are Automatically Opened. Checkout database.go
	defer CloseDb() // Needs this line only!

	if devMode == false {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	store, _ := sessions.NewRedisStore(10, "tcp", "localhost:6379", "xlaVkfVkfgo", []byte("secret"))
	store.Options(sessions.Options{
		MaxAge: SessionTimeSecs,
		Secure: true,
	})
	router.Use(sessions.Sessions("dotor_session", store))

	//newReditClient()

	router.Static("/img", (rootDir + "/img"))
	router.Static("/thumb", (rootDir + "/img/thumb"))

	router.GET("/server/status", checkServerStatus)

	router.POST("/user/register", registerTemp)
	router.POST("/user/login", login)

	router.GET("/user/get/:id", getUser)

	router.POST("/user/update/email/:email", updateEmail)
	router.POST("/user/update/nickname/:nickname", updateNickname)

	router.POST("/user/check/email/:email", checkEmail)          // Is email valid and unique?
	router.POST("/user/check/nickname/:nickname", checkNickname) // Is nickname valid and unique?

	router.GET("/user/verify/:email/:code", verifyEmail)

	router.GET("/pet/get/:id", getPet)

	router.POST("/pet/insert", insertPet)
	router.POST("/pet/update", updatePet)
	router.POST("/pet/delete/:id", deletePet)

	router.GET("/review/:id", getReview)
	router.POST("/review/insert", insertReview)
	router.POST("/review/update", updateReview)
	router.POST("/review/delete/:id", deleteReview)

	router.POST("/review/like/:id", likeReview)

	router.GET("/reviews/all", getReviews)
	router.GET("/reviews/my", getMyReviews)
	router.POST("/reviews/location", getReviewsByLocation)
	router.POST("/reviews/pet", getReviewsByPet)
	router.POST("/reviews/category/:categories", getReviewsByCategory)

	router.POST("/image/insert", insertImage)
	router.GET("/image/:id", getImage)

	router.GET("/hospital/get/:id", getHospital)
	router.POST("/hospital/insert", insertHospital)
	router.POST("/hospitals/nearby", getHospitalsNearby)

	router.GET("/notification/:id", getNotification)

	router.GET("/notifications", getMyNotifications)

	router.POST("/notification/readall", readAllNotification)
	router.POST("/notification/read/:id", readNotification)
	router.POST("/notification/received/:id", receivedNotification)

	router.POST("/comment/insert/:category/:relatedid", insertComment)
	router.POST("/comment/update/:id", updateComment)
	router.POST("/comment/delete/:id", deleteComment)

	router.GET("/comments/:category/:relatedid", getComments)
	//router.POST("/comment/like/:id", likeComment)

	router.POST("/settings/push/upsert", upsertPushSetting)

	router.POST("/report/:category/:id", insertReport)

	router.POST("/feedback/insert", insertFeedback)

	if devMode == true {
		hsDev := make(HostSwitch)
		hsDev["dotor.team88.net:8088"] = router
		err := http.ListenAndServeTLS(":8088", "/var/lib/acme/live/team88.net/fullchain", "/var/lib/acme/live/team88.net/privkey", hsDev)
		if err != nil {
			log.Fatal("ListenAndServeTLS: ", err)
		}
		//router.RunTLS(":8088", "/var/lib/acme/live/team88.net/fullchain", "/var/lib/acme/live/team88.net/privkey")
	} else {
		// PRODUCTION runs on port 8080 for Dotor.
		hs := make(HostSwitch)
		hs["team88.net:8080"] = router
		hs["dotor.team88.net:8080"] = router
		err := http.ListenAndServeTLS(":8080", "/var/lib/acme/live/team88.net/fullchain", "/var/lib/acme/live/team88.net/privkey", hs)
		if err != nil {
			log.Fatal("ListenAndServeTLS on 8080: ", err)
		}
		//http.ListenAndServeTLS(":443", "/var/lib/acme/live/team88.net/fullchain", "/var/lib/acme/live/team88.net/privkey", hs)
		//router.RunTLS(":443", "/var/lib/acme/live/team88.net/fullchain", "/var/lib/acme/live/team88.net/privkey")
	}
}

func reset(gc *gin.Context) {
	//DropDb()
	//resetRedis()
	//os.RemoveAll("./web/img")
	//os.Mkdir("./web/img", 0775)
	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "RESET DB!"})
}

func checkServerStatus(gc *gin.Context) {
	// #TODO Report finer result
	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Server is good and running!"})
}

func getIdFromParam(gc *gin.Context) (id bson.ObjectId, err error) {
	idStr := gc.Param("id")
	if bson.IsObjectIdHex(idStr) == false {
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Invalid ObjectId.",
		})
		return bson.NewObjectId(), errors.New("Invalid ObjectId")
	}
	id = bson.ObjectIdHex(idStr)
	return id, nil
}
