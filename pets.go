package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const TableNamePets = "pets"

var dbcPets *mgo.Collection

func init() {
	const tableName = TableNamePets
	dbcPets = dbSession.DB(dbName).C(tableName)
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

const PetMale = 0
const PetFemale = 1

const PetOther = -1
const PetDog = 0
const PetCat = 1

const PetSizeSmall = 0
const PetSizeMedium = 1
const PetSizeLarge = 2
const PetSizeXLarge = 3
const PetSizeXXLarge = 4

// Pet owned by User
type Pet struct {
	Id             bson.ObjectId   `bson:"_id" json:"petid"`
	UserId         bson.ObjectId   `bson:"userid" json:"userid"`
	Name           string          `json:"name"`
	Type           int             `json:"type"`
	Gender         int             `json:"gender"`
	Size           int             `bson:",omitempty" json:"size,omitempty"`
	Age            int             `bson:",omitempty" json:"age,omitempty"`
	Birthday       time.Time       `bson:",omitempty" json:"birthday"`
	Breed          string          `bson:",omitempty" json:"breed"`
	BreedKor       string          `bson:",omitempty" json:"breed_kor"`
	ProfileImageId bson.ObjectId   `bson:",omitempty" json:"profile_imageid"`
	ImageIds       []bson.ObjectId `bson:",omitempty" json:"-"`
	Updated        time.Time       `json:"updated"`
	Created        time.Time       `json:"created"`
}

/////////////////////      BASIC OPERATIONS      //////////////////////

func (i *Pet) Insert() (bson.ObjectId, error) {
	if i.Id.Valid() == false {
		i.Id = bson.NewObjectId()
	}

	if i.UserId.Valid() == false {
		return i.Id, errors.New("UserId is required.")
	}

	if i.Created.IsZero() {
		i.Created = time.Now()
	}

	err := dbcPets.Insert(&i)
	if err != nil {
		log.Println("Could not insert a pet.")
		return i.Id, err
	}

	log.Printf("Inserted a pet. PetId: %s, Owner UserId: %s.", i.Id.Hex(), i.UserId.Hex())

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
	if isLoggedIn, _ := isLoggedIn(gc); isLoggedIn == false {
		return
	}

	idStr := gc.Query("id")
	if bson.IsObjectIdHex(idStr) == false {
		log.Println("Invalid pet id.")
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Pet Id."})
		return
	}
	petId := bson.ObjectIdHex(idStr)
	pet := Pet{}
	if err := pet.GetById(petId); err != nil {
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Error while getting pet."})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Fetched Pet.", "pet": pet})
	return
}

func insertPet(gc *gin.Context) {
	isLoggedIn, myAccount := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	DumpRequestBody(gc)

	pet := Pet{}

	if err := gc.BindJSON(&pet); err != nil {
		log.Print(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed binding data. Maybe required value is missing."})
		return
	}

	pet.UserId = myAccount.Id

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
	isLoggedIn, myAccount := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	//DumpRequestBody(gc)

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

	if pet.UserId != myAccount.Id {
		log.Println("User does not own this pet! petId: " + pet.Id.Hex())
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid Owner."})
		return
	}

	pet.Name = postedPet.Name
	pet.Type = postedPet.Type
	pet.Gender = postedPet.Gender
	pet.Size = postedPet.Size
	pet.Breed = postedPet.Breed
	pet.Birthday = postedPet.Birthday

	if changeInfo, err := pet.Update(); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Failed update pet."})
	} else {
		gc.JSON(http.StatusOK, gin.H{"status": 0, "message": "Updated " + strconv.Itoa(changeInfo.Updated) + " field(s)."})
	}
	return
}

func deletePet(gc *gin.Context) {
	isLoggedIn, myAccount := isLoggedIn(gc)
	if isLoggedIn == false {
		return
	}

	petIdStr := gc.Query("id")

	pet := Pet{
		Id: bson.ObjectIdHex(petIdStr),
	}

	if err := pet.GetById(pet.Id); err != nil {
		log.Println(err)
		gc.JSON(http.StatusOK, gin.H{"status": -1, "message": "Invalid PetId."})
		return
	}

	if pet.UserId != myAccount.Id {
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
