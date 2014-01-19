package controllers

import (
	"app/models"
	"bytes"
	crand "crypto/rand"
	"fmt"
	"github.com/rif/forms"
	"html/template"
	"image/jpeg"
	"labix.org/v2/mgo/bson"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

var layerTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"reverse": reverse,
	"neq":     neq,
	"trans":   trans,
}).Parse(`
<div>
<p class='galleria-info-title'>{{.p.Title}}</p>
<p class='galleria-info-description'>{{.p.Description}}</p>
{{ if .ctx.User }}
{{ if neq .ctx.User.Id .p.User }}
<span id='vote-0' href='{{ reverse "get_vote" "photo" .p.Id.Hex "contest" "" }}'></span>
{{ end }}
{{ else }}
<p>{{ trans "Login to rate this photo" .ctx }}.</p>
{{ end }}
<span class='btn-group'>
<a id='ext-link' tip='{{ trans "Link" .ctx }}' class='btn btn-mini btn-link' href='{{reverse "external_photo" "id" .p.User.Hex "kind" "p" "photo" .p.Id.Hex }}' target='_blank'><i class='icon-white icon-share'></i></a>
{{ if .ctx.User }}
{{ if neq .ctx.User.Id .p.User }}
<a tip='{{ trans "Fake" .ctx }}' id='fake-link' class='btn btn-mini btn-link' href='{{ reverse "fake" "photo" .p.Id.Hex "csrf_token" .ctx.Session.Values.csrf_token }}'><i class='icon-white icon-thumbs-down'></i></a>
<a tip='{{ trans "Abuse" .ctx }}' id='abuse-link' class='btn btn-mini btn-link' href='{{ reverse "abuse" "photo" .p.Id.Hex  "csrf_token" .ctx.Session.Values.csrf_token }}'><i class='icon-white icon-fire'></i></a>
<a tip='{{ trans "Message" .ctx }}' id='mes-link' href='{{ reverse "send_message" "to" .p.User.Hex}}' class='btn btn-link btn-mini'><i class='icon-white icon-comment'></i></a>
{{ end }}
{{ end }}
</span>
<span class='muted' id='tip'></span>
<a style='display:none;' class='comment-link' href='{{ reverse "comments" "kind" "p" "id" .p.Id.Hex }}'></a>
</div>
`))

func UploadForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	//set up the collection and query
	query := ctx.C(P).Find(bson.M{"user": ctx.User.Id, "deleted": false}).Sort("-_id")

	var photos []*models.Photo
	if err = query.All(&photos); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}

	// get id for edit mode
	id := req.URL.Query().Get(":id")
	if bson.IsObjectIdHex(id) {
		p := &models.Photo{}
		if err := ctx.C(P).Find(bson.M{"_id": bson.ObjectIdHex(id), "user": ctx.User.Id, "deleted": false}).One(p); err != nil {
			return perform_status(w, req, http.StatusNotFound)
		}
		if p == nil {
			return perform_status(w, req, http.StatusNotFound)
		}
		r, ok := ctx.Data["result"]
		a := ""
		if p.Active {
			a = "yes"
		}
		if !ok {
			r = forms.Result{
				Values: map[string]string{
					"title":       p.Title,
					"description": p.Description,
					"country":     p.Country,
					"location":    p.Location,
					"age":         strconv.Itoa(p.Age),
					"gender":      p.Gender,
					"active":      a,
				},
			}
			ctx.Data["result"] = r
		}
	}

	// default values
	r, ok := ctx.Data["result"]
	if !ok {
		r = forms.Result{Values: map[string]string{"active": "yes"}}
		ctx.Data["result"] = r
		r.(forms.Result).Values["country"] = ctx.User.Country
		r.(forms.Result).Values["location"] = ctx.User.Location
		r.(forms.Result).Values["gender"] = ctx.User.Gender
		r.(forms.Result).Values["age"] = strconv.Itoa(ctx.User.Age())
	}
	// find the index of the photo
	ctx.Data["index"] = 0
	for i, p := range photos {
		if p.Id.Hex() == id {
			ctx.Data["index"] = i
			break
		}
	}

	return T("upload.html").Execute(w, map[string]interface{}{
		"photos": photos,
		"id":     id,
		"ctx":    ctx,
	})
}

