package controllers

import (
	"app/models"
	"bytes"
	"fmt"
	"html/template"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strings"
)

var responseTemplate = template.Must(template.New("").Parse(`<table class='user-result'><tr><td class='user-avatar'><img src='{{.Avatar}}'/></td><td><div class='user-name'>{{.FullName}}</div><div class='user-info'>{{ .Sex }} {{ .Age }}, {{ .Location }} - {{ .Country }}</div></td></tr></table>`))

func Search(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	var users []*models.User
	q := req.FormValue("q")
	query := bson.M{"$or": []bson.M{
		{"firstname": bson.RegEx{q, "i"}},
		{"lastname": bson.RegEx{q, "i"}},
	},
	}
	if err := ctx.C(U).Find(query).Select(bson.M{"password": 0, "birthdate": 0}).Limit(10).All(&users); err != nil {
		return err
	}
	response := ""
	var text bytes.Buffer
	for _, u := range users {
		responseTemplate.Execute(&text, u)
		response += fmt.Sprintf(`{"id":"%s","text":"%s", "name":"%s"},`, u.Id.Hex(), text.String(), u.FullName())
		text.Reset()
	}
	response = strings.TrimRight(response, ",")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, "["+response+"]")
	return nil
}
