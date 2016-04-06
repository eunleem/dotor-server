package main

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const TableNameImages = "images"

var dbcImages *mgo.Collection

func init() {
	const tableName = TableNameImages
	dbcImages = dbSession.DB(dbName).C(tableName)

	dbCols[tableName] = dbSession.DB(dbName).C(tableName)
	index := mgo.Index{
		Key:        []string{"userid"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}
	if err := dbCols[tableName].EnsureIndex(index); err != nil {
		log.Printf("Error while ensuring index for '%s'. %s", tableName, err.Error())
		panic(err)
	}
}

type Image struct {
	Id            bson.ObjectId `bson:"_id" json:"id"`
	UserId        bson.ObjectId `json:"-"`
	Category      string        `bson:",omitempty" json:"category" form:"category"`
	RelatedId     bson.ObjectId `bson:",omitempty" json:"relatedid" form:"relatedid"`
	Filename      string        `json:"filename"`
	ThumbnailPath string        `json:"thumb_path"`
	Taken         time.Time     `json:"-"`
	Created       time.Time     `json:"created"`
}

////////////////////////     BASIC OPS     /////////////////////
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

func (i *Image) Delete() (err error) {
	if i.Id.Valid() == false {
		return errors.New("Invalid Id for Image. Could not delete.")
	}

	if err = dbcImages.RemoveId(i.Id); err != nil {
		log.Println("Failed to delete an image. err:" + err.Error())
		return
	}

	if err = os.Remove("./web/img/" + i.Filename); err != nil {
		log.Println("Failed to delete an image file. err:" + err.Error())
	}
	return
}

func (i *Image) GetById(id bson.ObjectId) (err error) {
	if err = dbcImages.FindId(id).One(&i); err != nil {
		log.Println("Could not find ImageById. err:" + err.Error())
		return
	}

	return
}

///////////////////////// CONTROLLERS /////////////////////////

func getImage(gc *gin.Context) {
	loggedIn, _ := isLoggedIn(gc)
	if loggedIn == false {
		return
	}

	idStr := gc.Param("id")
	if bson.IsObjectIdHex(idStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Id."})
		return
	}
	id := bson.ObjectIdHex(idStr)

	var image Image
	if err := dbcImages.FindId(id).One(&image); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Image not found."})
		return
	}

	sizeStr := gc.DefaultQuery("size", "original")
	fileDir := "./web/img/"
	var filepath string
	if sizeStr == "original" {
		filepath = fileDir + image.Filename
	} else if sizeStr == "thumbnail" {
		filepath = fileDir + "/thumb/" + image.Filename
	}

	gc.File(filepath)
	return
}

func insertImage(gc *gin.Context) {
	loggedIn, myAccount := isLoggedIn(gc)
	if loggedIn == false {
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
	relatedIdStr := gc.Request.FormValue("relatedid")
	if bson.IsObjectIdHex(relatedIdStr) == false {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid relatedId!"})
		return
	}

	log.Println("category: " + category)
	log.Println("relatedId: " + relatedIdStr)

	var posted Image
	posted.Id = bson.NewObjectId()
	posted.UserId = myAccount.Id
	posted.Category = category
	posted.RelatedId = bson.ObjectIdHex(relatedIdStr)

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

	} else if category == "pet" {
		var pet Pet
		if err := dbcPets.FindId(posted.RelatedId).One(&pet); err != nil {
			log.Println(err)
			gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while updating pet!"})
			return
		}

		pet.ProfileImageId = posted.Id

		if _, err := pet.Update(); err != nil {
			log.Println(err)
			gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while updating pet! B"})
			return
		}

		if pet.ProfileImageId.Valid() == true {
			image := Image{
				Id: pet.ProfileImageId,
			}
			if err := image.Delete(); err != nil {
				log.Print("Error deleting pet image")
			}
		}
	}

	gc.JSON(http.StatusOK, gin.H{
		"status":   0,
		"message":  "Successfully uploded a file!",
		"newid":    posted.Id.Hex(),
		"filename": fileName,
	})
	return
}
