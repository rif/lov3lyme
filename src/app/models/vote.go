package models

import (
	"errors"
	"labix.org/v2/mgo/bson"
	"strconv"
	"strings"
	"time"
)

type Vote struct {
	Id          bson.ObjectId `bson:"_id,omitempty"`
	Photo       bson.ObjectId
	PhotoUser   bson.ObjectId
	Title       string
	Description string
	Country     string
	Location    string
	Age         int
	Gender      string
	Active      bool
	Contest     bson.ObjectId `bson:"contest,omitempty"`
	Score       float64
	UpdatedOn   time.Time
	User        bson.ObjectId
}

type Filter struct {
	Country  string
	Location string
	MinAge   int
	MaxAge   int
	Gender   string
}

func (f *Filter) ParseAge(age string) (err error) {
	if strings.TrimSpace(age) == "" {
		f.MinAge, f.MaxAge = 0, 0
		return
	}
	dashCount := strings.Count(age, "-")
	if dashCount > 1 {
		err = errors.New("Invalid age rage")
		return
	}
	var min, max int
	if dashCount == 1 {
		limits := strings.Split(age, "-")
		if len(limits) != 2 {
			err = errors.New("Invalid age rage")
			return
		}
		min, err = strconv.Atoi(strings.TrimSpace(limits[0]))
		if err != nil || min < 0 || min > 130 {
			err = errors.New("Invalid age rage")
			return
		}
		max, err = strconv.Atoi(strings.TrimSpace(limits[1]))
		if err != nil || max < 0 || max > 130 {
			err = errors.New("Invalid age rage")
			return
		}
	} else {
		min, err = strconv.Atoi(strings.TrimSpace(age))
		if err != nil {
			return
		}
		max = min
	}
	f.MinAge, f.MaxAge = min, max
	return
}

func (f *Filter) Age() (result string) {
	if f.MinAge != 0 {
		result += strconv.Itoa(f.MinAge)
	}
	if f.MaxAge != 0 && f.MaxAge != f.MinAge {
		result += " - " + strconv.Itoa(f.MaxAge)
	}
	return
}

func (f *Filter) String() (result string) {
	if f.Country != "" {
		result += "country: " + f.Country
	}
	if f.Location != "" {
		result += "location: " + f.Location
	}
	if f.Age() != "" {
		result += "age: " + f.Age()
	}
	if f.Gender != "" {
		result += "gender: " + f.Gender
	}
	return
}

func (f *Filter) AddQuery(m bson.M) {
	if f.Country != "" {
		m["country"] = bson.M{"$in": []string{f.Country, ""}}
	}
	if f.Location != "" {
		m["location"] = bson.M{"$in": []string{f.Location, ""}}
	}
	if f.MinAge > 0 {
		m["age"] = bson.M{"$gte": f.MinAge}
	}
	if f.MaxAge > 0 {
		ageQuery, ok := m["age"].(bson.M)
		if !ok {
			ageQuery = bson.M{}
		}
		ageQuery["$lte"] = f.MaxAge
	}
	if f.Gender != "" {
		m["gender"] = f.Gender
	}
}

func (f *Filter) AddContestQuery(m bson.M) {
	if f.Country != "" {
		m["country"] = bson.M{"$in": []string{f.Country, ""}}
	}
	if f.Location != "" {
		m["location"] = bson.M{"$in": []string{f.Location, ""}}
	}
	if f.MinAge > 0 {
		m["minage"] = bson.M{"$lte": f.MinAge}
	}
	if f.MaxAge > 0 {
		m["maxage"] = bson.M{"$gte": f.MaxAge}
	}
	if f.Gender != "" {
		m["gender"] = f.Gender
	}
}
