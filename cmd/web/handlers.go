package main

import (
	"errors"
	"github.com/rockstaedt/swimmate/internal/models"
	"net/http"
	"strconv"
	"time"
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
	
	swims, err := app.swims.GetPaginated(userId, 20, 0)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := struct {
		Swims  []*models.Swim
		Offset int
		Limit  int
	}{swims, 0, 20}

	app.render(w, r, http.StatusOK, "swims.tmpl", app.newTemplateData(r, data))
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
		swims, err := app.swims.GetPaginated(userId, 20, offset)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		for _, swim := range swims {
			w.Write([]byte(`<tr>`))
			w.Write([]byte(`<td>` + swim.Date.Format("2006-01-02") + `</td>`))
			w.Write([]byte(`<td>` + numberFormat(swim.DistanceM) + ` m</td>`))
			w.Write([]byte(`<td>`))
			
			for i := 0; i < swim.Assessment+1; i++ {
				w.Write([]byte(`<i class="fas fa-star"></i>`))
			}
			for i := swim.Assessment+1; i < 3; i++ {
				w.Write([]byte(`<i class="far fa-star"></i>`))
			}
			
			w.Write([]byte(`</td>`))
			w.Write([]byte(`</tr>`))
		}

		// Add the new button row or end
		if len(swims) == 20 {
			newOffset := offset + 20
			w.Write([]byte(`<tr id="load-more-row"><td colspan="3" style="text-align: center; padding: 2rem;"><button hx-get="/swims/more?offset=` + strconv.Itoa(newOffset) + `" hx-target="#load-more-row" hx-swap="outerHTML">Load More</button></td></tr>`))
		}
		return
	}

	// For direct browser requests, show full page with all swims up to offset + 20
	totalSwims := offset + 20
	swims, err := app.swims.GetPaginated(userId, totalSwims, 0)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := struct {
		Swims  []*models.Swim
		Offset int
		Limit  int
	}{swims, offset, 20}

	app.render(w, r, http.StatusOK, "swims.tmpl", app.newTemplateData(r, data))
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
