package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"strconv"
	"time"
)

var dbcPets *mgo.Collection

// Pet owned by User
type Pet struct {
	Id           bson.ObjectId `bson:"_id" json:"petid"`
	OwnerUserId  bson.ObjectId `bson:"owneruserid" json:"-"`
	Name         string        `json:"name" binding:"required"`
	Type         string        `json:"type" binding:"required"`
	Gender       string        `json:"gender" binding:"required"`
	Age          int           `json:"age" binding:"required"`
	Size         string        `json:"size" binding:"required"`
	ThumbnailURL string        `json:"thumbnailurl"`
	PictureURL   string        `json:"pictureurl"`
	Created      time.Time     `json:"created"`
}

func ensureIndexesPets() (err error) {
	index := mgo.Index{
		Key:        []string{"owneruserid"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}

	if err = dbcPets.EnsureIndex(index); err != nil {
		return errors.New("Could not ensure index for Pets.")
	}

	return
}

//////////////////////////      PET      /////////////////////////

func (i *Pet) Insert() (bson.ObjectId, error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}

	if i.Created.IsZero() {
		i.Created = time.Now()
	}

	err := dbcPets.Insert(&i)
	if err != nil {
		log.Println("Could not insert a pet.")
		return i.Id, err
	}

	log.Println("Inserted a pet. OwnerUserId: " + i.OwnerUserId.Hex())
	log.Println("PetId: " + i.Id.Hex())

	return i.Id, nil
}

func (i *Pet) Update() (changeInfo *mgo.ChangeInfo, err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Pet Id")
		return
	}

	changeInfo, err = dbcPets.UpsertId(i.Id, &i)
	return
}

func (i *Pet) Delete() (err error) {
	if i.Id.Valid() == false {
		err = errors.New("Invalid Pet Id")
		return
	}

	err = dbcPets.RemoveId(i.Id)
	return
}

func (i *Pet) GetById(id bson.ObjectId) error {
	err := dbcPets.FindId(id).One(&i)
	if err != nil {
		log.Println("Could not find PetById.")
		return err
	}

	return nil
}

/////////////////////////    CONTROLLERS    ////////////////////////

func getPet(gc *gin.Context) {

}

func insertPet(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	pet := Pet{
		OwnerUserId: user.Id,
	}

	if err := gc.BindJSON(&pet); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	// #TODO Limit the number of Pets per user

	if newId, err := pet.Insert(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{
			"status":  -1,
			"message": "Failed! Insert pet info.",
		})

	} else {
		gc.JSON(http.StatusOK, gin.H{
			"status":  0,
			"message": "Successful! Insert pet info.",
			"newid":   newId.Hex(),
		})
	}
}

func updatePet(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	DumpRequestBody(gc)

	postedPet := Pet{}

	if err := gc.BindJSON(&postedPet); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	log.Println("petid: " + postedPet.Id.Hex())

	pet := Pet{}

	if err := pet.GetById(postedPet.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid PetId. Id: " + postedPet.Id.Hex()})
		return
	}

	if pet.OwnerUserId != user.Id {
		log.Println("User does not own this pet! petId: " + pet.Id.Hex())
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Owner."})
		return
	}

	pet.Name = postedPet.Name
	pet.Age = postedPet.Age
	pet.Size = postedPet.Size
	pet.Gender = postedPet.Gender

	if changeInfo, err := pet.Update(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed update pet."})
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Updated " + strconv.Itoa(changeInfo.Updated) + " field(s)."})
	}
	return

}

func deletePet(gc *gin.Context) {
	isLoggedIn, user := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	postedPet := Pet{}

	if err := gc.BindJSON(&postedPet); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Required Form value is missing."})
		return
	}

	pet := Pet{}

	if err := pet.GetById(postedPet.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid PetId."})
		return
	}

	if pet.OwnerUserId != user.Id {
		log.Println("User does not own this pet!")
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Owner."})
		return
	}

	if err := pet.Delete(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed update pet."})
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Removed pet."})
	}
	return

}
