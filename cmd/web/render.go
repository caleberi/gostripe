package main

import (
	"embed"
	"fmt"
	"net/http"
	"strings"
	"text/template"
)

type templateData struct {
	StringMap       map[string]string
	IntMap          map[string]int
	FloatMap        map[string]float64
	Data            map[string]interface{}
	Flash           string
	Warning         string
	CSRFToken       string
	IsAuthenticated bool
	Error           string
	CSSVersion      string
	AppVersion      string
	API             string
}

var functions = template.FuncMap{}

//go:embed templates
var templateFS embed.FS

func (app *application) addDefaultData(td *templateData, r *http.Request) *templateData {
	td.API = app.config.api
	return td
}

func (app *application) renderTemplate(w http.ResponseWriter, r *http.Request, page string, td *templateData, partials ...string) error {
	var t *template.Template
	var err error
	templateToRender := fmt.Sprintf("templates/%s.gohtml", page)
	_, exist := app.templateCache[templateToRender]
	if app.config.env == "production" && exist {
		t = app.templateCache[templateToRender]
	} else {
		t, err = app.parseTemplate(partials, page, templateToRender)
		if err != nil {
			app.errorLog.Println(err)
			return err
		}
	}

	if td == nil {
		td = &templateData{}
	}

	td = app.addDefaultData(td, r)
	err = t.Execute(w, td)
	if err != nil {
		app.errorLog.Println(err)
		return err
	}
	return nil
}

func (app *application) parseTemplate(partials []string, page, templateToRender string) (*template.Template, error) {
	var t *template.Template
	var err error

	// building partials
	if len(partials) > 0 {
		for i, p := range partials {
			partials[i] = fmt.Sprintf("templates/partials/%s.gohtml", p)
		}
	}
	path := fmt.Sprintf("%s.gohtml", page)
	if len(partials) > 0 {
		t, err = template.New(path).Funcs(functions).ParseFS(
			templateFS,
			"templates/base.gohtml",
			strings.Join(partials, ","),
			templateToRender,
		)
	} else {
		t, err = template.New(path).Funcs(functions).ParseFS(
			templateFS,
			"templates/base.gohtml",
			templateToRender,
		)
	}

	if err != nil {
		app.errorLog.Println(err)
		return nil, err
	}
	app.templateCache[templateToRender] = t
	return t, nil
}
