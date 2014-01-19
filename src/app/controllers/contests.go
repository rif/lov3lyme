package controllers

import (
	"app/models"
	"bytes"
	"errors"
	"fmt"
	"github.com/rif/forms"
	"html/template"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strconv"
	"time"
)

var (
	fm = template.FuncMap{
		"reverse": reverse,
		"trunc":   truncateString,
		"trans":   trans,
	}
	admissionTemplate = template.Must(template.New("adm").Funcs(fm).Parse(`{{ .c.Name }} - {{ if .ctx.User }}<a data-toggle="modal" data-target="#cmo-modal" href="{{ reverse "register_contest" "id" .c.Id.Hex }}"><i class="icon-edit"></i> Enroll in contest</a>{{else}}{{ trans "Login to enroll" .ctx }}{{end}}<p class="indented"><small class="muted">{{trunc .c.Description 160}}</small></p>`))
	votingTemplate    = template.Must(template.New("vot").Funcs(fm).Parse(`<a data-toggle="modal" data-target="#cmo-modal" href="{{ reverse "view_contest" "id" .c.Id.Hex "photo" "" }}">{{ .c.Name }}</a><p class="indented"><small class="muted">{{trunc .c.Description 160}}</small></p>`))
	finishedTemplate  = template.Must(template.New("fin").Funcs(fm).Parse(`{{ .c.Name }}<p class="indented"><small class="muted">{{trunc .c.Description 160}}</small></p>`))
)

func ContestForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	// get id for edit mode
	id := req.URL.Query().Get(":id")
	if id != "" {
		c := &models.Contest{}
		if err := ctx.C(C).Find(bson.M{"_id": bson.ObjectIdHex(id), "user": ctx.User.Id}).One(c); err != nil {
			return perform_status(w, req, http.StatusNotFound)
		}
		if c == nil {
			return perform_status(w, req, http.StatusNotFound)
		}
		r, ok := ctx.Data["result"]
		v := ""
		if c.RequireApproval {
			v = "yes"
		}
		if !ok {
			r = forms.Result{
				Values: map[string]string{
					"name":               c.Name,
					"description":        c.Description,
					"country":            c.Country,
					"location":           c.Location,
					"gender":             c.Gender,
					"min_age":            strconv.Itoa(c.MinAge),
					"max_age":            strconv.Itoa(c.MaxAge),
					"admission_deadline": c.AdmissionDeadline.Format("2006-01-02"),
					"voting_deadline":    c.VotingDeadline.Format("2006-01-02"),
					"require_approval":   v,
				},
			}
			ctx.Data["result"] = r
		}
	}
	var contests []*models.Contest
	query := bson.M{"user": ctx.User.Id}
	max, _ := ctx.C(C).Find(query).Count()
	p := NewPagination(max, req.URL.Query())
	skip := p.PerPage * (p.Current - 1)

	ctx.C(C).Find(query).Skip(skip).Limit(p.PerPage).All(&contests)

	return T("contests.html").Execute(w, map[string]interface{}{
		"ctx":      ctx,
		"id":       id,
		"contests": contests,
		"p":        p,
	})
}

