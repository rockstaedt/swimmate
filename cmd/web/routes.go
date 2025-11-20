package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/rockstaedt/swimmate/ui"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	fileServer := http.FileServer(http.FS(ui.Files))
	router.Handler(http.MethodGet, "/static/*filepath", fileServer)

	dynamic := alice.New(app.sessionManager.LoadAndSave)

	router.Handler(http.MethodGet, "/login", dynamic.ThenFunc(app.login))
	router.Handler(http.MethodPost, "/authenticate", dynamic.ThenFunc(app.authenticate))
	router.Handler(http.MethodPost, "/logout", dynamic.ThenFunc(app.logout))
	router.Handler(http.MethodGet, "/about", dynamic.ThenFunc(app.about))

	protected := dynamic.Append(app.requireAuthentication)

	router.Handler(http.MethodGet, "/", protected.ThenFunc(app.home))
	router.Handler(http.MethodGet, "/swims", protected.ThenFunc(app.swimsList))
	router.Handler(http.MethodGet, "/swims/more", protected.ThenFunc(app.swimsMore))
	router.Handler(http.MethodGet, "/yearly-figures", protected.ThenFunc(app.yearlyFigures))
	router.Handler(http.MethodGet, "/swim", protected.ThenFunc(app.createSwim))
	router.Handler(http.MethodPost, "/swim", protected.ThenFunc(app.storeSwim))
	router.Handler(http.MethodGet, "/swims/edit/:id", protected.ThenFunc(app.editSwim))
	router.Handler(http.MethodPost, "/swims/edit/:id", protected.ThenFunc(app.updateSwim))
	router.Handler(http.MethodPut, "/swims/edit/:id", protected.ThenFunc(app.updateSwim))

	standard := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	return standard.Then(router)
}
