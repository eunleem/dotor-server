package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var dbcImages *mgo.Collection

type Image struct {
	Id        bson.ObjectId `bson:"_id" json:"id"`
	UserId    bson.ObjectId `json:"-"`
	Category  string        `bson:",omitempty" json:"category" form:"category"`
	RelatedId bson.ObjectId `bson:",omitempty" json:"relatedid" form:"relatedid"`
	Filename  string        `json:"filename"`
	Taken     time.Time     `json:"-"`
	Created   time.Time     `json:"created"`
}

func ensureIndexesImages() (err error) {
	index := mgo.Index{
		Key:        []string{"userid"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	if err = dbcImages.EnsureIndex(index); err != nil {
		return errors.New("Could not ensure index for Images")
	}

	return
}

//////////////////////// IMAGES /////////////////////
func (i *Image) Insert() (newId bson.ObjectId, err error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}

	if i.Created.IsZero() {
		i.Created = time.Now()
	}

	if err = dbcImages.Insert(&i); err != nil {
		log.Println("Could not insert an image")
		return i.Id, err
	}

	return i.Id, err
}

func (i *Image) GetById(id bson.ObjectId) (err error) {
	if err = dbcImages.FindId(id).One(&i); err != nil {
		log.Println("Could not find ImageById. err:" + err.Error())
		return
	}

	return
}

func (i *Image) Delete() (err error) {
	if i.Id.Valid() == false {
		return errors.New("Invalid Id for Image. Could not delete.")
	}

	if err = dbcImages.RemoveId(i.Id); err != nil {
		log.Println("Failed to delete an image. err:" + err.Error())
		return
	}

	return
}

///////////////////////// CONTROLLERS /////////////////////////

func getImage(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	id := gc.Query("id")

	var image Image
	if err := dbcImages.FindId(bson.ObjectIdHex(id)).One(&image); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Image not found."})
		return
	}

	filepath := "./web/img/" + image.Filename
	gc.File(filepath)
	return
}

func insertImage(gc *gin.Context) {
	log.Println("InsertImage Called!")
	loggedIn, myaccount := isLoggedIn(gc)
	if loggedIn == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Not Logged In."})
		return
	}

	tempFile, handler, err := gc.Request.FormFile("image")
	if err != nil {
		log.Print("FormFile:" + err.Error())
		return
	}
	defer tempFile.Close()

	ext := filepath.Ext(handler.Filename)
	if strings.EqualFold(ext, ".jpg") == false &&
		strings.EqualFold(ext, ".jpeg") == false &&
		strings.EqualFold(ext, ".png") == false {
		log.Println("Unsupported file is uploaded. ext: " + ext)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Unsupported File type!"})
		return
	}

	category := gc.Request.FormValue("category")
	relatedId := gc.Request.FormValue("relatedid")

	log.Println("category: " + category)
	log.Println("relatedId: " + relatedId)

	var posted Image
	posted.Id = bson.NewObjectId()
	posted.UserId = myaccount.Id

	posted.Category = category
	posted.RelatedId = bson.ObjectIdHex(relatedId)

	fileName := posted.Id.Hex() + ext
	destFile, err := os.OpenFile(("./web/img/" + fileName), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println(err)
		return
	}

	defer destFile.Close()
	io.Copy(destFile, tempFile)

	posted.Filename = fileName
	if _, err := posted.Insert(); err != nil {
		log.Println("Error inserting image to db. err: " + err.Error())
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error insert image to db!"})
		return
	}
	gc.JSON(http.StatusOK, gin.H{
		"status":   0,
		"message":  "Successfully uploded a file!",
		"newid":    posted.Id.Hex(),
		"filename": fileName,
	})

	if category == "review" {
		var review Review
		if err := dbcReviews.FindId(posted.RelatedId).One(&review); err != nil {
			log.Println(err)
			gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while updating review!"})
			return
		}

		if review.Images == nil {
			review.Images = make([]bson.ObjectId, 0)
		}
		review.Images = append(review.Images, posted.Id)
		review.IsDraft = false

		if _, err := review.Update(); err != nil {
			log.Println(err)
			gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while updating review! B"})
			return
		}
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Successfully uploded a file!", "newid": posted.Id.Hex(), "filename": fileName})
	return
}
