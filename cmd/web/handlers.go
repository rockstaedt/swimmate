package main

import (
	"errors"
	"github.com/rockstaedt/swimmate/internal/models"
	"net/http"
	"strconv"
	"time"
)

const (
	itemsPerPage = 20
	swimsTemplate = "swims.tmpl"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		app.notFound(w)
	}

	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")

	app.render(w, r, http.StatusOK, "home.tmpl", app.newTemplateData(r, app.swims.Summarize(userId)))
}

func (app *application) login(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "login.tmpl", app.newTemplateData(r, nil))
}

func (app *application) authenticate(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.logger.Error(err.Error())
		app.clientError(w, http.StatusBadRequest)
		return
	}

	id, err := app.users.Authenticate(r.PostForm.Get("username"), r.PostForm.Get("password"))
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			app.sessionManager.Put(r.Context(), "flashText", "Invalid credentials.")
			app.sessionManager.Put(r.Context(), "flashType", "flash-error")
			app.render(w, r, http.StatusOK, "login.tmpl", app.newTemplateData(r, nil))
			return
		}

		app.serverError(w, r, err)
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)
	app.sessionManager.Put(r.Context(), "flashText", "Successfully logged in.")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) logout(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")
	app.sessionManager.Put(r.Context(), "flashText", "Successfully logged out.")

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *application) yearlyFigures(w http.ResponseWriter, r *http.Request) {
	year := time.Now().Year()
	if r.URL.Query().Has("year") {
		year, _ = strconv.Atoi(r.URL.Query().Get("year"))
	}

	summary := app.swims.Summarize(app.sessionManager.GetInt(r.Context(), "authenticatedUserID"))
	data := struct {
		Summary *models.SwimSummary
		Year    int
	}{summary, year}

	app.render(w, r, http.StatusOK, "yearly-figures.tmpl", app.newTemplateData(r, data))
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "about.tmpl", app.newTemplateData(r, nil))
}

func (app *application) createSwim(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "swim-create.tmpl", app.newTemplateData(r, nil))
}

func (app *application) swimsList(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	
	swims, err := app.swims.GetPaginated(userId, itemsPerPage, 0)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := struct {
		Swims  []*models.Swim
		Offset int
		Limit  int
	}{swims, 0, itemsPerPage}

	app.render(w, r, http.StatusOK, swimsTemplate, app.newTemplateData(r, data))
}

func (app *application) swimsMore(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		offset = 0
	}

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Return partial HTML for HTMX
		swims, err := app.swims.GetPaginated(userId, itemsPerPage, offset)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		for _, swim := range swims {
			app.renderPartial(w, r, swimsTemplate, "swim-row", swim)
		}

		// Add the new button row or end
		if len(swims) == itemsPerPage {
			nextOffset := offset + itemsPerPage
			app.renderPartial(w, r, swimsTemplate, "load-more-button", nextOffset)
		}
		return
	}

	// For direct browser requests, show full page with all swims up to offset + 20
	limit := offset + itemsPerPage
	swims, err := app.swims.GetPaginated(userId, limit, 0)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := struct {
		Swims  []*models.Swim
		Offset int
		Limit  int
	}{swims, offset, itemsPerPage}

	app.render(w, r, http.StatusOK, swimsTemplate, app.newTemplateData(r, data))
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

	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	err = app.swims.Insert(date, distanceM, assessment, userId)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flashText", "Successfully created!")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