func Upload(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	form := models.UploadForm
	r := (&form).Load(req)
	ctx.Data["result"] = r
	if len(r.Errors) != 0 {
		return UploadForm(w, req, ctx)
	}
	p := r.Value.(map[string]interface{})
	rand, err := crand.Int(crand.Reader, big.NewInt(1000000))
	if err != nil {
		models.Log("error generating random number:", err.Error())
	}
	photo := &models.Photo{
		Title:       p["title"].(string),
		Description: p["description"].(string),
		Country:     p["country"].(string),
		Location:    p["location"].(string),
		Age:         p["age"].(int),
		Gender:      p["gender"].(string),
		Active:      p["active"].(bool),
		Deleted:     false,
		User:        ctx.User.Id,
		UpdatedOn:   time.Now(),
		Rand:        rand.Int64(),
	}
	var objectToBeUpdated interface{}
	var nid bson.ObjectId
	id := ""
	if id = req.URL.Query().Get(":id"); bson.IsObjectIdHex(id) { //edit mode
		nid = bson.ObjectIdHex(id)
		if nid != ctx.User.Id { // doesn't belong to the registered user
			return perform_status(w, req, http.StatusForbidden)
		}
		objectToBeUpdated = p
		// update previous votes
		if _, err := ctx.C(V).UpdateAll(bson.M{"photo": nid}, bson.M{"$set": bson.M{
			"title":       photo.Title,
			"description": photo.Description,
			"country":     photo.Country,
			"location":    photo.Location,
			"age":         photo.Age,
			"gender":      photo.Gender,
			"active":      photo.Active,
		}}); err != nil {
			models.Log("Error updating votes on photo edit: ", err.Error())
		}

	} else { // upload mode
		objectToBeUpdated = photo
		x1, _ := strconv.Atoi(req.FormValue("x1"))
		y1, _ := strconv.Atoi(req.FormValue("y1"))
		x2, _ := strconv.Atoi(req.FormValue("x2"))
		y2, _ := strconv.Atoi(req.FormValue("y2"))
		ff, _, err := req.FormFile("photo")
		if err != nil {
			ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem uploading photo:", ctx), err.Error()))
			models.Log(err.Error())
			return UploadForm(w, req, ctx)
		}
		defer ff.Close()
		// decode jpeg into image.Image
		img, err := jpeg.Decode(ff)
		if err != nil {
			ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem uploading photo:", ctx), err.Error()))
			models.Log(err.Error())
			r.Err = err
			return UploadForm(w, req, ctx)
		}
		photo.Id = bson.NewObjectId()
		nid = photo.Id
		if err := photo.SaveImage(img, x1, y1, x2, y2); err != nil {
			ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem uploading photo:", ctx), err.Error()))
			models.Log(err.Error())
			r.Err = err
			return UploadForm(w, req, ctx)
		}
	}
	photo.Id = ""
	if _, err := ctx.C(P).UpsertId(nid, bson.M{"$set": objectToBeUpdated}); err != nil {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem editing photo:", ctx), err.Error()))
		models.Log(err.Error())
		r.Err = err
		return UploadForm(w, req, ctx)
	}
	ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Photo updated succesfully!", ctx)))
	http.Redirect(w, req, reverse("upload", "id", id), http.StatusSeeOther)
	return nil
}

func Delete(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
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

	if rc, _ := ctx.C(C).Find(bson.M{"registered.photo": bson.ObjectIdHex(id)}).Count(); rc != 0 {
		// the photo is registered in contests
		// only mark as deleted
		if err := ctx.C(P).UpdateId(bson.ObjectIdHex(id), bson.M{"$set": bson.M{"deleted": true, "active": false}}); err != nil {
			return perform_status(w, req, http.StatusNotFound)
		}
		// delete non contest votes
		// delete related votes
		if _, err := ctx.C(V).RemoveAll(bson.M{"photo": bson.ObjectIdHex(id), "contest": bson.M{"$exists": false}}); err != nil {
			models.Log("Error deleting votes on photo delete: ", err.Error())
		}
	} else {
		// the photo is not registered in any contest
		if err := ctx.C(P).RemoveId(bson.ObjectIdHex(id)); err != nil {
			return perform_status(w, req, http.StatusNotFound)
		}
		// delete from disk
		err := os.Remove(path.Join(models.DATA_DIR, models.UPLOADS, fmt.Sprintf("%s.jpg", id)))
		if err != nil {
			models.Log("Error deleting image:", err.Error())
		}
		err = os.Remove(path.Join(models.DATA_DIR, models.UPLOADS, fmt.Sprintf("%s_thumb.jpg", id)))
		if err != nil {
			models.Log("Error deleting image:", err.Error())
		}

		// delete related votes
		if _, err := ctx.C(V).RemoveAll(bson.M{"photo": bson.ObjectIdHex(id)}); err != nil {
			models.Log("Error deleting votes on photo delete: ", err.Error())
		}
	}

	http.Redirect(w, req, reverse("upload", "id", ""), http.StatusSeeOther)
	return nil
}