func Contest(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	if req.FormValue("csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	r := models.ContestForm.Load(req)
	ctx.Data["result"] = r
	if len(r.Errors) != 0 {
		return ContestForm(w, req, ctx)
	}
	c := r.Value.(map[string]interface{})
	gender := c["gender"].(string)
	if gender != "m" && gender != "f" {
		r.Errors["gender"] = errors.New("Please select Male or Female")
	}
	now := time.Now()
	at := c["admission_deadline"].(time.Time)
	vt := c["voting_deadline"].(time.Time)
	if at.Before(now) || at.Equal(now) {
		r.Errors["admission_deadline"] = errors.New("Must be in the future")
	}
	if at.After(now.AddDate(0, 1, 1)) {
		r.Errors["admission_deadline"] = errors.New("Admission deadline can be maximum one month from now")
	}
	if vt.Before(at) || vt.Equal(at) {
		r.Errors["voting_deadline"] = errors.New("Must be in after admission deadline")
	}
	if at.After(now.AddDate(0, 2, 1)) {
		r.Errors["admission_deadline"] = errors.New("Voting deadline can be maximum two months from now")
	}
	if len(r.Errors) != 0 {
		return ContestForm(w, req, ctx)
	}
	contest := &models.Contest{
		Name:              c["name"].(string),
		Description:       c["description"].(string),
		Country:           c["country"].(string),
		Location:          c["location"].(string),
		Gender:            c["gender"].(string),
		MinAge:            c["min_age"].(int),
		MaxAge:            c["max_age"].(int),
		AdmissionDeadline: c["admission_deadline"].(time.Time),
		VotingDeadline:    c["voting_deadline"].(time.Time),
		RequireApproval:   c["require_approval"].(bool),
		Public:            false,
		User:              ctx.User.Id,
	}
	var nid bson.ObjectId
	if id := req.URL.Query().Get(":id"); bson.IsObjectIdHex(id) { //edit mode
		nid = bson.ObjectIdHex(id)
	} else {
		nid = bson.NewObjectId()
	}
	query := bson.M{"_id": nid, "user": ctx.User.Id, "public": false}
	if _, err := ctx.C(C).Upsert(query, bson.M{"$set": contest}); err != nil {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem updating contest:", ctx), err.Error()))
		models.Log(err.Error())
		models.Log("contest creation/updating: ", err.Error())
		r.Err = err // for pnotify
		return ContestForm(w, req, ctx)
	}
	ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Contest updated succesully!", ctx)))
	http.Redirect(w, req, reverse("contest", "id", ""), http.StatusSeeOther)
	return nil
}

func DeleteContest(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	if req.URL.Query().Get(":csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}

	did := bson.ObjectIdHex(req.URL.Query().Get(":id"))
	// delete from db
	query := bson.M{"_id": did, "user": ctx.User.Id}

	if err := ctx.C(C).Remove(query); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}

	// clean related votes (not delete them but remove the contest)
	query = bson.M{"contest": did}
	if _, err := ctx.C(V).UpdateAll(query, bson.M{"$unset": bson.M{"contest": 1}}); err != nil {
		models.Log("Error cleaning votes on contest delete: ", err.Error())
	}

	http.Redirect(w, req, reverse("contest", "id", ""), http.StatusSeeOther)
	return nil
}

func RegisterContestForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	id := req.URL.Query().Get(":id")
	contest := &models.Contest{}
	if err := ctx.C(C).FindId(bson.ObjectIdHex(id)).One(contest); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}

	var photos []*models.Photo
	if err := ctx.C(P).Find(bson.M{"user": ctx.User.Id, "active": true}).All(&photos); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}
	return AJAX("register_contest.html").Execute(w, map[string]interface{}{
		"contest": contest,
		"photos":  photos,
		"ctx":     ctx,
	})
}

func RegisterContest(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	if req.FormValue("csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	id := req.URL.Query().Get(":id")
	photoId := req.FormValue("photo")
	if !bson.IsObjectIdHex(photoId) || !bson.IsObjectIdHex(id) {
		return perform_status(w, req, http.StatusNotFound)
	}

	contest := &models.Contest{}
	if err := ctx.C(C).FindId(bson.ObjectIdHex(id)).One(contest); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}
	photo := &models.Photo{}
	if err := ctx.C(P).FindId(bson.ObjectIdHex(photoId)).One(photo); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}
	if !contest.CanRegister() {
		return perform_status(w, req, http.StatusForbidden)
	}

	ri := &models.RegItem{
		User:        ctx.User.Id,
		UserName:    ctx.User.FullName(),
		UserInfo:    ctx.User.Info(),
		Photo:       photo.Id,
		Title:       photo.Title,
		Description: photo.Description,
		Approved:    !contest.RequireApproval || contest.User == ctx.User.Id,
	}
	query := bson.M{"_id": contest.Id, "registered.user": bson.M{"$ne": ctx.User.Id}}
	if err := ctx.C(C).Update(query, bson.M{"$push": bson.M{"registered": ri}}); err != nil {
		// allready in
		query := bson.M{"_id": contest.Id, "registered.user": ctx.User.Id}
		if err = ctx.C(C).Update(query, bson.M{"$set": bson.M{
			"registered.$.photo":       ri.Photo,
			"registered.$.title":       ri.Title,
			"registered.$.description": ri.Description,
			"registered.$.approved":    ri.Approved,
		}}); err != nil {
			models.Log("error updating contest registered list: ", err.Error())
		}
	}

	ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Registration succesfull!", ctx)))
	http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
	return nil
}

