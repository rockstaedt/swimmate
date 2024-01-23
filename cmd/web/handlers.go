package main

import (
	"github.com/rockstaedt/swimmate/ui"
	"html/template"
	"net/http"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		app.notFound(w)
	}

	files := []string{
		"html/base.tmpl",
		"html/pages/home.tmpl",
	}

	ts, err := template.ParseFS(ui.Files, files...)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	versionTxt := app.version
	if len(versionTxt) == 0 {
		versionTxt = "_dev"
	}
	data := templateData{Version: versionTxt}
	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}
