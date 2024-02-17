package main

import (
	"errors"
	"github.com/rockstaedt/swimmate/internal/models"
	"net/http"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		app.notFound(w)
	}

	swims, err := app.swims.GetAll()
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.render(w, r, http.StatusOK, "home.tmpl", app.newTemplateData(app.swims.Summarize(swims)))
}

func (app *application) createSwim(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "swim-create.tmpl", app.newTemplateData(nil))
}
