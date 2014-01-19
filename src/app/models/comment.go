package models

import (
	"github.com/rif/forms"
	"labix.org/v2/mgo/bson"
)

type Comment struct {
	Id       bson.ObjectId `bson:"_id,omitempty"`
	User     bson.ObjectId
	UserName string
	Avatar   string
	Body     string
}

type Commenter interface {
	CommentList() []*Comment
}

var (
	CommentForm = &forms.Form{
		Fields: []forms.Field{
			forms.Field{Name: "body", Validators: []forms.Validator{forms.NonemptyValidator}},
		},
	}
)
