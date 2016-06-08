package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
)

func AlreadyLoggedIn(gc *gin.Context) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  100,
		"message": "Already Logged In.",
	})
}

func AlreadyLoggedInSlideExpiry(gc *gin.Context) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  100,
		"message": "Already Logged In.",
	})

	session := sessions.Default(gc)
	session.Options(sessions.Options{
		MaxAge: SessionTimeSecs,
	})

	session.Set("userid", session.Get("userid").(string))

	if err := session.Save(); err != nil {
		log.Print(err)
	}

	log.Print("Extended Expiry.")
}

func RequireLogin(gc *gin.Context) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  -100,
		"message": "Not Authorized.",
	})
}

func NotAuthorized(gc *gin.Context) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  -110,
		"message": "Not Authorized.",
	})
}

func ErrorBinding(gc *gin.Context) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  -200,
		"message": "Error binding data.",
	})
}

func MissingRequiredValue(gc *gin.Context, values ...string) {
	message := ""
	for _, val := range values {
		message += " " + val
	}
	gc.JSON(http.StatusOK, gin.H{
		"status":  -1,
		"message": "Missing Required Value!" + message,
	})
}

func DataNotFound(gc *gin.Context) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  -404,
		"message": "Data Not Found.",
	})
}

func DatabaseError(gc *gin.Context) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  -450,
		"message": "Database Error.",
	})
}

func ServerError(gc *gin.Context) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  -500,
		"message": "Server Error.",
	})
}

func Successful(gc *gin.Context) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Successful.",
	})
}

func InsertSuccessful(gc *gin.Context, data gin.H) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Insert Successful.",
		"data":    data,
	})
}

func UpdateSuccessful(gc *gin.Context) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Update Successful.",
	})
}

func DeleteSuccessful(gc *gin.Context) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Delete Successful.",
	})
}

func GetSuccessful(gc *gin.Context, data *gin.H) {
	gc.JSON(http.StatusOK, gin.H{
		"status":  0,
		"message": "Get Successful.",
		"data":    data,
	})
}
