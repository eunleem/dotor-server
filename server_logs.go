package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

const TableNameLogs = "server_logs"

var dbcLogs *mgo.Collection

func init() {
	const tableName = TableNameLogs
	dbcLogs = dbSession.DB(dbName).C(tableName)
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

	// TODO Ensure 2dsphere index
}

const LogTypeGeneral = 0
const LogTypeDebug = 600
const LogTypeError = 900
const LogTypeSecurity = 1000

const LogPriorityLow = 0
const LogPriorityMedium = 10
const LogPriorityHigh = 20
const LogPriorityVeryHigh = 30
const LogPriorityCritical = 999

// Review for review
type ServerLog struct {
	Id       bson.ObjectId `bson:"_id" json:"id"`
	Type     int           `json:"type"`
	Priority int           `json:"priority"`
	Message  string        `json:"message"`
	UserId   bson.ObjectId `bson:",omitempty" json:"userid"`
	Created  time.Time     `json:"created"`
}
