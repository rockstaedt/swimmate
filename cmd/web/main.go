package main

import (
	"log/slog"
	"net/http"
	"os"
)

type application struct {
	logger *slog.Logger
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	app := &application{
		logger: logger,
	}

	port := ":8998"
	logger.Info("starting server", "port", port)

	err := http.ListenAndServe(port, app.routes())
	logger.Error(err.Error())
	os.Exit(1)
}
