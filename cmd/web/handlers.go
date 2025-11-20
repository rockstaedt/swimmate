package main

import (
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/rockstaedt/swimmate/internal/models"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	itemsPerPage  = 20
	swimsTemplate = "swims.tmpl"
)

type swimsPageData struct {
	Swims     []*swimRowTemplateData
	Offset    int
	Sort      string
	Direction string
	LoadMore  *loadMoreData
}

type swimRowTemplateData struct {
	Swim      *models.Swim
	Sort      string
	Direction string
}

type editSwimPageData struct {
	Swim      *models.Swim
	Sort      string
	Direction string
}

type loadMoreData struct {
	NextOffset int
	Sort       string
	Direction  string
}

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

func (app *application) editSwim(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	swimID, err := strconv.Atoi(params.ByName("id"))
	if err != nil || swimID <= 0 {
		app.notFound(w)
		return
	}

	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	swim, err := app.swims.GetByID(userId, swimID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	sort, direction := parseSwimSort(r)
	data := editSwimPageData{
		Swim:      swim,
		Sort:      sort,
		Direction: direction,
	}

	app.render(w, r, http.StatusOK, "swim-edit.tmpl", app.newTemplateData(r, data))
}

func (app *application) swimsList(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	sort, direction := parseSwimSort(r)

	swims, err := app.swims.GetPaginated(userId, itemsPerPage, 0, sort, direction)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	swimRows := make([]*swimRowTemplateData, len(swims))
	for i, swim := range swims {
		swimRows[i] = newSwimRowTemplateData(swim, sort, direction)
	}

	data := swimsPageData{
		Swims:     swimRows,
		Offset:    0,
		Sort:      sort,
		Direction: direction,
		LoadMore:  newLoadMoreData(len(swims) == itemsPerPage, itemsPerPage, sort, direction),
	}

	app.render(w, r, http.StatusOK, swimsTemplate, app.newTemplateData(r, data))
}

func (app *application) swimsMore(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	sort, direction := parseSwimSort(r)

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		offset = 0
	}

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Return partial HTML for HTMX
		swims, err := app.swims.GetPaginated(userId, itemsPerPage, offset, sort, direction)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		for _, swim := range swims {
			app.renderPartial(w, r, swimsTemplate, "swim-row", newSwimRowTemplateData(swim, sort, direction))
		}

		// Add the new button row or end
		if loadMore := newLoadMoreData(len(swims) == itemsPerPage, offset+itemsPerPage, sort, direction); loadMore != nil {
			app.renderPartial(w, r, swimsTemplate, "load-more-button", loadMore)
		}
		return
	}

	// For direct browser requests, show full page with all swims up to offset + 20
	limit := offset + itemsPerPage
	swims, err := app.swims.GetPaginated(userId, limit, 0, sort, direction)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	swimRows := make([]*swimRowTemplateData, len(swims))
	for i, swim := range swims {
		swimRows[i] = newSwimRowTemplateData(swim, sort, direction)
	}

	data := swimsPageData{
		Swims:     swimRows,
		Offset:    offset,
		Sort:      sort,
		Direction: direction,
		LoadMore:  newLoadMoreData(len(swims) == limit, limit, sort, direction),
	}

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

func (app *application) updateSwim(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	swimID, err := strconv.Atoi(params.ByName("id"))
	if err != nil || swimID <= 0 {
		app.notFound(w)
		return
	}

	err = r.ParseForm()
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

	sort := normalizeSwimSortValue(r.PostForm.Get("sort"))
	direction := normalizeSortDirectionValue(r.PostForm.Get("direction"))

	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	err = app.swims.Update(swimID, userId, date, distanceM, assessment)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flashText", "Successfully updated!")

	values := url.Values{}
	values.Set("sort", sort)
	values.Set("direction", direction)

	http.Redirect(w, r, "/swims?"+values.Encode(), http.StatusSeeOther)
}

func parseSwimSort(r *http.Request) (string, string) {
	sort := normalizeSwimSortValue(r.URL.Query().Get("sort"))
	direction := normalizeSortDirectionValue(r.URL.Query().Get("direction"))
	return sort, direction
}

func newLoadMoreData(hasMore bool, nextOffset int, sort, direction string) *loadMoreData {
	if !hasMore {
		return nil
	}

	return &loadMoreData{
		NextOffset: nextOffset,
		Sort:       sort,
		Direction:  direction,
	}
}

func newSwimRowTemplateData(swim *models.Swim, sort, direction string) *swimRowTemplateData {
	return &swimRowTemplateData{
		Swim:      swim,
		Sort:      sort,
		Direction: direction,
	}
}

func normalizeSwimSortValue(sort string) string {
	sort = strings.ToLower(sort)
	if sort != models.SwimSortDate && sort != models.SwimSortDistance && sort != models.SwimSortAssessment {
		return models.SwimSortDate
	}
	return sort
}

func normalizeSortDirectionValue(direction string) string {
	direction = strings.ToLower(direction)
	if direction != models.SortDirectionAsc && direction != models.SortDirectionDesc {
		return models.SortDirectionDesc
	}
	return direction
}
