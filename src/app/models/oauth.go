package models

import (
	"encoding/json"
	"fmt"
	"github.com/ungerik/go-gravatar"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strings"
	"time"
)

type FacebookProfile struct {
	Id          string           
	First_name  string           
	Middle_name string           
	Last_name   string           
	Username    string           
	Birthday    string           
	Gender      string           
	Email       string           
	Timezone    int              
	Locale      string           
	Location    FacebookLocation 
}

func (fbp *FacebookProfile) GetBirthdate() (time.Time, error) {
	return time.Parse("01/02/2006", fbp.Birthday)
}

func (fbp *FacebookProfile) GetGender() string {
	if fbp.Gender == "male" {
		return "m"
	}
	return "f"
}

func (fbp *FacebookProfile) GetLocation() (string, string) {
	loc := strings.Split(fbp.Location.Name, ",")
	if len(loc) != 2 {
		return "", ""
	}
	return strings.TrimSpace(loc[0]), strings.TrimSpace(loc[1])
}

//Login validates and returns a user object if they exist in the database.
func LoginWithFacebook(ctx *Context, fbp *FacebookProfile) (u *User, redirect string, err error) {
	// check if facebook profile has no email set
	if fbp.Email == "" {
		fbp.Email = fbp.Username + "@facebook.com"
	}
	err = ctx.C("users").Find(bson.M{"email": fbp.Email}).One(&u)
	if err != nil {
		bday, err := fbp.GetBirthdate()
		if err != nil {
			bday = time.Time{}
		}
		city, country := fbp.GetLocation()
		u = &User{
			Id:        bson.NewObjectId(),
			Email:     fbp.Email,
			FirstName: fbp.First_name + " " + fbp.Middle_name,
			LastName:  fbp.Last_name,
			Country:   country,
			Location:  city,
			BirthDate: bday,
			Gender:    fbp.GetGender(),
			FbId:      fbp.Id,
		}
		// set avatar
		if resp, err := http.Get(fmt.Sprintf("https://graph.facebook.com/%s?fields=picture", u.FbId)); err == nil {
			defer resp.Body.Close()
			if profile, err := ioutil.ReadAll(resp.Body); err == nil {
				a := struct {
					Picture struct{ Data struct{ Url string } }
				}{}
				if err = json.Unmarshal(profile, &a); err == nil {
					Cache.Set(u.FbId, a.Picture.Data.Url, 0)
					u.Avatar = a.Picture.Data.Url
				}
			}
		}
		if u.Avatar == "" {
			u.Avatar = gravatar.UrlSize(u.Email, 80)
		}
		err = ctx.C("users").Insert(u)
		if err != nil {
			return nil, "login", err
		}
		redirect = "profile"
	}
	redirect = "index"
	return
}

type GoogleProfile struct {
	Id          string 
	Name        string 
	Given_name  string 
	Family_name string 
	Link        string 
	Birthday    string 
	Gender      string 
	Email       string 
	Locale      string 
	Picture     string 
}

func (gp *GoogleProfile) GetBirthdate() (time.Time, error) {
	return time.Parse("2006-01-02", gp.Birthday)
}

func (gp *GoogleProfile) GetGender() string {
	if gp.Gender == "male" {
		return "m"
	}
	return "f"
}

//Login validates and returns a user object if they exist in the database.
func LoginWithGoogle(ctx *Context, gp *GoogleProfile) (u *User, redirect string, err error) {
	err = ctx.C("users").Find(bson.M{"email": gp.Email}).One(&u)
	if err != nil {
		bday, err := gp.GetBirthdate()
		if err != nil {
			bday = time.Time{}
		}
		u = &User{
			Id:        bson.NewObjectId(),
			Email:     gp.Email,
			FirstName: gp.Given_name,
			LastName:  gp.Family_name,
			BirthDate: bday,
			Gender:    gp.GetGender(),
			GlId:      gp.Id,
		}

		// set avatar
		u.Avatar = gp.Picture
		if u.Avatar == "" {
			u.Avatar = gravatar.UrlSize(u.Email, 80)
		}
		err = ctx.C("users").Insert(u)
		if err != nil {
			return nil, "login", err
		}
		redirect = "profile"
	}
	redirect = "index"
	return
}
