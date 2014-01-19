package controllers

import (
	"app/models"
	"code.google.com/p/go.crypto/bcrypt"
	"errors"
	"fmt"
	"github.com/dchest/captcha"
	"github.com/rif/forms"
	"github.com/ungerik/go-gravatar"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strings"
	"time"
)

func LoginForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	// should not be logged in
	if ctx.User != nil {
		ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Already logged in!", ctx)))
		http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
		return nil
	}

	return T("login.html").Execute(w, map[string]interface{}{
		"ctx":         ctx,
		"fbLoginLink": FbConfig().AuthCodeURL(""),
		"glLoginLink": GlConfig().AuthCodeURL(""),
	})
}

func Login(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	// should not be logged in
	if ctx.User != nil {
		ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Already logged in!", ctx)))
		http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
		return nil
	}
	email, password := req.FormValue("email"), req.FormValue("password")

	if req.FormValue("csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}

	user, e := models.Login(ctx, email, password)
	if e != nil {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Invalid E-mail/Password", ctx)))
		return LoginForm(w, req, ctx)
	}

	//store the user id in the values and redirect to index
	ctx.Session.Values["user"] = user.Id
	http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
	return nil
}

func Logout(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if req.URL.Query().Get(":csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	delete(ctx.Session.Values, "user")
	http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
	return nil
}

func RegisterForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if ctx.User != nil {
		http.Redirect(w, req, reverse("logout"), http.StatusSeeOther)
		return nil
	}
	ctx.Data["title"] = "Register"
	ctx.Data["cap"] = captcha.New()
	return T("register.html").Execute(w, map[string]interface{}{
		"ctx":         ctx,
		"fbLoginLink": FbConfig().AuthCodeURL(models.GenUUID()),
		"glLoginLink": GlConfig().AuthCodeURL(models.GenUUID()),
	})
}

func Register(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User != nil {
		http.Redirect(w, req, reverse("logout"), http.StatusSeeOther)
		return nil
	}
	if req.FormValue("csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	ctx.Data["title"] = "Register"
	r := (&models.UserForm).Load(req)
	ctx.Data["result"] = r
	if len(r.Errors) != 0 {
		return RegisterForm(w, req, ctx)
	}
	u := r.Value.(map[string]interface{})
	password1 := u["password1"].(string)
	if len(password1) < 5 {
		r.Errors["password1"] = errors.New("Passwords too short (5 chars or more)")
	}
	password2 := u["password2"].(string)
	if password2 != password1 {
		r.Errors["password2"] = errors.New("Passwords do not match")
	}
	gender := u["gender"].(string)
	if gender != "m" && gender != "f" {
		r.Errors["gender"] = errors.New("Please select Male or Female")
	}
	now := time.Now()
	oldest := time.Date(now.Year()-120, 1, 1, 0, 0, 0, 0, time.UTC)
	bDate := u["birthdate"].(time.Time)
	if bDate.Before(oldest) || bDate.After(now) {
		r.Errors["birthdate"] = errors.New("Invalid birth date")
	}
	if len(r.Errors) != 0 {
		return RegisterForm(w, req, ctx)
	}
	if r.Err != nil {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem registering user:", ctx), r.Err.Error()))
		return RegisterForm(w, req, ctx)
	}
	if !captcha.VerifyString(req.FormValue("captchaId"), req.FormValue("captchaSolution")) {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("The control numbers do not match!", ctx)))
		return RegisterForm(w, req, ctx)
	}
	pass, err := models.EncryptPassword(u["password1"].(string))
	if err != nil {
		return internal_error(w, req, err.Error()) // bcrypt errors on invalid costs
	}
	u["password"] = pass
	delete(u, "password1")
	delete(u, "password2")
	u["_id"] = bson.NewObjectId()
	u["avatar"] = gravatar.UrlSize(u["email"].(string), 80)
	if err := ctx.C("users").Insert(u); err != nil {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem registering user:", ctx), err.Error()))
		models.Log(err.Error())
		r.Err = err
		return RegisterForm(w, req, ctx)
	}

	//store the user id in the values and redirect to index
	ctx.Session.Values["user"] = u["_id"]
	ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Welcome to lov3ly.me!", ctx)))
	http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
	return nil
}

func ProfileForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	ctx.Data["title"] = "Profile"
	u := ctx.User

	r, ok := ctx.Data["result"]
	if !ok {
		r = forms.Result{
			Values: map[string]string{
				"firstname": u.FirstName,
				"lastname":  u.LastName,
				"email":     u.Email,
				"country":   u.Country,
				"location":  u.Location,
				"birthdate": fmt.Sprintf("%d-%02d-%02d", u.BirthDate.Year(), u.BirthDate.Month(), u.BirthDate.Day()),
				"gender":    u.Gender,
			},
		}
		ctx.Data["result"] = r
	}
	return T("register.html").Execute(w, map[string]interface{}{
		"ctx": ctx,
	})
}

func Profile(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	if req.FormValue("csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	ctx.Data["title"] = "Profile"
	form := models.UserForm
	form.Fields = form.Fields[2:] // remove passwords
	r := (&form).Load(req)
	ctx.Data["result"] = r
	if len(r.Errors) != 0 {
		return ProfileForm(w, req, ctx)
	}
	u := r.Value.(map[string]interface{})
	gender := u["gender"].(string)
	if gender != "m" && gender != "f" {
		r.Errors["gender"] = errors.New("Please select Male or Female")
	}
	now := time.Now()
	oldest := time.Date(now.Year()-120, 1, 1, 0, 0, 0, 0, time.UTC)
	bDate := u["birthdate"].(time.Time)
	if bDate.Before(oldest) || bDate.After(now) {
		r.Errors["birthdate"] = errors.New("Invalid birth date")
	}
	if r.Err != nil {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem editing profile:", ctx), r.Err.Error()))
		return ProfileForm(w, req, ctx)
	}
	if len(r.Errors) != 0 {
		return ProfileForm(w, req, ctx)
	}
	if err := ctx.C(U).UpdateId(ctx.User.Id, bson.M{"$set": u}); err != nil {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem editing profile:", ctx), err.Error()))
		models.Log(err.Error())
		r.Err = err
	}
	ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Profile updated succesfully!", ctx)))
	return ProfileForm(w, req, ctx)
}

func ResetPasswordForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	// should not be logged in
	if ctx.User != nil {
		ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Already logged in!", ctx)))
		http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
		return nil
	}
	return T("reset.html").Execute(w, map[string]interface{}{
		"ctx": ctx,
	})
}

func ResetPassword(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	// should not be logged in
	if ctx.User != nil {
		ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Already logged in!", ctx)))
		http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
		return nil
	}
	form := models.UserForm
	form.Fields = form.Fields[2:3]
	r := (&form).Load(req)
	ctx.Data["result"] = r
	if r.Err != nil {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem reseting password:", ctx), r.Err.Error()))
		return ResetPasswordForm(w, req, ctx)
	}
	if req.FormValue("csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	if len(r.Errors) != 0 {
		return ResetPasswordForm(w, req, ctx)
	}
	email := r.Values["email"]
	u := &models.User{}
	err := ctx.C(U).Find(bson.M{"email": email}).One(&u)
	if err == nil {
		pt := &models.PasswordToken{
			Uuid:      models.GenUUID(),
			User:      u.Id,
			CreatedOn: time.Now(),
		}
		// set new password to database
		if err := ctx.C(PT).Insert(pt); err != nil {
			ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem reseting password:", ctx), err.Error()))
			models.Log(err.Error())
			r.Err = err
			return ResetPasswordForm(w, req, ctx)
		}
		// sending mail
		body := fmt.Sprintf("Subject: lov3ly.me password reset\r\n\r\nChange password link: http://%s\n\nIf you have NOT requested this, please ignore. Link available for 24 hours.\n\nHave fun,\nlov3ly.me Team", req.Host+reverse("change_token", "uuid", pt.Uuid))
		go func() {
			err := models.SendEmail([]byte(body), email)
			if err != nil {
				models.Log("Error sending mail: ", err.Error())
			}
		}()
		ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Email sent succesfully!", ctx)))
	} else {
		ctx.Session.AddFlash(models.F(models.NOTICE, trans("Email not in our database:", ctx), err.Error()))
	}
	http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
	return nil
}

//change password when loged in
func ChangePasswordForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	return T("change.html").Execute(w, map[string]interface{}{
		"ctx":  ctx,
		"uuid": "",
	})
}

