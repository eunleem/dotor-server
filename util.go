package main

import (
	"crypto/sha512"
	"errors"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"math/rand"
	"net/http"
	"time"
)

func HashString(str string) []byte {
	h := sha512.New()
	h.Write([]byte(str))
	return h.Sum(nil)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-!@#$%^&*=|?<>"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// #REF: Generate Random String in Go
// http://stackoverflow.com/a/31832326
var src = rand.NewSource(time.Now().UnixNano())

func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

//////////     NET      ////////////
func RequirePostValues(gc *gin.Context, fields ...string) error {
	for _, elem := range fields {
		val := gc.PostForm(elem)
		if val == "" {
			gc.JSON(http.StatusOK, bson.M{"status": -1, "message": "Form value '" + elem + "' is required."})
			return errors.New("Required field " + elem + " is missing")
		}
	}

	return nil
}
