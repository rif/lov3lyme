package models

import (
	"github.com/rif/forms"
	"labix.org/v2/mgo/bson"
	"time"
)

type Contest struct {
	Id                bson.ObjectId `bson:"_id,omitempty"`
	Name              string
	Description       string
	Country           string
	Location          string
	Gender            string
	MinAge            int
	MaxAge            int
	AdmissionDeadline time.Time
	VotingDeadline    time.Time
	Public            bool
	RequireApproval   bool
	Registered        []*RegItem
	Comments          []*Comment `bson:"comments,omitempty"`
	User              bson.ObjectId
}

type RegItem struct {
	User        bson.ObjectId
	UserName    string
	UserInfo    string
	Photo       bson.ObjectId
	Title       string
	Description string
	Approved    bool
}

func (ri *RegItem) Id() bson.ObjectId {
	return ri.Photo
}

func (c *Contest) Sex() string {
	if c.Gender == "m" {
		return "Male"
	}
	return "Female"
}

func (c *Contest) CanRegister() bool {
	now := time.Now()
	return now.Before(c.AdmissionDeadline)
}

func (c *Contest) CanVote() bool {
	now := time.Now()
	return now.After(c.AdmissionDeadline) && now.Before(c.VotingDeadline)
}

func (c *Contest) AD() string {
	return c.AdmissionDeadline.Format("02 Jan 2006")
}

func (c *Contest) VD() string {
	return c.VotingDeadline.Format("02 Jan 2006")
}

func (c *Contest) ToBeApproved() (res []*RegItem) {
	for _, ri := range c.Registered {
		if !ri.Approved {
			res = append(res, ri)
		}
	}
	return
}

func (c *Contest) CommentList() []*Comment {
	return c.Comments
}

var (
	ContestForm = &forms.Form{
		Fields: []forms.Field{
			forms.Field{Name: "name", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "description"},
			forms.Field{Name: "country"},
			forms.Field{Name: "location"},
			forms.Field{Name: "gender"},
			forms.Field{Name: "min_age", Converter: forms.IntConverter, Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "max_age", Converter: forms.IntConverter, Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "admission_deadline", Converter: forms.TimeConverter},
			forms.Field{Name: "voting_deadline", Converter: forms.TimeConverter},
			forms.Field{Name: "require_approval", Converter: forms.BoolConverter},
		},
	}
)
