package main

import (
	"github.com/rockstaedt/swimmate/ui"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
)

type templateData struct {
	Version         string
	Data            interface{}
	Flash           *Flash
	IsAuthenticated bool
	CurrentDate     string
}

type Flash struct {
	Text string
	Type string
}

func (app *application) newTemplateData(r *http.Request, data interface{}) templateData {
	versionTxt := "development"
	if len(app.version) != 0 {
		versionTxt = app.version
	}

	flash := newFlash(
		app.sessionManager.PopString(r.Context(), "flashText"),
		app.sessionManager.PopString(r.Context(), "flashType"),
	)

	return templateData{
		Version:         versionTxt,
		Data:            data,
		Flash:           flash,
		IsAuthenticated: app.isAuthenticated(r),
		CurrentDate:     time.Now().Format("2006-01-02"),
	}
}

func newFlash(text, flashType string) *Flash {
	flash := &Flash{
		Text: text,
		Type: flashType,
	}

	if flash.Type == "" {
		flash.Type = "flash-success"
	}
	if flash.Text == "" {
		flash = nil
	}

	return flash
}

var functions = template.FuncMap{
	"numberFormat": numberFormat,
	"sub":          sub,
	"add":          add,
	"seq":          seq,
	"min":          min,
}

func numberFormat(n int) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}

	s := strconv.FormatInt(int64(n), 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}

	return sign + s
}

func sub(a, b int) int {
	return a - b
}

func add(a, b int) int {
	return a + b
}

func seq(n int) []int {
	result := make([]int, n)
	for i := 0; i < n; i++ {
		result[i] = i + 1
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
		
		// Add partials if they exist
		partials, err := fs.Glob(ui.Files, "html/partials/*.tmpl")
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, partials...)

		ts, errPars := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
		if errPars != nil {
			return nil, errPars
		}

		cache[name] = ts
	}
	return cache, nil
}