func ChangePassword(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	if ctx.User == nil {
		http.Redirect(w, req, reverse("login"), http.StatusSeeOther)
		return nil
	}
	if req.FormValue("csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	old_pass := req.FormValue("password")
	u := models.User{}
	if err := ctx.C(U).Find(bson.M{"email": ctx.User.Email}).One(&u); err != nil {
		return perform_status(w, req, http.StatusNotFound)
	}
	if len(u.Password) > 0 { // if the account was not created with social auth
		err := bcrypt.CompareHashAndPassword(u.Password, []byte(old_pass))
		if err != nil {
			ctx.Session.AddFlash(models.F(models.ERROR, trans("Invalid Old Password", ctx)))
			return ChangePasswordForm(w, req, ctx)
		}
	}

	new_pass := req.FormValue("password1")
	if len(new_pass) < 5 {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Passwords too short (5 chars or more)", ctx)))
		return ChangePasswordForm(w, req, ctx)
	}
	vfy_pass := req.FormValue("password2")
	if new_pass != vfy_pass {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Password did not match", ctx)))
		return ChangePasswordForm(w, req, ctx)
	}
	hpass, err := models.EncryptPassword(new_pass)
	if err != nil {
		return internal_error(w, req, err.Error())
	}
	// set new password to database
	if err := ctx.C(U).UpdateId(ctx.User.Id, bson.M{"$set": bson.M{"password": hpass}}); err != nil {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem changing password:", ctx), err.Error()))
		models.Log(err.Error())
		return ChangePasswordForm(w, req, ctx)
	}
	ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Password changed succesfully!", ctx)))
	http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
	return nil
}

// Change password from password reset link
func ChangePasswordTokenForm(w http.ResponseWriter, req *http.Request, ctx *models.Context) (err error) {
	// should not be logged in
	if ctx.User != nil {
		ctx.Session.AddFlash(models.F(models.SUCCESS, trans("You can change password normally!", ctx)))
		http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
		return nil
	}
	uuid := req.URL.Query().Get(":uuid")
	if uuid == "" {
		return perform_status(w, req, http.StatusForbidden)
	}
	return T("change.html").Execute(w, map[string]interface{}{
		"ctx":  ctx,
		"uuid": uuid,
	})
}

func ChangePasswordToken(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	go models.RemoveOldPasswordTokens()
	// should not be logged in
	if ctx.User != nil {
		ctx.Session.AddFlash(models.F(models.SUCCESS, trans("You can change password normally!", ctx)))
		http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
		return nil
	}
	uuid := req.URL.Query().Get(":uuid")
	if uuid == "" {
		return perform_status(w, req, http.StatusForbidden)
	}
	pt := &models.PasswordToken{}
	if err := ctx.C(PT).FindId(uuid).One(&pt); err != nil {
		ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Password token expired!", ctx)))
		http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
		return nil
	}

	if req.FormValue("csrf_token") != ctx.Session.Values["csrf_token"] {
		return perform_status(w, req, http.StatusForbidden)
	}
	new_pass := req.FormValue("password1")
	if len(new_pass) < 5 {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Passwords too short (5 chars or more)", ctx)))
		return ChangePasswordForm(w, req, ctx)
	}
	vfy_pass := req.FormValue("password2")
	if new_pass != vfy_pass {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Password did not match", ctx)))
		return ChangePasswordForm(w, req, ctx)
	}
	hpass, err := models.EncryptPassword(new_pass)
	if err != nil {
		return internal_error(w, req, err.Error())
	}
	// set new password to database
	if err := ctx.C(U).UpdateId(pt.User, bson.M{"$set": bson.M{"password": hpass}}); err != nil {
		ctx.Session.AddFlash(models.F(models.ERROR, trans("Problem changing password:", ctx), err.Error()))
		models.Log(err.Error())
		return ChangePasswordForm(w, req, ctx)
	}
	ctx.Session.Values["user"] = pt.User
	// delete password token
	if err := ctx.C(PT).RemoveId(pt.Uuid); err != nil {
		models.Log("error deleting password token: ", err.Error())
	}
	ctx.Session.AddFlash(models.F(models.SUCCESS, trans("Password changed succesfully!", ctx)))
	http.Redirect(w, req, reverse("index"), http.StatusSeeOther)
	return nil
}

func Location(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	results := typeAhead("location", req.FormValue("query"), ctx)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, results)
	return nil
}

func Country(w http.ResponseWriter, req *http.Request, ctx *models.Context) error {
	results := typeAhead("country", req.FormValue("query"), ctx)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, results)
	return nil
}

func typeAhead(field, q string, ctx *models.Context) string {
	var users []map[string]string
	query := bson.M{field: bson.RegEx{"^" + q, "i"}}
	ctx.C(U).Find(query).Select(bson.M{field: 1, "_id": 0}).All(&users)
	distinct := make(map[string]bool)
	for _, u := range users {
		if _, ok := distinct[u[field]]; !ok {
			distinct[u[field]] = true
		}
	}
	var result string
	for k, _ := range distinct {
		result += `"` + k + `",`
	}
	result = strings.TrimRight(result, ",")
	return `{"options":[` + result + `]}`
}
