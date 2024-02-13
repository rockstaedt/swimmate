package main

import (
	"context"
	"database/sql"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
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

	db, err := openDB()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	logger.Info("database connection pool established")

	app := &application{
		logger:        logger,
		templateCache: templateCache,
		version:       version,
	}

	port := ":8998"
	srv := &http.Server{
		Addr:     port,
		Handler:  app.routes(),
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	logger.Info("starting server", "port", port)

	err = srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}

func openDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", os.Getenv("DB_DSN"))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}
	// Return the sql.DB connection pool.
	return db, nil
}
