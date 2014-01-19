package controllers

import (
	"app/models"
	"labix.org/v2/mgo/bson"
	"net/http"
)

func CommentForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	//set up the collection and query
	id := req.URL.Query().Get(":id")
	kind := req.URL.Query().Get(":kind")
	if !bson.IsObjectIdHex(id) {
		return perform_status(w, req, http.StatusForbidden)
	}
	var object models.Commenter
	switch kind {
	case "p":
		query := ctx.C(P).FindId(bson.ObjectIdHex(id))

		//execute the query
		photo := &models.Photo{}
		if err := query.One(&photo); err != nil {
			return perform_status(w, req, http.StatusNotFound)
		}
		object = photo
	case "c":
		query := ctx.C(C).FindId(bson.ObjectIdHex(id))
		//execute the query
		contest := &models.Contest{}
		if err := query.One(&contest); err != nil {
			return perform_status(w, req, http.StatusNotFound)
		}
		object = contest
	}

	//execute the template
	return AJAX("comments.html").Execute(w, map[string]interface{}{
		"object": object,
		"kind":   kind,
		"ctx":    ctx,
	})
}

func Comment(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}

	if req.FormValue("csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	form := models.CommentForm
	r := form.Load(req)
	ctx.Data["result"] = r
	if len(r.Errors) != 0 {
		return CommentForm(w, req, ctx)
	}

	id := req.URL.Query().Get(":id")
	if !bson.IsObjectIdHex(id) {
		return perform_status(w, req, http.StatusForbidden)
	}
	c := &models.Comment{
		Id:       bson.NewObjectId(),
		User:     ctx.User.Id,
		UserName: ctx.User.FullName(),
		Avatar:   ctx.User.Avatar,
		Body:     r.Values["body"],
	}
	col := P
	if req.URL.Query().Get(":kind") == "c" {
		col = C
	}
	if err := ctx.C(col).UpdateId(bson.ObjectIdHex(id), bson.M{"$push": bson.M{"comments": c}}); err != nil {
		r.Errors["body"] = err
		return CommentForm(w, req, ctx)
	}
	return CommentForm(w, req, ctx)
}
