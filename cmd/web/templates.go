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
	CurrentYear     int
	CurrentMonth    int
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

	now := time.Now()
	return templateData{
		Version:         versionTxt,
		Data:            data,
		Flash:           flash,
		IsAuthenticated: app.isAuthenticated(r),
		CurrentDate:     now.Format("2006-01-02"),
		CurrentYear:     now.Year(),
		CurrentMonth:    int(now.Month()),
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
	"div":          div,
	"seq":          seq,
	"min":          min,
	"emptyStars":   emptyStars,
	"atoi":         atoi,
	"slice":        slice,
	"monthAbbr":    monthAbbr,
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

func div(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
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

func emptyStars(assessment int) int {
	maxStars := 2
	return maxStars - min(assessment, maxStars)
}

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func slice(s string, start, end int) string {
	if start < 0 || start > len(s) {
		return ""
	}
	if end < 0 || end > len(s) {
		end = len(s)
	}
	if start >= end {
		return ""
	}
	return s[start:end]
}

func monthAbbr(month int) string {
	abbrs := []string{"J", "F", "M", "A", "M", "J", "J", "A", "S", "O", "N", "D"}
	if month < 1 || month > 12 {
		return ""
	}
	return abbrs[month-1]
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
