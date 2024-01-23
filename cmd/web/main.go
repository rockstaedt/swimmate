package main

import (
	"github.com/rockstaedt/swimmate/ui"
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

	mux := http.NewServeMux()

	fileServer := http.FileServer(http.FS(ui.Files))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("/", app.home)

	port := ":8998"
	logger.Info("starting server", "port", port)

	err := http.ListenAndServe(port, mux)
	logger.Error(err.Error())
	os.Exit(1)
}
