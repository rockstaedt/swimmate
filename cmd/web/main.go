package main

import (
	"log/slog"
	"net/http"
	"os"
)

var version string

type application struct {
	logger  *slog.Logger
	version string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	app := &application{
		logger:  logger,
		version: version,
	}

	port := ":8998"
	logger.Info("starting server", "port", port)

	err := http.ListenAndServe(port, app.routes())
	logger.Error(err.Error())
	os.Exit(1)
}
