package main

import (
	"app/controllers"
	"app/models"
	"github.com/dchest/captcha"
	"log"
	"net/http"
	"os"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	router := models.Router
	// static
	router.Add("GET", "/static/", http.FileServer(http.Dir(models.BASE_DIR))).Name("static")
	router.Add("GET", "/uploads/", http.FileServer(http.Dir(models.DATA_DIR))).Name("uploads")
	// auth
	router.Add("GET", "/login/", controllers.Handler(controllers.LoginForm)).Name("login")
	router.Add("POST", "/login/", controllers.Handler(controllers.Login))
	router.Add("GET", "/fblogin", controllers.Handler(controllers.FbLogin)).Name("fblogin")
	router.Add("GET", "/gllogin", controllers.Handler(controllers.GlLogin)).Name("gllogin")

	router.Add("GET", "/logout/{csrf_token:[0-9a-z]+}", controllers.Handler(controllers.Logout)).Name("logout")

	router.Add("GET", "/register/", controllers.Handler(controllers.RegisterForm)).Name("register")
	router.Add("POST", "/register/", controllers.Handler(controllers.Register))
	router.Add("GET", "/captcha/", captcha.Server(captcha.StdWidth, captcha.StdHeight))

	router.Add("GET", "/profile/", controllers.Handler(controllers.ProfileForm)).Name("profile")
	router.Add("POST", "/profile/", controllers.Handler(controllers.Profile))

	router.Add("GET", "/reset/", controllers.Handler(controllers.ResetPasswordForm)).Name("reset")
	router.Add("POST", "/reset/", controllers.Handler(controllers.ResetPassword))

	router.Add("GET", "/change/", controllers.Handler(controllers.ChangePasswordForm)).Name("change")
	router.Add("POST", "/change/", controllers.Handler(controllers.ChangePassword))

	router.Add("GET", "/changetoken/{uuid:[0-9a-z]+}", controllers.Handler(controllers.ChangePasswordTokenForm)).Name("change_token")
	router.Add("POST", "/changetoken/{uuid:[0-9a-z]+}", controllers.Handler(controllers.ChangePasswordToken))

	// autocomplete stuff
	router.Add("GET", "/location", controllers.Handler(controllers.Location)).Name("location")
	router.Add("GET", "/country", controllers.Handler(controllers.Country)).Name("country")
	router.Add("GET", "/search", controllers.Handler(controllers.Search)).Name("search")

	// photos
	router.Add("GET", "/upload/{id:[0-9a-z]*}", controllers.Handler(controllers.UploadForm)).Name("upload")
	router.Add("POST", "/upload/{id:[0-9a-z]*}", controllers.Handler(controllers.Upload))

	router.Add("GET", "/delete/{id:[0-9a-z]+}/{csrf_token:[0-9a-z]+}", controllers.Handler(controllers.Delete)).Name("delete")

	router.Add("GET", "/photos/{id:[0-9a-z]+}/{photo:[0-9a-z]*}", controllers.Handler(controllers.Photos)).Name("photos")
	router.Add("GET", "/photo/{id:[0-9a-z]+}/{kind:p|c}/{photo:[0-9a-z]*}", controllers.Handler(controllers.ExternalPhoto)).Name("external_photo")
	router.Add("GET", "/fake/{photo:[0-9a-z]+}/{csrf_token:[0-9a-z]+}", controllers.Handler(controllers.Fake)).Name("fake")
	router.Add("GET", "/abuse/{photo:[0-9a-z]+}/{csrf_token:[0-9a-z]+}", controllers.Handler(controllers.Abuse)).Name("abuse")
	router.Add("GET", "/avatar/{photo:[0-9a-z]+}/{csrf_token:[0-9a-z]+}", controllers.Handler(controllers.SetAvatar)).Name("avatar")
	router.Add("GET", "/top/{page:[0-9]+}", controllers.Handler(controllers.TopVoted)).Name("top")
	router.Add("GET", "/latest/{page:[0-9]+}", controllers.Handler(controllers.Latest)).Name("latest")
	router.Add("GET", "/random/{page:[0-9]+}", controllers.Handler(controllers.Random)).Name("random")
	router.Add("GET", "/empty", controllers.Handler(controllers.Empty)).Name("empty")

	// admin
	router.Add("GET", "/lov3lymin1", controllers.Handler(controllers.Admin)).Name("admin")
	router.Add("GET", "/lov3lymin2/delphoto/{id:[0-9a-z]+}", controllers.Handler(controllers.DelPhoto)).Name("del_photo")
	router.Add("GET", "/lov3lymin3/deluser/{id:[0-9a-z]+}", controllers.Handler(controllers.DelUser)).Name("del_user")

	// comments
	router.Add("GET", "/comment/{kind:p|c}/{id:[0-9a-z]+}", controllers.Handler(controllers.CommentForm)).Name("comments")
	router.Add("POST", "/comment/{kind:p|c}/{id:[0-9a-z]+}", controllers.Handler(controllers.Comment))

	// votes
	router.Add("GET", "/vote/{photo:[0-9a-z]+}/{csrf_token:[0-9a-z]+}/{contest:[0-9a-z]*}", controllers.Handler(controllers.Vote)).Name("vote")
	router.Add("GET", "/getvote/{photo:[0-9a-z]+}/{contest:[0-9a-z]*}", controllers.Handler(controllers.GetVote)).Name("get_vote")
	router.Add("POST", "/filter/", controllers.Handler(controllers.Filter)).Name("filter")
	router.Add("GET", "/getphotovotes/{id:[0-9a-z]+}", controllers.Handler(controllers.GetPhotoVotes)).Name("get_photo_votes")

	// contests
	router.Add("GET", "/contests/{id:[0-9a-z]*}", controllers.Handler(controllers.ContestForm)).Name("contest")
	router.Add("POST", "/contests/{id:[0-9a-z]*}", controllers.Handler(controllers.Contest))
	router.Add("GET", "/deletecontest/{id:[0-9a-z]+}/{csrf_token:[0-9a-z]+}", controllers.Handler(controllers.DeleteContest)).Name("delete_contest")
	router.Add("GET", "/publishcontest/{id:[0-9a-z]+}/{csrf_token:[0-9a-z]+}", controllers.Handler(controllers.PublishContest)).Name("publish_contest")

	router.Add("GET", "/registercontest/{id:[0-9a-z]+}", controllers.Handler(controllers.RegisterContestForm)).Name("register_contest")
	router.Add("POST", "/registercontest/{id:[0-9a-z]+}", controllers.Handler(controllers.RegisterContest))

	router.Add("GET", "/pendingaprovals/{id:[0-9a-z]+}", controllers.Handler(controllers.PendingApprovals)).Name("pending_approvals")
	router.Add("GET", "/conteststatus/{id:[0-9a-z]+}", controllers.Handler(controllers.ContestStatus)).Name("contest_status")
	router.Add("GET", "/approvecontest/{id:[0-9a-z]+}/{user:[0-9a-z]+}/{csrf_token:[0-9a-z]+}/{res:y|n}", controllers.Handler(controllers.ApproveContest)).Name("approve_contest")
	router.Add("GET", "/viewcontest/{id:[0-9a-z]+}/{photo:[0-9a-z]*}", controllers.Handler(controllers.ViewContest)).Name("view_contest")
	router.Add("GET", "/contestlist/{list:adm|vot|fin|pop}", controllers.Handler(controllers.ContestList)).Name("contest_list")

	// rankings
	router.Add("GET", "/rankings", controllers.Handler(controllers.Rankings)).Name("rankings")

	// messageds
	router.Add("GET", "/sendmessage/{to:[0-9a-z]+}", controllers.Handler(controllers.SendMessageForm)).Name("send_message")
	router.Add("POST", "/sendmessage/{to:[0-9a-z]+}", controllers.Handler(controllers.SendMessage))
	router.Add("GET", "/messages", controllers.Handler(controllers.Messages)).Name("messages")
	router.Add("GET", "/delmessage/{id:[0-9a-z]+}", controllers.Handler(controllers.DelMessage)).Name("delete_message")

	//google web masters
	router.Add("GET", "/google4b899b9e0462f0cd.html", http.HandlerFunc(controllers.GoogleSiteVerification)).Name("google1")
	router.Add("GET", "/robots.txt", http.HandlerFunc(controllers.Robots)).Name("robots")

	//contact
	router.Add("GET", "/contact/", controllers.Handler(controllers.ContactForm)).Name("contact")
	router.Add("POST", "/contact/", controllers.Handler(controllers.Contact))

	// static
	router.Add("GET", "/page/{p:[a-z]+}", controllers.Handler(controllers.Static)).Name("page")

	// language
	router.Add("GET", "/language/{lang:[a-z]{2}}", controllers.Handler(controllers.SetLanguage)).Name("language")

	// index
	router.Add("GET", "/", controllers.Handler(controllers.Index)).Name("index")

	log.Print("The server is listening...")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := http.ListenAndServe(os.Getenv("HOST")+":"+port, router); err != nil {
		log.Print("cmo server: ", err)
	}
}
