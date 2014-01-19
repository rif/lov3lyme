package controllers

import (
	"app/models"
	"labix.org/v2/mgo/bson"
	"net/http"
)

func Admin(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil || !ctx.User.Admin {
		return perform_status(w, req, http.StatusForbidden)
	}

	photoCount, _ := ctx.C(P).Find(nil).Count()
	pp := NewPagination(photoCount, req.URL.Query())
	photoSkip := pp.PerPage * (pp.Current - 1)
	var photos []*models.Photo
	if err := ctx.C(P).Find(nil).Skip(photoSkip).Limit(pp.PerPage).All(&photos); err != nil {
		models.Log("error getting photos: ", err.Error())
		return err
	}

	userCount, _ := ctx.C(U).Find(nil).Count()
	up := NewPagination(userCount, req.URL.Query())
	userSkip := up.PerPage * (up.Current - 1)
	var users []*models.User
	if err := ctx.C(U).Find(nil).Skip(userSkip).Limit(up.PerPage).All(&users); err != nil {
		models.Log("error getting users: ", err.Error())
		return err
	}

	return T("admin.html").Execute(w, map[string]interface{}{
		"ctx":    ctx,
		"pp":     pp,
		"photos": photos,
		"up":     up,
		"users":  users,
	})
}

func DelUser(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil || !ctx.User.Admin {
		return perform_status(w, req, http.StatusForbidden)
	}
	id := req.URL.Query().Get(":id")
	if !bson.IsObjectIdHex(id) {
		return perform_status(w, req, http.StatusForbidden)
	}
	if err := ctx.C(U).RemoveId(bson.ObjectIdHex(id)); err != nil {
		models.Log("error deleting user: ", err.Error())
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem deleting user:", ctx), err.Error()))
		http.Redirect(w, req, reverse("admin"), http.StatusSeeOther)
		return err
	}
	ctx.Session.AddFlash(models.F(models.SUCCESS, trans("User deleted!", ctx)))
	http.Redirect(w, req, reverse("admin"), http.StatusSeeOther)
	return nil
}

func DelPhoto(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil || !ctx.User.Admin {
		return perform_status(w, req, http.StatusForbidden)
	}
	http.Redirect(w, req, reverse("admin"), http.StatusSeeOther)
	return nil
}
