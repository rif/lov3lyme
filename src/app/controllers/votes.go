package controllers

import (
	"app/models"
	"fmt"
	"labix.org/v2/mgo/bson"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func Vote(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		return perform_status(w, req, http.StatusForbidden)
	}
	if req.URL.Query().Get(":csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	photoId := req.URL.Query().Get(":photo")
	if !bson.IsObjectIdHex(photoId) {
		return perform_status(w, req, http.StatusForbidden)
	}
	photo := models.Photo{}
	if err := ctx.C(P).FindId(bson.ObjectIdHex(photoId)).One(&photo); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}
	// check photo is own photo and return forbidden
	if c, _ := ctx.C(P).Find(bson.M{"_id": bson.ObjectIdHex(photoId), "user": ctx.User.Id}).Limit(1).Count(); c != 0 {
		return perform_status(w, req, http.StatusForbidden)
	}

	contestId := req.URL.Query().Get(":contest")
	var contest bson.ObjectId
	if contestId != "" && bson.IsObjectIdHex(contestId) {
		contest = bson.ObjectIdHex(contestId)
		// check contest is in voting period
		now := time.Now()
		query := ctx.C(C).Find(bson.M{
			"_id":               contest,
			"admissiondeadline": bson.M{"$lt": now},
			"votingdeadline":    bson.M{"$gt": now},
		}).Limit(1)
		if count, err := query.Count(); count != 1 || err != nil {
			return perform_status(w, req, http.StatusForbidden)
		}
	}
	hearts, err := strconv.ParseFloat(req.FormValue("v"), 64)
	if err != nil || hearts < 1 || hearts > 5 {
		return perform_status(w, req, http.StatusForbidden)
	}
	v := &models.Vote{
		Photo:       photo.Id,
		PhotoUser:   photo.User,
		Title:       photo.Title,
		Description: photo.Description,
		Country:     photo.Country,
		Location:    photo.Location,
		Age:         photo.Age,
		Gender:      photo.Gender,
		Active:      photo.Active,
		Contest:     contest,
		Score:       hearts,
		UpdatedOn:   time.Now(),
		User:        ctx.User.Id,
	}
	query := bson.M{"photo": v.Photo, "user": v.User}
	if contestId != "" {
		query["contest"] = contest
	} else {
		query["contest"] = bson.M{"$exists": false}
	}
	if _, err := ctx.C(V).Upsert(query, v); err != nil {
		models.Log("vote err: ", err.Error())
	}
	return nil
}

func GetVote(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		return perform_status(w, req, http.StatusForbidden)
	}
	photoId := req.URL.Query().Get(":photo")
	if !bson.IsObjectIdHex(photoId) {
		return perform_status(w, req, http.StatusForbidden)
	}
	contestId := req.URL.Query().Get(":contest")
	if contestId != "" && !bson.IsObjectIdHex(contestId) {
		return perform_status(w, req, http.StatusForbidden)
	}
	query := bson.M{"photo": bson.ObjectIdHex(photoId), "user": ctx.User.Id}
	if contestId != "" {
		query["contest"] = bson.ObjectIdHex(contestId)
	}
	vote := &models.Vote{}

	ctx.C(V).Find(query).One(vote) // on error we still want the stars

	hearts := ""
	for i := 0; i < 5; i++ {
		if float64(i) < vote.Score {
			hearts += `<s class="voted">`
		} else {
			hearts += "<s>"
		}
	}
	return AJAX("vote.html").Execute(w, map[string]interface{}{
		"v":         vote,
		"photoId":   photoId,
		"contestId": contestId,
		"hearts":    SafeHtml(hearts),
		"ctx":       ctx,
	})
}

func Filter(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if req.FormValue("csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	f := &models.Filter{
		Country:  strings.TrimSpace(req.FormValue("country")),
		Location: req.FormValue("location"),
		Gender:   strings.TrimSpace(req.FormValue("gender")),
	}
	err := f.ParseAge(req.FormValue("age"))
	if err != nil {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Invalid age range", ctx)))
		http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
		return nil
	}
	ctx.Session.Values["filter"] = f
	http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
	return nil
}

func Rankings(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	var results models.WilsonSorter
	pipe := ctx.C(V).Pipe([]bson.M{
		{"$match": bson.M{"active": true}},
		{"$group": bson.M{
			"_id":         "$photo",
			"count":       bson.M{"$sum": 1},
			"avg":         bson.M{"$avg": "$score"},
			"scores":      bson.M{"$push": "$score"},
			"title":       bson.M{"$addToSet": "$title"},
			"description": bson.M{"$addToSet": "$description"},
			"user":        bson.M{"$addToSet": "$photouser"},
		}},
		{"$unwind": "$user"},
		{"$unwind": "$title"},
		{"$unwind": "$description"},
	})
	pipe.All(&results)
	// calculate wilson rating http://www.goproblems.com/test/wilson/wilson-new.php
	vc := make([]int, 5)
	for _, r := range results {
		scores := r["scores"].([]interface{})
		for _, s := range scores {
			index := int(s.(float64) - 1)
			vc[index] += 1
		}
		sum := 0.0
		for i, c := range vc {
			w := float64(i) / 4.0
			sum += float64(c) * w
		}
		r["wilson"] = models.Wilson(len(scores), sum)
		vc[0], vc[1], vc[2], vc[3], vc[4] = 0, 0, 0, 0, 0
	}
	sort.Sort(results)
	return AJAX("rankings.html").Execute(w, map[string]interface{}{
		"results": results,
		"ctx":     ctx,
	})
}

func GetPhotoVotes(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	id := req.URL.Query().Get(":id")
	if !bson.IsObjectIdHex(id) {
		return perform_status(w, req, http.StatusForbidden)
	}
	match := bson.M{"photo": bson.ObjectIdHex(id)}
	if f, ok := ctx.Session.Values["filter"]; ok {
		f.(*models.Filter).AddQuery(match)
	}
	var result bson.M
	pipe := ctx.C(V).Pipe([]bson.M{
		{"$match": match},
		{"$group": bson.M{
			"_id":   "$photo",
			"avg":   bson.M{"$avg": "$score"},
			"count": bson.M{"$sum": 1},
			"user":  bson.M{"$addToSet": "$photouser"},
		}},
		{"$unwind": "$user"},
	})
	pipe.One(&result)
	if result["user"] != nil && result["user"].(bson.ObjectId) != ctx.User.Id {
		return perform_status(w, req, http.StatusForbidden)
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"avg": %.1f, "count": %d}`, result["avg"], result["count"])
	return nil
}
