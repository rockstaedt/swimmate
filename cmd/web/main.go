package main

import (
	"html/template"
	"log/slog"
	"net/http"
	"os"
)

var version string

type application struct {
	logger        *slog.Logger
	templateCache map[string]*template.Template
	version       string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	app := &application{
		logger:        logger,
		templateCache: templateCache,
		version:       version,
	}

	port := ":8998"
	logger.Info("starting server", "port", port)

	err = http.ListenAndServe(port, app.routes())
	logger.Error(err.Error())
	os.Exit(1)
}
