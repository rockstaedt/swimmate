package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoutes(t *testing.T) {
	app := newTestApplication()

	app.templateCache["login.tmpl"] = createTestTemplate("base", `{{define "base"}}Login{{end}}`)
	app.templateCache["about.tmpl"] = createTestTemplate("base", `{{define "base"}}About{{end}}`)
	app.templateCache["home.tmpl"] = createTestTemplate("base", `{{define "base"}}Home{{end}}`)
	app.templateCache["swims.tmpl"] = createTestTemplate("swims.tmpl", `
		{{define "base"}}Swims{{end}}
		{{define "swim-row"}}<tr>{{.DistanceM}}</tr>{{end}}
		{{define "load-more-button"}}<button>Load More</button>{{end}}
	`)
	app.templateCache["yearly-figures.tmpl"] = createTestTemplate("base", `{{define "base"}}Yearly{{end}}`)
	app.templateCache["swim-create.tmpl"] = createTestTemplate("base", `{{define "base"}}Create{{end}}`)
	app.templateCache["swim-edit.tmpl"] = createTestTemplate("base", `{{define "base"}}Edit{{end}}`)

	handler := app.routes()

	tests := []struct {
		name           string
		method         string
		path           string
		authenticated  bool
		expectedStatus int
		description    string
	}{
		{
			name:           "public login page",
			method:         http.MethodGet,
			path:           "/login",
			authenticated:  false,
			expectedStatus: http.StatusOK,
			description:    "Login page should be accessible without authentication",
		},
		{
			name:           "public about page",
			method:         http.MethodGet,
			path:           "/about",
			authenticated:  false,
			expectedStatus: http.StatusOK,
			description:    "About page should be accessible without authentication",
		},
		{
			name:           "home requires authentication",
			method:         http.MethodGet,
			path:           "/",
			authenticated:  false,
			expectedStatus: http.StatusSeeOther,
			description:    "Home page should redirect to login when not authenticated",
		},
		{
			name:           "home with authentication",
			method:         http.MethodGet,
			path:           "/",
			authenticated:  true,
			expectedStatus: http.StatusOK,
			description:    "Home page should be accessible when authenticated",
		},
		{
			name:           "swims list requires authentication",
			method:         http.MethodGet,
			path:           "/swims",
			authenticated:  false,
			expectedStatus: http.StatusSeeOther,
			description:    "Swims list should redirect to login when not authenticated",
		},
		{
			name:           "swims list with authentication",
			method:         http.MethodGet,
			path:           "/swims",
			authenticated:  true,
			expectedStatus: http.StatusOK,
			description:    "Swims list should be accessible when authenticated",
		},
		{
			name:           "yearly figures requires authentication",
			method:         http.MethodGet,
			path:           "/yearly-figures",
			authenticated:  false,
			expectedStatus: http.StatusSeeOther,
			description:    "Yearly figures should redirect to login when not authenticated",
		},
		{
			name:           "yearly figures with authentication",
			method:         http.MethodGet,
			path:           "/yearly-figures",
			authenticated:  true,
			expectedStatus: http.StatusOK,
			description:    "Yearly figures should be accessible when authenticated",
		},
		{
			name:           "create swim requires authentication",
			method:         http.MethodGet,
			path:           "/swim",
			authenticated:  false,
			expectedStatus: http.StatusSeeOther,
			description:    "Create swim page should redirect to login when not authenticated",
		},
		{
			name:           "create swim with authentication",
			method:         http.MethodGet,
			path:           "/swim",
			authenticated:  true,
			expectedStatus: http.StatusOK,
			description:    "Create swim page should be accessible when authenticated",
		},
		{
			name:           "swims more requires authentication",
			method:         http.MethodGet,
			path:           "/swims/more?offset=20",
			authenticated:  false,
			expectedStatus: http.StatusSeeOther,
			description:    "Swims more should redirect to login when not authenticated",
		},
		{
			name:           "swims more with authentication",
			method:         http.MethodGet,
			path:           "/swims/more?offset=20",
			authenticated:  true,
			expectedStatus: http.StatusOK,
			description:    "Swims more should be accessible when authenticated",
		},
		{
			name:           "not found route",
			method:         http.MethodGet,
			path:           "/nonexistent",
			authenticated:  false,
			expectedStatus: http.StatusNotFound,
			description:    "Non-existent routes should return 404",
		},
		{
			name:           "not found route with authentication",
			method:         http.MethodGet,
			path:           "/does-not-exist",
			authenticated:  true,
			expectedStatus: http.StatusNotFound,
			description:    "Non-existent routes should return 404 even when authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(tt.method, tt.path, nil)

			if tt.authenticated {
				ctx, _ := app.sessionManager.Load(r.Context(), "")
				app.sessionManager.Put(ctx, "authenticatedUserID", 1)
				r = r.WithContext(ctx)
			}

			handler.ServeHTTP(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code, tt.description)

			if tt.expectedStatus == http.StatusSeeOther && !tt.authenticated {
				location := rr.Header().Get("Location")
				assert.Equal(t, "/login", location, "Should redirect to login page")
			}
		})
	}
}