func Photos(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	id := req.URL.Query().Get(":id")
	if !bson.IsObjectIdHex(id) {
		return perform_status(w, req, http.StatusNotFound)
	}
	var photos []*models.Photo
	if err := ctx.C(P).Find(bson.M{"user": bson.ObjectIdHex(id), "active": true}).All(&photos); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}
	user := new(models.User)
	if err := ctx.C("users").FindId(bson.ObjectIdHex(id)).One(user); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}
	// find the index of the photo
	photoId := req.URL.Query().Get(":photo")
	ctx.Data["index"] = 0
	var pIds []bson.ObjectId
	for i, p := range photos {
		if p.Id.Hex() == photoId {
			ctx.Data["index"] = i
		}
		pIds = append(pIds, p.Id)
	}

	return AJAX("galleria.html").Execute(w, map[string]interface{}{
		"photos": photos,
		"user":   user,
		"hash":   models.GenUUID(),
		"ctx":    ctx,
	})
}

func Empty(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	var photos []*models.Photo
	ctx.Data["index"] = 0
	return AJAX("galleria.html").Execute(w, map[string]interface{}{
		"photos": photos,
		"hash":   "0",
		"ctx":    ctx,
	})
}

func TopVoted(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	page := 1
	page, err := strconv.Atoi(req.URL.Query().Get(":page"))
	if err != nil && page == 0 {
		page = 1
	}
	skip := ITEMS_PER_PAGE * (page - 1)
	match := bson.M{"active": true}
	if f, ok := ctx.Session.Values["filter"]; ok {
		f.(*models.Filter).AddQuery(match)
	}
	var results models.WilsonSorter
	pipe := ctx.C(V).Pipe([]bson.M{
		{"$match": match},
		{"$group": bson.M{
			"_id":         "$photo",
			"scores":      bson.M{"$push": "$score"},
			"title":       bson.M{"$addToSet": "$title"},
			"description": bson.M{"$addToSet": "$description"},
			"user":        bson.M{"$addToSet": "$photouser"},
		}},
		{"$skip": skip},
		{"$limit": ITEMS_PER_PAGE},
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

	data := ""
	var layer bytes.Buffer
	for _, r := range results {
		p := &models.Photo{
			Id:          r["_id"].(bson.ObjectId),
			User:        r["user"].(bson.ObjectId),
			Title:       r["title"].(string),
			Description: r["description"].(string),
		}
		err := layerTemplate.Execute(&layer, map[string]interface{}{"p": p, "ctx": ctx})
		if err != nil {
			models.Log("layer template: ", err.Error())
		}
		data += fmt.Sprintf(`{"image":"%s","thumb":"%s","title":"%s","description":"%s", "layer":"%s"},`,
			models.ImageUrl(p.Id.Hex(), ""),
			models.ImageUrl(p.Id.Hex(), "thumb"),
			p.Title,
			p.Description,
			strings.Replace(layer.String(), "\n", "", -1),
		)
		layer.Reset()
	}

	data = strings.TrimRight(data, ",")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "["+data+"]")
	return nil
}

func Latest(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	page := 1
	page, err := strconv.Atoi(req.URL.Query().Get(":page"))
	if err != nil && page == 0 {
		page = 1
	}
	skip := ITEMS_PER_PAGE * (page - 1)
	query := bson.M{"active": true}
	if f, ok := ctx.Session.Values["filter"]; ok {
		f.(*models.Filter).AddQuery(query)
	}
	var photos []*models.Photo
	if err := ctx.C("photos").Find(query).Skip(skip).Limit(ITEMS_PER_PAGE).Sort("-_id").All(&photos); err != nil {
		return internal_error(w, req, err.Error())
	}
	data := ""
	var layer bytes.Buffer
	for _, p := range photos {
		err := layerTemplate.Execute(&layer, map[string]interface{}{"p": p, "ctx": ctx})
		if err != nil {
			models.Log("layer template: ", err.Error())
		}
		data += fmt.Sprintf(`{"image":"%s","thumb":"%s","title":"%s","description":"%s", "layer":"%s"},`,
			models.ImageUrl(p.Id.Hex(), ""),
			models.ImageUrl(p.Id.Hex(), "thumb"),
			p.Title,
			p.Description,
			strings.Replace(layer.String(), "\n", "", -1),
		)
		layer.Reset()
	}

	data = strings.TrimRight(data, ",")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "["+data+"]")
	return nil
}

