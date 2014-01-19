package controllers

import (
	"app/models"
	"fmt"
	"net/http"
)

func Index(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if _, ok := ctx.Session.Values["lang"]; !ok {
		detectLanguage(req.Header["Accept-Language"], ctx)
	}
	return T("index.html").Execute(w, map[string]interface{}{
		"ctx": ctx,
	})
}

func Static(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	page := req.URL.Query().Get(":p")
	defer func() {
		if r := recover(); r != nil {
			// redirect to index if the page is not found
			http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
			return
		}
	}()
	return T(fmt.Sprintf("static/%s.html", page)).Execute(w, map[string]interface{}{
		"ctx": ctx,
	})
}

func SetLanguage(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	ctx.Session.Values["lang"] = req.URL.Query().Get(":lang")
	http.Redirect(w, req, req.Header["Referer"][0], http.StatusSeeOther)
	return nil
}

func ContactForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	return T("contact.html").Execute(w, map[string]interface{}{
		"ctx": ctx,
	})
}

func Contact(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if req.FormValue("csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	r := models.ContactForm.Load(req)
	if len(r.Errors) != 0 {
		ctx.Data["result"] = r
		return ContactForm(w, req, ctx)
	}
	body := fmt.Sprintf("Subject: lov3ly.me message\r\n\r\n %s\n%s\n\n%s", r.Values["name"], r.Values["email"], r.Values["message"])
	go func() {
		err := models.SendEmail([]byte(body), "radu@fericean.ro")
		if err != nil {
			models.Log("Error sending mail: ", err.Error())
		}
	}()
	ctx.Session.AddFlash(models.F(models.SUCCESS, "Message sent. Thank you!"))
	return ContactForm(w, req, ctx)
}

func GoogleSiteVerification(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "google-site-verification: google4b899b9e0462f0cd.html")
}

func Robots(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, `User-agent: *
Disallow: /login/
Disallow: /register/
Disallow: /reset/
`)
}
