package controllers

import (
	"app/models"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"code.google.com/p/goauth2/oauth"
)

const (
	DOMAIN_URL = "http://lov3lyme-rif.rhcloud.com"
)

func FbConfig() *oauth.Config {
	config := &oauth.Config{
		ClientId:     "*****",
		ClientSecret: "*****",
		Scope:        "user_birthday, email, user_location",
		AuthURL:      "https://www.facebook.com/dialog/oauth",
		TokenURL:     "https://graph.facebook.com/oauth/access_token",
		RedirectURL:  DOMAIN_URL + "/fblogin",
	}
	return config
}

func FbLogin(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	// Set up a configurgation

	if auth_error := req.FormValue("error"); auth_error != "" {
		models.Log("Facebook login error: ", auth_error)
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}

	code := req.FormValue("code")
	if code == "" {
		return nil
	}

	config := FbConfig()

	// Set up a Transport with our config, define the cache
	t := &oauth.Transport{Config: config}
	if _, err := t.Exchange(code); err != nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		models.Log("Facebook exchange: ", err.Error())
		return err
	}

	// Make the request.
	r, err := t.Client().Get("https://graph.facebook.com/me")
	if err != nil {
		models.Log("Facebook profile request: ", err.Error())
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
	}
	defer r.Body.Close()
	profile, err := ioutil.ReadAll(r.Body)
	if err != nil {
		models.Log("Facebook profile read: ", err.Error())
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return err
	}
	p := models.FacebookProfile{}
	err = json.Unmarshal(profile, &p)
	if err != nil {
		models.Log("Facebook unmarsahlling fb profile: ", err.Error())
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return err
	}
	user, redirect, err := models.LoginWithFacebook(ctx, &p)

	//store the user id in the values and redirect to index
	ctx.Session.Values["user"] = user.Id
	http.Redirect(w, req, reverse(redirect), http.StatusSeeOther)
	return nil
}

func GlConfig() *oauth.Config {
	config := &oauth.Config{
		ClientId:     "*******",
		ClientSecret: "*******",
		Scope:        "https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/userinfo.email",
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
		RedirectURL:  DOMAIN_URL + "/gllogin",
	}
	return config
}

func GlLogin(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	// Set up a configuration

	if auth_error := req.FormValue("error"); auth_error != "" {
		models.Log("Google login error: ", err.Error())
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}

	code := req.FormValue("code")
	if code == "" {
		return nil
	}

	config := GlConfig()

	// Set up a Transport with our config, define the cache
	t := &oauth.Transport{Config: config}
	if _, err := t.Exchange(code); err != nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		models.Log("Google exchange: ", err.Error())
		return err
	}

	// Make the request.
	r, err := t.Client().Get("https://www.googleapis.com/oauth2/v1/userinfo")
	if err != nil {
		models.Log("Google profile request: ", err.Error())
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
	}
	defer r.Body.Close()
	profile, err := ioutil.ReadAll(r.Body)
	if err != nil {
		models.Log("Google profile read error: ", err.Error())
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return err
	}
	p := models.GoogleProfile{}
	err = json.Unmarshal(profile, &p)
	if err != nil {
		models.Log("Google unmarsahlling gl profile: ", err.Error())
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return err
	}
	user, redirect, err := models.LoginWithGoogle(ctx, &p)

	//store the user id in the values and redirect to index
	ctx.Session.Values["user"] = user.Id
	http.Redirect(w, req, reverse(redirect), http.StatusSeeOther)
	return nil
}