func Random(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	page := 1
	page, err := strconv.Atoi(req.URL.Query().Get(":page"))
	if err != nil && page == 0 {
		page = 1
	}
	skip := ITEMS_PER_PAGE * (page - 1)
	query := bson.M{"active": true}
	if f, ok := ctx.Session.Values["filter"]; ok {
		f.(*models.Filter).AddQuery(query)
	}

	//execute the query
	var photos []*models.Photo
	if err := ctx.C("photos").Find(query).Skip(skip).Limit(ITEMS_PER_PAGE).Sort("-_id").All(&photos); err != nil {
		return internal_error(w, req, err.Error())
	}
	data := ""
	var layer bytes.Buffer
	for _, i := range rand.Perm(len(photos)) {
		p := photos[i]
		err := layerTemplate.Execute(&layer, map[string]interface{}{"p": p, "ctx": ctx})
		if err != nil {
			models.Log("layer template: ", err.Error())
		}
		data += fmt.Sprintf(`{"image":"%s","thumb":"%s","title":"%s","description":"%s","layer":"%s"},`,
			models.ImageUrl(p.Id.Hex(), ""),
			models.ImageUrl(p.Id.Hex(), "thumb"),
			p.Title,
			p.Description,
			strings.Replace(layer.String(), "\n", "", -1),
		)
		layer.Reset()
	}

	data = strings.TrimRight(data, ",")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "["+data+"]")
	return nil
}

func Fake(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	return report(w, req, ctx, "fake")
}

func Abuse(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	return report(w, req, ctx, "abuse")
}

func report(w http.ResponseWriter, req *http.Request, ctx *models.Context, repType string) error {
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
	query := bson.M{"_id": bson.ObjectIdHex(photoId), "active": true, repType + "reporters": bson.M{"$ne": ctx.User.Id}}
	update := bson.M{
		"$push": bson.M{repType + "reporters": ctx.User.Id},
		"$inc":  bson.M{repType + "count": 1},
	}
	if err := ctx.C(P).Update(query, update); err != nil {
		// toggle report
		// This query succeeds when the voter has already voted on the story.
		//query   = {_id: ObjectId("4bcc9e697e020f2d44471d27"), voters: user_id};

		// Update to remove the user from the array and decrement the number of votes.
		//update  = {'$pull': {'voters': user_id}, '$inc': {vote_count: -1}}

		//db.stories.update(query, update);
	}
	return nil
}

func SetAvatar(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		return perform_status(w, req, http.StatusForbidden)
	}
	if req.URL.Query().Get(":csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	photoId := req.URL.Query().Get(":photo")
	if bson.IsObjectIdHex(photoId) {
		newAvatar := models.ImageUrl(photoId, "thumb")
		ctx.User.Avatar = newAvatar
		ctx.C(U).UpdateId(ctx.User.Id, bson.M{"$set": bson.M{"avatar": newAvatar}})
		//ctx.C(P).Update(bson.M{"comments.user": ctx.User.Id}, bson.M{"comments.$.avatar": bson.M{"$set": ctx.User.Gravatar(80)}})
	}
	return nil
}

func ExternalPhoto(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	id := req.URL.Query().Get(":id")
	photoId := req.URL.Query().Get(":photo")
	if !bson.IsObjectIdHex(id) || (photoId != "" && !bson.IsObjectIdHex(photoId)) {
		return perform_status(w, req, http.StatusForbidden)
	}
	muxName := "photos"
	if req.URL.Query().Get(":kind") == "c" {
		muxName = "view_contest"
	}
	return T("ajax_wrapper.html").Execute(w, map[string]interface{}{
		"ajaxurl": reverse(muxName, "id", id, "photo", photoId),
		"ctx":     ctx,
	})
	return nil
}
