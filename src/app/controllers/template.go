package controllers

import (
	"app/models"
	"fmt"
	"github.com/dustin/go-humanize"
	"html/template"
	"path"
	"path/filepath"
	"reflect"
	"sync"
)

var (
	cachedTemplates = map[string]*template.Template{}
	cachedMutex     sync.Mutex

	funcs = template.FuncMap{
		"reverse":    reverse,
		"eq":         eq,
		"neq":        neq,
		"to_p":       to_p,
		"image":      models.ImageUrl,
		"human_time": humanize.Time,
		"trunc":      truncateString,
		"trans":      trans,
	}
)

func T(name string) *template.Template {
	cachedMutex.Lock()
	defer cachedMutex.Unlock()

	if t, ok := cachedTemplates[name]; ok {
		return t
	}

	t := template.New("base.html").Funcs(funcs)

	t = template.Must(t.ParseFiles(
		path.Join(models.BASE_DIR, "src/app/tmpl/base.html"),
		filepath.Join(models.BASE_DIR, "src/app/tmpl", name),
	))
	cachedTemplates[name] = t

	return t
}

func AJAX(name string) *template.Template {
	cachedMutex.Lock()
	defer cachedMutex.Unlock()

	if t, ok := cachedTemplates[name]; ok {
		return t
	}

	t := template.New(name).Funcs(funcs)
	t = template.Must(t.ParseFiles(filepath.Join(models.BASE_DIR, "src/app/tmpl", name)))
	cachedTemplates[name] = t

	return t
}

// eq reports whether the first argument is equal to
// any of the remaining arguments.
func eq(args ...interface{}) bool {
	if len(args) == 0 {
		return false
	}
	x := args[0]
	switch x := x.(type) {
	case string, int, int64, byte, float32, float64:
		for _, y := range args[1:] {
			if x == y {
				return true
			}
		}
		return false
	}

	for _, y := range args[1:] {
		if reflect.DeepEqual(x, y) {
			return true
		}
	}
	return false
}

func neq(args ...interface{}) bool {
	return !eq(args...)
}

func to_p(i interface{}) template.HTML {
	f := i.(*models.Flash)
	return template.HTML(fmt.Sprintf(`<p class="%s">%s</p>`, f.Type, f.Message))
}

func SafeHtml(text string) template.HTML {
	return template.HTML(text)
}

func truncateString(s string, size int) string {
	if len(s) > size+3 {
		return s[:size] + "..."
	}
	return s
}

func trans(str string, ctx *models.Context) string {
	if ctx == nil || ctx.Session.Values["lang"] == nil {
		return str
	}
	if trans, ok := models.Translations[ctx.Session.Values["lang"].(string)]; ok {
		if newStr, ok := trans[str]; ok {
			return newStr
		}
	}
	return str
}