func PendingApprovals(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	id := req.URL.Query().Get(":id")
	if !bson.IsObjectIdHex(id) {
		return perform_status(w, req, http.StatusForbidden)
	}
	contest := &models.Contest{}
	if err := ctx.C(C).FindId(bson.ObjectIdHex(id)).One(contest); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}
	// check user owns contest
	if contest.User != ctx.User.Id {
		return perform_status(w, req, http.StatusForbidden)
	}
	return AJAX("pending_approval.html").Execute(w, map[string]interface{}{
		"contest": contest,
		"ctx":     ctx,
	})
	return nil
}

func ContestStatus(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	id := req.URL.Query().Get(":id")
	if !bson.IsObjectIdHex(id) {
		return perform_status(w, req, http.StatusForbidden)
	}
	contest := &models.Contest{}
	if err := ctx.C(C).FindId(bson.ObjectIdHex(id)).One(contest); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}
	// check user owns contest
	if contest.User != ctx.User.Id {
		return perform_status(w, req, http.StatusForbidden)
	}
	return AJAX("contest_status.html").Execute(w, map[string]interface{}{
		"contest": contest,
		"ctx":     ctx,
	})
	return nil
}

func ApproveContest(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	if req.URL.Query().Get(":csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	id := req.URL.Query().Get(":id")
	userId := req.URL.Query().Get(":user")
	if !bson.IsObjectIdHex(id) || !bson.IsObjectIdHex(userId) {
		return perform_status(w, req, http.StatusForbidden)
	}
	// TODO: veryfy if contest belongs to the ctx.User

	res := req.URL.Query().Get(":res")
	switch res {
	case "y":
		query := bson.M{"_id": bson.ObjectIdHex(id), "registered.user": bson.ObjectIdHex(userId)}
		ctx.C(C).Update(query, bson.M{"$set": bson.M{"registered.$.approved": true}})
	case "n":
		// removing from registered list
		ctx.C(C).UpdateId(bson.ObjectIdHex(id), bson.M{"$pull": bson.M{"registered": bson.M{"user": bson.ObjectIdHex(userId)}}})
	}
	return nil
}

func PublishContest(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	if req.URL.Query().Get(":csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	id := req.URL.Query().Get(":id")
	if !bson.IsObjectIdHex(id) {
		return perform_status(w, req, http.StatusForbidden)
	}
	if err := ctx.C(C).UpdateId(bson.ObjectIdHex(id), bson.M{"$set": bson.M{"public": true}}); err != nil {
		models.Log("error making contest public: ", err.Error())
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Failed to make project public: ", ctx), err.Error()))
	} else {
		ctx.Session.AddFlash(models.F(models.SUCCESS, trans("The contest is now public!", ctx)))
	}
	http.Redirect(w, req, reverse("contest", "id", ""), http.StatusSeeOther)
	return nil
}

func ViewContest(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	id := req.URL.Query().Get(":id")
	if !bson.IsObjectIdHex(id) {
		return perform_status(w, req, http.StatusForbidden)
	}
	contest := &models.Contest{}
	ctx.C(C).FindId(bson.ObjectIdHex(id)).One(contest)
	ctx.Data["index"] = 0
	return AJAX("galleria.html").Execute(w, map[string]interface{}{
		"photos":  contest.Registered,
		"contest": contest,
		"hash":    models.GenUUID(),
		"ctx":     ctx,
	})
	return nil
}

func ContestList(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	var contests []*models.Contest
	query := bson.M{"public": true}
	if f, ok := ctx.Session.Values["filter"]; ok {
		f.(*models.Filter).AddContestQuery(query)
	}
	list := req.URL.Query().Get(":list")
	now := time.Now()
	var t *template.Template
	switch list {
	case "adm":
		query["admissiondeadline"] = bson.M{"$gt": now}
		t = admissionTemplate
	case "vot":
		query["admissiondeadline"] = bson.M{"$lt": now}
		query["votingdeadline"] = bson.M{"$gt": now}
		t = votingTemplate
	case "fin":
		query["votingdeadline"] = bson.M{"$lt": now}
		t = finishedTemplate
	case "pop":

	default:
		return nil
	}
	if err := ctx.C(C).Find(query).Sort("-_id").All(&contests); err != nil {
		return internal_error(w, req, err.Error())
	}
	cl := ""
	var text bytes.Buffer
	for _, c := range contests {
		t.Execute(&text, map[string]interface{}{"ctx": ctx, "c": c})
		cl += fmt.Sprintf("%s", text.String())
		text.Reset()
	}
	if cl == "" {
		cl = "<p>" + trans("No contest is matching selected criteria", ctx) + ".</p>"
	}
	fmt.Fprintf(w, cl)
	return nil
}
