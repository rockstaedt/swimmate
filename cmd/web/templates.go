package main

import (
	"html/template"
	"path/filepath"
)

type templateData struct {
	Version string
}

func (app *application) newTemplateData() templateData {
	versionTxt := "_dev"
	if len(app.version) != 0 {
		versionTxt = app.version
	}

	return templateData{Version: versionTxt}
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := filepath.Glob("./ui/html/pages/*.tmpl")
	if err != nil {
		return nil, err
	}
	for _, page := range pages {
		name := filepath.Base(page)
		files := []string{
			"./ui/html/base.tmpl",
			page,
		}
		ts, err := template.ParseFiles(files...)
		if err != nil {
			return nil, err
		}
		cache[name] = ts
	}
	return cache, nil
}