func TestRoutes_HTTPMethodEnforcement(t *testing.T) {
	app := newTestApplication()

	app.templateCache["login.tmpl"] = createTestTemplate("base", `{{define "base"}}Login{{end}}`)
	app.templateCache["home.tmpl"] = createTestTemplate("base", `{{define "base"}}Home{{end}}`)
	app.templateCache["swim-create.tmpl"] = createTestTemplate("base", `{{define "base"}}Create{{end}}`)
	app.templateCache["swim-edit.tmpl"] = createTestTemplate("base", `{{define "base"}}Edit{{end}}`)

	handler := app.routes()

	tests := []struct {
		name             string
		method           string
		path             string
		authenticated    bool
		expectedStatus   int
		shouldBeAllowed  bool
	}{
		{
			name:            "GET /login allowed",
			method:          http.MethodGet,
			path:            "/login",
			authenticated:   false,
			expectedStatus:  http.StatusOK,
			shouldBeAllowed: true,
		},
		{
			name:            "POST /login not allowed",
			method:          http.MethodPost,
			path:            "/login",
			authenticated:   false,
			expectedStatus:  http.StatusMethodNotAllowed,
			shouldBeAllowed: false,
		},
		{
			name:            "POST /authenticate allowed",
			method:          http.MethodPost,
			path:            "/authenticate",
			authenticated:   false,
			expectedStatus:  http.StatusOK,
			shouldBeAllowed: true,
		},
		{
			name:            "GET /authenticate not allowed",
			method:          http.MethodGet,
			path:            "/authenticate",
			authenticated:   false,
			expectedStatus:  http.StatusMethodNotAllowed,
			shouldBeAllowed: false,
		},
		{
			name:            "POST /logout allowed",
			method:          http.MethodPost,
			path:            "/logout",
			authenticated:   false,
			expectedStatus:  http.StatusSeeOther,
			shouldBeAllowed: true,
		},
		{
			name:            "GET /logout not allowed",
			method:          http.MethodGet,
			path:            "/logout",
			authenticated:   false,
			expectedStatus:  http.StatusMethodNotAllowed,
			shouldBeAllowed: false,
		},
		{
			name:            "GET / allowed when authenticated",
			method:          http.MethodGet,
			path:            "/",
			authenticated:   true,
			expectedStatus:  http.StatusOK,
			shouldBeAllowed: true,
		},
		{
			name:            "POST / not allowed",
			method:          http.MethodPost,
			path:            "/",
			authenticated:   true,
			expectedStatus:  http.StatusMethodNotAllowed,
			shouldBeAllowed: false,
		},
		{
			name:            "GET /swim allowed when authenticated",
			method:          http.MethodGet,
			path:            "/swim",
			authenticated:   true,
			expectedStatus:  http.StatusOK,
			shouldBeAllowed: true,
		},
		{
			name:            "POST /swim allowed when authenticated",
			method:          http.MethodPost,
			path:            "/swim",
			authenticated:   true,
			expectedStatus:  http.StatusBadRequest,
			shouldBeAllowed: true,
		},
		{
			name:            "PUT /swim not allowed",
			method:          http.MethodPut,
			path:            "/swim",
			authenticated:   true,
			expectedStatus:  http.StatusMethodNotAllowed,
			shouldBeAllowed: false,
		},
		{
			name:            "GET /swims/edit/:id allowed when authenticated",
			method:          http.MethodGet,
			path:            "/swims/edit/1",
			authenticated:   true,
			expectedStatus:  http.StatusOK,
			shouldBeAllowed: true,
		},
		{
			name:            "POST /swims/edit/:id allowed when authenticated",
			method:          http.MethodPost,
			path:            "/swims/edit/1",
			authenticated:   true,
			expectedStatus:  http.StatusBadRequest,
			shouldBeAllowed: true,
		},
		{
			name:            "PUT /swims/edit/:id allowed when authenticated",
			method:          http.MethodPut,
			path:            "/swims/edit/1",
			authenticated:   true,
			expectedStatus:  http.StatusBadRequest,
			shouldBeAllowed: true,
		},
		{
			name:            "DELETE /swims/edit/:id not allowed",
			method:          http.MethodDelete,
			path:            "/swims/edit/1",
			authenticated:   true,
			expectedStatus:  http.StatusMethodNotAllowed,
			shouldBeAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(tt.method, tt.path, nil)

			if tt.authenticated {
				ctx, _ := app.sessionManager.Load(r.Context(), "")
				app.sessionManager.Put(ctx, "authenticatedUserID", 1)
				r = r.WithContext(ctx)
			}

			handler.ServeHTTP(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestRoutes_StaticFiles(t *testing.T) {
	app := newTestApplication()
	handler := app.routes()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "static CSS file exists",
			path:           "/static/css/main.css",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "static file not found",
			path:           "/static/nonexistent.css",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)

			handler.ServeHTTP(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestRoutes_MiddlewareChain(t *testing.T) {
	app := newTestApplication()

	app.templateCache["home.tmpl"] = createTestTemplate("base", `{{define "base"}}Home{{end}}`)

	handler := app.routes()

	t.Run("authentication middleware redirects unauthenticated requests", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		handler.ServeHTTP(rr, r)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/login", rr.Header().Get("Location"))
	})

	t.Run("authentication middleware allows authenticated requests", func(t *testing.T) {
		app2 := newTestApplication()
		app2.templateCache["home.tmpl"] = createTestTemplate("base", `{{define "base"}}Home{{end}}`)
		handler2 := app2.routes()

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		ctx, _ := app2.sessionManager.Load(r.Context(), "")
		app2.sessionManager.Put(ctx, "authenticatedUserID", 1)
		r = r.WithContext(ctx)

		handler2.ServeHTTP(rr, r)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestRoutes_NotFoundHandler(t *testing.T) {
	app := newTestApplication()
	handler := app.routes()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "random path",
			path: "/random/path/that/does/not/exist",
		},
		{
			name: "typo in route",
			path: "/swimms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)

			handler.ServeHTTP(rr, r)

			assert.Equal(t, http.StatusNotFound, rr.Code)
		})
	}
}
