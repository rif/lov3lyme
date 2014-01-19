package controllers

import (
	"app/models"
	"fmt"
	"labix.org/v2/mgo/bson"
	"net/http"
)

func SendMessageForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	to := req.URL.Query().Get(":to")
	if !bson.IsObjectIdHex(to) {
		return perform_status(w, req, http.StatusForbidden)
	}
	return AJAX("message.html").Execute(w, map[string]interface{}{
		"Id":  to,
		"ctx": ctx,
	})
}

func SendMessage(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	to := req.URL.Query().Get(":to")
	if !bson.IsObjectIdHex(to) {
		return perform_status(w, req, http.StatusForbidden)
	}
	if to == ctx.User.Id.Hex() {
		return perform_status(w, req, http.StatusForbidden)
	}
	m := models.Message{
		Id:       bson.NewObjectId(),
		From:     ctx.User.Id,
		To:       bson.ObjectIdHex(to),
		UserName: ctx.User.FullName(),
		Avatar:   ctx.User.Avatar,
		Subject:  req.FormValue("subject"),
		Body:     req.FormValue("body"),
	}
	if err := ctx.C(M).Insert(m); err != nil {
		models.Log("Error sending message: ", err.Error())
	}
	return nil
}

func Messages(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	var messages []*models.Message
	ctx.C(M).Find(bson.M{"to": ctx.User.Id}).All(&messages)
	return T("messages.html").Execute(w, map[string]interface{}{
		"ctx":      ctx,
		"messages": messages,
	})
}

func DelMessage(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	id := req.URL.Query().Get(":id")
	if !bson.IsObjectIdHex(id) {
		return perform_status(w, req, http.StatusForbidden)
	}

	if err := ctx.C(M).Remove(bson.M{"_id": bson.ObjectIdHex(id), "to": ctx.User.Id}); err != nil {
		models.Log("error removing message: ", err.Error())
		return err
	}
	fmt.Fprint(w, "ok")
	return nil
}
