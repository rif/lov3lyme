package models

import (
	"labix.org/v2/mgo/bson"
	"reflect"
	"testing"
)

func TestIntervalAge(t *testing.T) {
	f := &Filter{}
	f.ParseAge("30-40")
	if f.MinAge != 30 || f.MaxAge != 40 {
		t.Error("Error parsing interval: ", f)
	}
}

func TestAddQuery(t *testing.T) {
	f := &Filter{}
	f.ParseAge("30-40")
	query := bson.M{}
	expected := bson.M{"$gte": f.MinAge, "$lte": f.MaxAge}
	f.AddQuery(query)
	if !reflect.DeepEqual(query["age"], expected) {
		t.Error("Error adding query: ", query)
	}
}
