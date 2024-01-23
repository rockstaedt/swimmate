package main

import (
	"net/http"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		app.notFound(w)
	}

	versionTxt := app.version
	if len(versionTxt) == 0 {
		versionTxt = "_dev"
	}
	data := templateData{Version: versionTxt}

	app.render(w, r, http.StatusOK, "home.tmpl", data)
}
