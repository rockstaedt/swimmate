package main

import (
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/rockstaedt/swimmate/ui"
	"net/http"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	fileServer := http.FileServer(http.FS(ui.Files))
	router.Handler(http.MethodGet, "/static/*filepath", fileServer)

	router.HandlerFunc(http.MethodGet, "/", app.home)
	router.HandlerFunc(http.MethodGet, "/yearly-figures", app.yearlyFigures)
	router.HandlerFunc(http.MethodGet, "/about", app.about)
	router.HandlerFunc(http.MethodGet, "/swim", app.createSwim)
	router.HandlerFunc(http.MethodPost, "/swim", app.storeSwim)

	standard := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	return standard.Then(router)
}
