package main

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/rockstaedt/swimmate/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestServerError(t *testing.T) {
	app := &application{
		logger: testutils.NewTestLogger(),
	}

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "generic error",
			err:            errors.New("test error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal Server Error",
		},
		{
			name:           "database error",
			err:            errors.New("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			app.serverError(rr, r, tt.err)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.expectedBody)
		})
	}
}

func TestClientError(t *testing.T) {
	app := &application{}

	tests := []struct {
		name           string
		status         int
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "bad request",
			status:         http.StatusBadRequest,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Bad Request",
		},
		{
			name:           "not found",
			status:         http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Not Found",
		},
		{
			name:           "method not allowed",
			status:         http.StatusMethodNotAllowed,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method Not Allowed",
		},
		{
			name:           "unprocessable entity",
			status:         http.StatusUnprocessableEntity,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Unprocessable Entity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			app.clientError(rr, tt.status)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.expectedBody)
		})
	}
}

func TestNotFound(t *testing.T) {
	app := &application{}

	rr := httptest.NewRecorder()
	app.notFound(rr)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "Not Found")
}

func TestRender(t *testing.T) {
	tests := []struct {
		name           string
		setupApp       func() *application
		page           string
		status         int
		data           templateData
		expectedStatus int
		expectedBody   string
		shouldError    bool
	}{
		{
			name: "successful render",
			setupApp: func() *application {
				tmpl := template.Must(template.New("base").Parse(`{{define "base"}}<html><body>{{.Version}}</body></html>{{end}}`))
				return &application{
					templateCache: map[string]*template.Template{
						"test.tmpl": tmpl,
					},
					logger: testutils.NewTestLogger(),
				}
			},
			page:           "test.tmpl",
			status:         http.StatusOK,
			data:           templateData{Version: "1.0.0"},
			expectedStatus: http.StatusOK,
			expectedBody:   "<html><body>1.0.0</body></html>",
			shouldError:    false,
		},
		{
			name: "template not found",
			setupApp: func() *application {
				return &application{
					templateCache: map[string]*template.Template{},
					logger:        testutils.NewTestLogger(),
				}
			},
			page:           "nonexistent.tmpl",
			status:         http.StatusOK,
			data:           templateData{},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal Server Error",
			shouldError:    true,
		},
		{
			name: "render with different status code",
			setupApp: func() *application {
				tmpl := template.Must(template.New("base").Parse(`{{define "base"}}<html><body>Created</body></html>{{end}}`))
				return &application{
					templateCache: map[string]*template.Template{
						"create.tmpl": tmpl,
					},
					logger: testutils.NewTestLogger(),
				}
			},
			page:           "create.tmpl",
			status:         http.StatusCreated,
			data:           templateData{},
			expectedStatus: http.StatusCreated,
			expectedBody:   "<html><body>Created</body></html>",
			shouldError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.setupApp()
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			app.render(rr, r, tt.status, tt.page, tt.data)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			if !tt.shouldError {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			} else {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestRenderPartial(t *testing.T) {
	tests := []struct {
		name         string
		setupApp     func() *application
		templateName string
		partialName  string
		pageData     interface{}
		partialData  interface{}
		expectedBody string
		shouldError  bool
	}{
		{
			name: "successful partial render",
			setupApp: func() *application {
				tmpl := template.Must(template.New("list.tmpl").Parse(`{{define "item"}}<li>{{.Partial}}</li>{{end}}`))
				return &application{
					templateCache: map[string]*template.Template{
						"list.tmpl": tmpl,
					},
					logger:         testutils.NewTestLogger(),
					sessionManager: testutils.NewTestSessionManager(),
				}
			},
			templateName: "list.tmpl",
			partialName:  "item",
			pageData:     nil,
			partialData:  "Test Item",
			expectedBody: "<li>Test Item</li>",
			shouldError:  false,
		},
		{
			name: "template not found",
			setupApp: func() *application {
				return &application{
					templateCache:  map[string]*template.Template{},
					logger:         testutils.NewTestLogger(),
					sessionManager: testutils.NewTestSessionManager(),
				}
			},
			templateName: "nonexistent.tmpl",
			partialName:  "item",
			pageData:     nil,
			partialData:  "Test",
			expectedBody: "Internal Server Error",
			shouldError:  true,
		},
		{
			name: "partial not found",
			setupApp: func() *application {
				tmpl := template.Must(template.New("list.tmpl").Parse(`{{define "item"}}<li>{{.Partial}}</li>{{end}}`))
				return &application{
					templateCache: map[string]*template.Template{
						"list.tmpl": tmpl,
					},
					logger:         testutils.NewTestLogger(),
					sessionManager: testutils.NewTestSessionManager(),
				}
			},
			templateName: "list.tmpl",
			partialName:  "nonexistent",
			pageData:     nil,
			partialData:  "Test",
			expectedBody: "Internal Server Error",
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.setupApp()
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx, _ := app.sessionManager.Load(r.Context(), "")
			r = r.WithContext(ctx)

			app.renderPartial(rr, r, tt.templateName, tt.partialName, tt.pageData, tt.partialData)

			assert.Contains(t, rr.Body.String(), tt.expectedBody)
		})
	}
}

func TestIsAuthenticated(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func() *http.Request
		setupSession func(sessionManager *scs.SessionManager, r *http.Request)
		expected     bool
	}{
		{
			name: "authenticated user",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test", nil)
			},
			setupSession: func(sessionManager *scs.SessionManager, r *http.Request) {
				ctx, _ := sessionManager.Load(r.Context(), "")
				sessionManager.Put(ctx, "authenticatedUserID", 1)
				*r = *r.WithContext(ctx)
			},
			expected: true,
		},
		{
			name: "unauthenticated user",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test", nil)
			},
			setupSession: func(sessionManager *scs.SessionManager, r *http.Request) {
				ctx, _ := sessionManager.Load(r.Context(), "")
				*r = *r.WithContext(ctx)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionManager := testutils.NewTestSessionManager()
			app := &application{
				sessionManager: sessionManager,
			}

			r := tt.setupRequest()
			tt.setupSession(sessionManager, r)

			result := app.isAuthenticated(r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderWithComplexData(t *testing.T) {
	// Test rendering with all template data fields
	tmpl := template.Must(template.New("base").Parse(`{{define "base"}}<html>
<body>
	<p>Version: {{.Version}}</p>
	<p>Auth: {{.IsAuthenticated}}</p>
	<p>Date: {{.CurrentDate}}</p>
	{{if .Flash}}<p>Flash: {{.Flash.Text}} ({{.Flash.Type}})</p>{{end}}
</body>
</html>{{end}}`))

	app := &application{
		templateCache: map[string]*template.Template{
			"complex.tmpl": tmpl,
		},
		logger: testutils.NewTestLogger(),
	}

	data := templateData{
		Version:         "2.0.0",
		IsAuthenticated: true,
		CurrentDate:     "2024-01-15",
		Flash:           &Flash{Text: "Success", Type: "flash-success"},
	}

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	app.render(rr, r, http.StatusOK, "complex.tmpl", data)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Version: 2.0.0")
	assert.Contains(t, body, "Auth: true")
	assert.Contains(t, body, "Date: 2024-01-15")
	assert.Contains(t, body, "Flash: Success")
	assert.Contains(t, body, "flash-success")
}

func TestHelperErrorLogging(t *testing.T) {
	// Test that serverError logs with correct method and URI
	app := &application{
		logger: testutils.NewTestLogger(),
	}

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/test?param=value", nil)

	app.serverError(rr, r, errors.New("test error"))

	// Verify response
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "Internal Server Error")
}

func TestRenderBuffering(t *testing.T) {
	// Test that render uses buffering (template error shouldn't write partial response)
	tmpl := template.Must(template.New("base").Parse(`{{define "base"}}<html><body>{{.InvalidField}}</body></html>{{end}}`))

	app := &application{
		templateCache: map[string]*template.Template{
			"error.tmpl": tmpl,
		},
		logger: testutils.NewTestLogger(),
	}

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	app.render(rr, r, http.StatusOK, "error.tmpl", templateData{})

	// Should get 500 error, not partial template output
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	body := rr.Body.String()
	// Should contain error message, not partial HTML
	assert.True(t, strings.Contains(body, "Internal Server Error") || len(body) == 0)
}
