package main

import (
	"github.com/rockstaedt/swimmate/internal/models"
	"github.com/rockstaedt/swimmate/ui"
	"html/template"
	"io/fs"
	"path/filepath"
)

type templateData struct {
	Version string
	Swim    *models.Swim
}

func (app *application) newTemplateData(swim *models.Swim) templateData {
	versionTxt := "development"
	if len(app.version) != 0 {
		versionTxt = app.version
	}

	return templateData{Version: versionTxt, Swim: swim}
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := fs.Glob(ui.Files, "html/pages/*.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		patterns := []string{"html/base.tmpl", page}

		ts, errPars := template.New(name).ParseFS(ui.Files, patterns...)
		if errPars != nil {
			return nil, errPars
		}

		cache[name] = ts
	}
	return cache, nil
}
