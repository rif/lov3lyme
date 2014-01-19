package models

import (
	"labix.org/v2/mgo/bson"
)

type Message struct {
	Id       bson.ObjectId `bson:"_id,omitempty"`
	From     bson.ObjectId
	To       bson.ObjectId
	UserName string
	Avatar   string
	Subject  string
	Body     string
	Read     bool
}
