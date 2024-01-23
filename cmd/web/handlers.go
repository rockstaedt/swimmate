package main

import (
	"net/http"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		app.notFound(w)
	}

	app.render(w, r, http.StatusOK, "home.tmpl", app.newTemplateData())
}
