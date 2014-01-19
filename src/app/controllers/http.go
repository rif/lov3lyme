package controllers

import (
	"app/models"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"thegoods.biz/httpbuf"
)

const (
	P              = "photos"
	V              = "votes"
	U              = "users"
	C              = "contests"
	M              = "messages"
	PT             = "passwordtokens"
	ITEMS_PER_PAGE = 20
)

var (
	langRE = regexp.MustCompile(`([a-z]{2}(?:\-[a-z]{2})?(?:\-[a-z]{2})?)(?:[,;]|$)`)
)

// Package httpgzip provides a http handler wrapper to transparently
// add gzip compression. It will sniff the content type based on the
// uncompressed data if necessary.
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	sniffDone bool
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.sniffDone {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(b))
		}
		w.sniffDone = true
	}
	return w.Writer.Write(b)
}

type Handler func(http.ResponseWriter, *http.Request, *models.Context) error

func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Encoding", "gzip")
	gzz := gzip.NewWriter(w)
	defer gzz.Close()
	gz := gzipResponseWriter{Writer: gzz, ResponseWriter: w}
	//create the context
	ctx, err := models.NewContext(req)
	if err != nil {
		internal_error(gz, req, "new context err: "+err.Error())
		return
	}
	defer ctx.Close()

	//run the handler and grab the error, and report it
	buf := new(httpbuf.Buffer)
	err = h(buf, req, ctx)
	if err != nil {
		internal_error(gz, req, "buffer err: "+err.Error())
		return
	}

	//save the session
	if err = ctx.Session.Save(req, buf); err != nil {
		internal_error(gz, req, "session save err: "+err.Error())
		return
	}

	// set content type and length
	//buf.Header().Set("Content-Type", "text/html")
	//buf.Header().Set("Content-Length", string(buf.Len()))

	//apply the buffered response to the writer
	buf.Apply(gz)
}

//perform_status runs the passed in status on the request and calls the appropriate block
func perform_status(w http.ResponseWriter, req *http.Request, status int) error {
	//w.WriteHeader(status)
	return T(fmt.Sprintf("%d.html", status)).Execute(w, nil)
}

func reverse(name string, things ...interface{}) string {
	//convert the things to strings
	strs := make([]string, len(things))
	for i, th := range things {
		strs[i] = fmt.Sprint(th)
	}
	//grab the route
	u, err := models.Router.GetRoute(name).URL(strs...)
	if err != nil {
		models.Logf("reverse (%s %v): %s", name, things, err.Error())
		return "#"
	}
	return u.Path
}

//internal_error is what is called when theres an error processing something
func internal_error(w http.ResponseWriter, req *http.Request, err string) error {
	models.Log("!!!!error serving request page: ", err)
	return perform_status(w, req, http.StatusInternalServerError)
}

// extract language string from browser language selection
func detectLanguage(browser []string, ctx *models.Context) {
	if len(browser) > 0 {
		for _, lang := range langRE.FindAllStringSubmatch(browser[0], -1) {
			if len(lang) != 2 {
				continue
			}
			l := lang[1]
			if strings.Contains(l, "-") {
				l = strings.Split(l, "-")[0]
			}
			// if it is english, the default use it and return
			if l == "en" {
				ctx.Session.Values["lang"] = "en"
				return
			}
			// try to find the language in translations
			if _, ok := models.Translations[l]; ok {
				ctx.Session.Values["lang"] = l
				return
			}
		}
	}
}
