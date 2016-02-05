package main

import (
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"net/http"
)

func testjson(gc *gin.Context) {
	gc.JSON(http.StatusOK, bson.M{"status": 0, "Message": "JSON Server is running!"})
}

func testsession(gc *gin.Context) {
	session := sessions.Default(gc)

	var count int
	v := session.Get("count")
	if v == nil {
		count = 0
	} else {
		count = v.(int)
		count += 1
	}

	session.Set("count", count)
	session.Save()
	gc.JSON(200, gin.H{"count": count})
}
