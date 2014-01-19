package models

import (
	"code.google.com/p/go.crypto/bcrypt"
	"fmt"
	"github.com/rif/forms"
	"labix.org/v2/mgo/bson"
	"time"
)

type User struct {
	Id        bson.ObjectId `bson:"_id,omitempty"`
	Email     string
	Password  []byte
	FirstName string
	LastName  string
	Country   string
	Location  string
	BirthDate time.Time
	Gender    string
	Avatar    string `bson:"avatar,omitempty"`
	FbId      string
	GlId      string
	Admin     bool `bson:"admin,omitempty"`
}

type PasswordToken struct {
	Uuid      string `bson:"_id,omitempty"`
	User      bson.ObjectId
	CreatedOn time.Time
}

type FacebookLocation struct {
	Id   string
	Name string
}

func EncryptPassword(password string) (hpass []byte, err error) {
	hpass, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return

}

//SetPassword takes a plaintext password and hashes it with bcrypt and sets the
//password field to the hash.
func (u *User) SetPassword(password string) {
	hpass, err := EncryptPassword(password)
	if err != nil {
		Log("bcrypt: ", err.Error())
	}
	u.Password = hpass
}

// Returns full name
func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

func (u *User) Info() string {
	return fmt.Sprintf("%s %d, %s - %s", u.Sex(), u.Age(), u.Location, u.Country)
}

//Login validates and returns a user object if they exist in the database.
func Login(ctx *Context, email, password string) (u *User, err error) {
	err = ctx.C("users").Find(bson.M{"email": email}).One(&u)
	if err != nil {
		return
	}
	err = bcrypt.CompareHashAndPassword(u.Password, []byte(password))
	if err != nil {
		u = nil
	}
	return
}

func (u *User) Sex() string {
	if u.Gender == "m" {
		return "Male"
	}
	return "Female"
}

func (u *User) Age() (age int) {
	born := u.BirthDate
	today := time.Now()
	birthday := born
	if born.Month() != time.February && born.Day() != 29 {
		birthday = time.Date(today.Year(), born.Month(), born.Day(), 0, 0, 0, 0, time.UTC)
	} else {
		birthday = time.Date(today.Year(), born.Month(), born.Day()-1, 0, 0, 0, 0, time.UTC)
	}
	age = today.Year() - born.Year()
	if birthday.After(today) {
		age -= 1
	}
	return
}

var (
	UserForm = forms.Form{
		Fields: []forms.Field{
			forms.Field{Name: "password1", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "password2", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "email", Validators: []forms.Validator{forms.NonemptyValidator, forms.EmailValidator}},
			forms.Field{Name: "firstname", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "lastname", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "country", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "location", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "birthdate", Validators: []forms.Validator{forms.NonemptyValidator, forms.DateValidator}, Converter: forms.TimeConverter},
			forms.Field{Name: "gender", Validators: []forms.Validator{forms.NonemptyValidator}},
		},
	}
	ContactForm = forms.Form{
		Fields: []forms.Field{
			forms.Field{Name: "name", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "email", Validators: []forms.Validator{forms.NonemptyValidator, forms.EmailValidator}},
			forms.Field{Name: "message", Validators: []forms.Validator{forms.NonemptyValidator}},
		},
	}
)
