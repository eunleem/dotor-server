package main

import (
	"log"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

func isLoggedIn(gc *gin.Context) (bool, User) {
	session := sessions.Default(gc)
	useridStr := session.Get("userid")
	if useridStr == nil {
		RequireLogin(gc)
		return false, User{}
	}

	log.Println("User Session is found! userid:" + useridStr.(string))

	user := User{}
	if err := user.GetById(bson.ObjectIdHex(useridStr.(string))); err != nil {
		log.Print("Session exists but User data is not in DB! Session cleared! " + err.Error())
		session.Clear()
		session.Save()
		RequireLogin(gc)
		return false, User{}
	}

	return true, user
}

func isLoggedInNoAutoResp(gc *gin.Context) (bool, User) {
	session := sessions.Default(gc)
	useridStr := session.Get("userid")
	if useridStr == nil {
		log.Print("User is Not LoggedIn.")
		return false, User{}
	}

	log.Println("User Session is found! userid:" + useridStr.(string))

	user := User{}
	if err := user.GetById(bson.ObjectIdHex(useridStr.(string))); err != nil {
		log.Print("Session exists but User data is not in DB! Session cleared! " + err.Error())
		session.Clear()
		session.Save()
		return false, User{}
	}

	log.Print("User is LoggedIn.")

	return true, user
}

func isAdmin(myAccount User) bool {
	return myAccount.UserGroup == UserGroupAdmin
}
