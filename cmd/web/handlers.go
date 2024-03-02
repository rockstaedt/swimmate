package main

import (
	"github.com/rockstaedt/swimmate/internal/models"
	"net/http"
	"strconv"
	"time"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		app.notFound(w)
	}

	app.render(w, r, http.StatusOK, "home.tmpl", app.newTemplateData(app.swims.Summarize()))
}

func (app *application) yearlyFigures(w http.ResponseWriter, r *http.Request) {
	year := time.Now().Year()
	if r.URL.Query().Has("year") {
		year, _ = strconv.Atoi(r.URL.Query().Get("year"))
	}

	summary := app.swims.Summarize()
	data := struct {
		Summary *models.SwimSummary
		Year    int
	}{summary, year}

	app.render(w, r, http.StatusOK, "yearly-figures.tmpl", app.newTemplateData(data))
}

func (app *application) createSwim(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "swim-create.tmpl", app.newTemplateData(nil))
}

func (app *application) storeSwim(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.logger.Error(err.Error())
		app.clientError(w, http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", r.PostForm.Get("date"))
	if err != nil {
		app.logger.Error(err.Error())
		app.clientError(w, http.StatusBadRequest)
		return
	}

	distanceM, err := strconv.Atoi(r.PostForm.Get("distance_m"))
	if err != nil {
		app.logger.Error(err.Error())
		app.clientError(w, http.StatusBadRequest)
		return
	}

	assessment, err := strconv.Atoi(r.PostForm.Get("assessment"))
	if err != nil {
		app.logger.Error(err.Error())
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = app.swims.Insert(date, distanceM, assessment)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
