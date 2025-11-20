package main

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/rockstaedt/swimmate/internal/models"
	"github.com/rockstaedt/swimmate/internal/testutils"
	"github.com/stretchr/testify/assert"
)

// Helper to create a test application with mocks
func newTestApplication() *application {
	return &application{
		logger:         testutils.NewTestLogger(),
		swims:          &testutils.MockSwimModel{},
		users:          &testutils.MockUserModel{},
		sessionManager: testutils.NewTestSessionManager(),
		templateCache:  make(map[string]*template.Template),
		version:        "test",
	}
}

// Helper to create a simple template for testing
func createTestTemplate(name string, content string) *template.Template {
	return template.Must(template.New(name).Funcs(functions).Parse(content))
}

func TestHome(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setupMock      func(*testutils.MockSwimModel)
		expectedStatus int
		shouldCall404  bool
	}{
		{
			name: "successful home page render",
			path: "/",
			setupMock: func(m *testutils.MockSwimModel) {
				m.SummarizeFunc = func(userId int) *models.SwimSummary {
					return &models.SwimSummary{
						TotalDistance: 5000,
						TotalCount:    10,
						YearMap:       make(map[int]models.YearMap),
					}
				}
			},
			expectedStatus: http.StatusOK,
			shouldCall404:  false,
		},
		{
			name:           "wrong path returns 404",
			path:           "/invalid",
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusNotFound,
			shouldCall404:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()
			mockSwims := &testutils.MockSwimModel{}
			tt.setupMock(mockSwims)
			app.swims = mockSwims

			app.templateCache["home.tmpl"] = createTestTemplate("base", `{{define "base"}}Home{{end}}`)

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)

			ctx, _ := app.sessionManager.Load(r.Context(), "")
			app.sessionManager.Put(ctx, "authenticatedUserID", 1)
			r = r.WithContext(ctx)

			app.home(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestLogin(t *testing.T) {
	app := newTestApplication()
	app.templateCache["login.tmpl"] = createTestTemplate("base", `{{define "base"}}Login{{end}}`)

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)

	ctx, _ := app.sessionManager.Load(r.Context(), "")
	r = r.WithContext(ctx)

	app.login(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Login")
}

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name             string
		formData         url.Values
		setupMock        func(*testutils.MockUserModel)
		expectedStatus   int
		expectedLocation string
		expectFlash      bool
		flashType        string
	}{
		{
			name: "successful authentication",
			formData: url.Values{
				"username": []string{"testuser"},
				"password": []string{"password123"},
			},
			setupMock: func(m *testutils.MockUserModel) {
				m.AuthenticateFunc = func(username, password string) (int, error) {
					if username == "testuser" && password == "password123" {
						return 1, nil
					}
					return 0, models.ErrInvalidCredentials
				}
			},
			expectedStatus:   http.StatusSeeOther,
			expectedLocation: "/",
			expectFlash:      false,
		},
		{
			name: "invalid credentials",
			formData: url.Values{
				"username": []string{"testuser"},
				"password": []string{"wrongpassword"},
			},
			setupMock: func(m *testutils.MockUserModel) {
				m.AuthenticateFunc = func(username, password string) (int, error) {
					return 0, models.ErrInvalidCredentials
				}
			},
			expectedStatus: http.StatusOK,
			expectFlash:    true,
			flashType:      "flash-error",
		},
		{
			name: "database error",
			formData: url.Values{
				"username": []string{"testuser"},
				"password": []string{"password123"},
			},
			setupMock: func(m *testutils.MockUserModel) {
				m.AuthenticateFunc = func(username, password string) (int, error) {
					return 0, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "empty username",
			formData: url.Values{
				"username": []string{""},
				"password": []string{"password123"},
			},
			setupMock: func(m *testutils.MockUserModel) {
				m.AuthenticateFunc = func(username, password string) (int, error) {
					return 0, models.ErrInvalidCredentials
				}
			},
			expectedStatus: http.StatusOK,
			expectFlash:    true,
			flashType:      "flash-error",
		},
		{
			name: "empty password",
			formData: url.Values{
				"username": []string{"testuser"},
				"password": []string{""},
			},
			setupMock: func(m *testutils.MockUserModel) {
				m.AuthenticateFunc = func(username, password string) (int, error) {
					return 0, models.ErrInvalidCredentials
				}
			},
			expectedStatus: http.StatusOK,
			expectFlash:    true,
			flashType:      "flash-error",
		},
		{
			name: "successful authentication sets session",
			formData: url.Values{
				"username": []string{"admin"},
				"password": []string{"admin123"},
			},
			setupMock: func(m *testutils.MockUserModel) {
				m.AuthenticateFunc = func(username, password string) (int, error) {
					if username == "admin" && password == "admin123" {
						return 42, nil
					}
					return 0, models.ErrInvalidCredentials
				}
			},
			expectedStatus:   http.StatusSeeOther,
			expectedLocation: "/",
			expectFlash:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()
			mockUsers := &testutils.MockUserModel{}
			tt.setupMock(mockUsers)
			app.users = mockUsers

			app.templateCache["login.tmpl"] = createTestTemplate("base", `{{define "base"}}Login{{end}}`)

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/authenticate", strings.NewReader(tt.formData.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			ctx, _ := app.sessionManager.Load(r.Context(), "")
			r = r.WithContext(ctx)

			app.authenticate(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedLocation != "" {
				assert.Equal(t, tt.expectedLocation, rr.Header().Get("Location"))
			}
		})
	}
}

func TestLogout(t *testing.T) {
	tests := []struct {
		name               string
		authenticated      bool
		expectedStatus     int
		expectedLocation   string
		expectFlashMessage bool
	}{
		{
			name:               "successful logout with authenticated user",
			authenticated:      true,
			expectedStatus:     http.StatusSeeOther,
			expectedLocation:   "/login",
			expectFlashMessage: true,
		},
		{
			name:               "logout without authenticated user",
			authenticated:      false,
			expectedStatus:     http.StatusSeeOther,
			expectedLocation:   "/login",
			expectFlashMessage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/logout", nil)

			ctx, _ := app.sessionManager.Load(r.Context(), "")
			if tt.authenticated {
				app.sessionManager.Put(ctx, "authenticatedUserID", 1)
			}
			r = r.WithContext(ctx)

			app.logout(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedLocation, rr.Header().Get("Location"))

			if tt.expectFlashMessage {
				flashText := app.sessionManager.GetString(ctx, "flashText")
				assert.NotEmpty(t, flashText)
			}

			userID := app.sessionManager.GetInt(ctx, "authenticatedUserID")
			assert.Equal(t, 0, userID, "authenticatedUserID should be removed from session")
		})
	}
}

func TestYearlyFigures(t *testing.T) {
	tests := []struct {
		name       string
		queryParam string
		setupMock  func(*testutils.MockSwimModel)
	}{
		{
			name:       "current year",
			queryParam: "",
			setupMock: func(m *testutils.MockSwimModel) {
				m.SummarizeFunc = func(userId int) *models.SwimSummary {
					return &models.SwimSummary{YearMap: make(map[int]models.YearMap)}
				}
			},
		},
		{
			name:       "specific year",
			queryParam: "?year=2023",
			setupMock: func(m *testutils.MockSwimModel) {
				m.SummarizeFunc = func(userId int) *models.SwimSummary {
					return &models.SwimSummary{YearMap: make(map[int]models.YearMap)}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()
			mockSwims := &testutils.MockSwimModel{}
			tt.setupMock(mockSwims)
			app.swims = mockSwims

			app.templateCache["yearly-figures.tmpl"] = createTestTemplate("base", `{{define "base"}}Yearly{{end}}`)

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/yearly-figures"+tt.queryParam, nil)

			ctx, _ := app.sessionManager.Load(r.Context(), "")
			app.sessionManager.Put(ctx, "authenticatedUserID", 1)
			r = r.WithContext(ctx)

			app.yearlyFigures(rr, r)

			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestAbout(t *testing.T) {
	app := newTestApplication()
	app.templateCache["about.tmpl"] = createTestTemplate("base", `{{define "base"}}About{{end}}`)

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/about", nil)

	ctx, _ := app.sessionManager.Load(r.Context(), "")
	r = r.WithContext(ctx)

	app.about(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "About")
}

func TestCreateSwim(t *testing.T) {
	app := newTestApplication()
	app.templateCache["swim-create.tmpl"] = createTestTemplate("base", `{{define "base"}}Create{{end}}`)

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/swim", nil)

	ctx, _ := app.sessionManager.Load(r.Context(), "")
	r = r.WithContext(ctx)

	app.createSwim(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Create")
}

func TestSwimsList(t *testing.T) {
	tests := []struct {
		name       string
		requestURL string
		setupMock  func(*testutils.MockSwimModel, *testing.T)
		wantErr    bool
	}{
		{
			name:       "successful list with default sort",
			requestURL: "/swims",
			setupMock: func(m *testutils.MockSwimModel, t *testing.T) {
				m.GetPaginatedFunc = func(userId int, limit int, offset int, sort string, direction string) ([]*models.Swim, error) {
					assert.Equal(t, models.SwimSortDate, sort)
					assert.Equal(t, models.SortDirectionDesc, direction)
					return []*models.Swim{
						{Date: time.Now(), DistanceM: 1000, Assessment: 2},
					}, nil
				}
			},
		},
		{
			name:       "custom sort parameters",
			requestURL: "/swims?sort=distance&direction=asc",
			setupMock: func(m *testutils.MockSwimModel, t *testing.T) {
				m.GetPaginatedFunc = func(userId int, limit int, offset int, sort string, direction string) ([]*models.Swim, error) {
					assert.Equal(t, models.SwimSortDistance, sort)
					assert.Equal(t, models.SortDirectionAsc, direction)
					return []*models.Swim{
						{Date: time.Now(), DistanceM: 850, Assessment: 1},
					}, nil
				}
			},
		},
		{
			name:       "database error",
			requestURL: "/swims",
			setupMock: func(m *testutils.MockSwimModel, t *testing.T) {
				m.GetPaginatedFunc = func(userId int, limit int, offset int, sort string, direction string) ([]*models.Swim, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()
			mockSwims := &testutils.MockSwimModel{}
			tt.setupMock(mockSwims, t)
			app.swims = mockSwims

			app.templateCache["swims.tmpl"] = createTestTemplate("base", `{{define "base"}}Swims{{end}}`)

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, tt.requestURL, nil)

			ctx, _ := app.sessionManager.Load(r.Context(), "")
			app.sessionManager.Put(ctx, "authenticatedUserID", 1)
			r = r.WithContext(ctx)

			app.swimsList(rr, r)

			if tt.wantErr {
				assert.Equal(t, http.StatusInternalServerError, rr.Code)
			} else {
				assert.Equal(t, http.StatusOK, rr.Code)
			}
		})
	}
}

func TestSwimsMore(t *testing.T) {
	tests := []struct {
		name         string
		requestURL   string
		htmxRequest  bool
		setupMock    func(*testutils.MockSwimModel, *testing.T)
		expectStatus int
	}{
		{
			name:        "HTMX request with swims",
			requestURL:  "/swims/more?offset=20",
			htmxRequest: true,
			setupMock: func(m *testutils.MockSwimModel, t *testing.T) {
				m.GetPaginatedFunc = func(userId int, limit int, offset int, sort string, direction string) ([]*models.Swim, error) {
					assert.Equal(t, models.SwimSortDate, sort)
					assert.Equal(t, models.SortDirectionDesc, direction)
					swims := make([]*models.Swim, 20)
					for i := 0; i < 20; i++ {
						swims[i] = &models.Swim{Date: time.Now(), DistanceM: 1000, Assessment: 2}
					}
					return swims, nil
				}
			},
			expectStatus: http.StatusOK,
		},
		{
			name:        "regular request with custom sort",
			requestURL:  "/swims/more?offset=20&sort=assessment&direction=asc",
			htmxRequest: false,
			setupMock: func(m *testutils.MockSwimModel, t *testing.T) {
				m.GetPaginatedFunc = func(userId int, limit int, offset int, sort string, direction string) ([]*models.Swim, error) {
					assert.Equal(t, models.SwimSortAssessment, sort)
					assert.Equal(t, models.SortDirectionAsc, direction)
					return []*models.Swim{
						{Date: time.Now(), DistanceM: 1000, Assessment: 2},
					}, nil
				}
			},
			expectStatus: http.StatusOK,
		},
		{
			name:        "HTMX request with database error",
			requestURL:  "/swims/more?offset=20",
			htmxRequest: true,
			setupMock: func(m *testutils.MockSwimModel, t *testing.T) {
				m.GetPaginatedFunc = func(userId int, limit int, offset int, sort string, direction string) ([]*models.Swim, error) {
					return nil, errors.New("database error")
				}
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:        "non-HTMX request with database error",
			requestURL:  "/swims/more?offset=20",
			htmxRequest: false,
			setupMock: func(m *testutils.MockSwimModel, t *testing.T) {
				m.GetPaginatedFunc = func(userId int, limit int, offset int, sort string, direction string) ([]*models.Swim, error) {
					return nil, errors.New("database error")
				}
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:        "invalid offset defaults to zero",
			requestURL:  "/swims/more?offset=abc",
			htmxRequest: true,
			setupMock: func(m *testutils.MockSwimModel, t *testing.T) {
				m.GetPaginatedFunc = func(userId int, limit int, offset int, sort string, direction string) ([]*models.Swim, error) {
					assert.Equal(t, 0, offset, "invalid offset should default to 0")
					return []*models.Swim{
						{Date: time.Now(), DistanceM: 1000, Assessment: 2},
					}, nil
				}
			},
			expectStatus: http.StatusOK,
		},
		{
			name:        "negative offset defaults to zero",
			requestURL:  "/swims/more?offset=-10",
			htmxRequest: true,
			setupMock: func(m *testutils.MockSwimModel, t *testing.T) {
				m.GetPaginatedFunc = func(userId int, limit int, offset int, sort string, direction string) ([]*models.Swim, error) {
					return []*models.Swim{
						{Date: time.Now(), DistanceM: 1000, Assessment: 2},
					}, nil
				}
			},
			expectStatus: http.StatusOK,
		},
		{
			name:        "HTMX request with empty results",
			requestURL:  "/swims/more?offset=100",
			htmxRequest: true,
			setupMock: func(m *testutils.MockSwimModel, t *testing.T) {
				m.GetPaginatedFunc = func(userId int, limit int, offset int, sort string, direction string) ([]*models.Swim, error) {
					return []*models.Swim{}, nil
				}
			},
			expectStatus: http.StatusOK,
		},
		{
			name:        "non-HTMX request with empty results",
			requestURL:  "/swims/more?offset=100",
			htmxRequest: false,
			setupMock: func(m *testutils.MockSwimModel, t *testing.T) {
				m.GetPaginatedFunc = func(userId int, limit int, offset int, sort string, direction string) ([]*models.Swim, error) {
					return []*models.Swim{}, nil
				}
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()
			mockSwims := &testutils.MockSwimModel{}
			tt.setupMock(mockSwims, t)
			app.swims = mockSwims

			app.templateCache["swims.tmpl"] = createTestTemplate("swims.tmpl", `
				{{define "base"}}Swims{{end}}
				{{define "swim-row"}}<tr>{{.Partial.DistanceM}}</tr>{{end}}
				{{define "load-more-button"}}<button>Load More</button>{{end}}
			`)

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, tt.requestURL, nil)

			if tt.htmxRequest {
				r.Header.Set("HX-Request", "true")
			}

			ctx, _ := app.sessionManager.Load(r.Context(), "")
			app.sessionManager.Put(ctx, "authenticatedUserID", 1)
			r = r.WithContext(ctx)

			app.swimsMore(rr, r)

			assert.Equal(t, tt.expectStatus, rr.Code)
		})
	}
}

func TestParseSwimSort(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantSort   string
		wantDirect string
	}{
		{
			name:       "defaults when not provided",
			url:        "/swims",
			wantSort:   models.SwimSortDate,
			wantDirect: models.SortDirectionDesc,
		},
		{
			name:       "custom values",
			url:        "/swims?sort=distance&direction=asc",
			wantSort:   models.SwimSortDistance,
			wantDirect: models.SortDirectionAsc,
		},
		{
			name:       "invalid values fallback",
			url:        "/swims?sort=unknown&direction=sideways",
			wantSort:   models.SwimSortDate,
			wantDirect: models.SortDirectionDesc,
		},
		{
			name:       "upper case parameters are normalized",
			url:        "/swims?sort=ASSESSMENT&direction=ASC",
			wantSort:   models.SwimSortAssessment,
			wantDirect: models.SortDirectionAsc,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			gotSort, gotDirection := parseSwimSort(req)
			assert.Equal(t, tt.wantSort, gotSort)
			assert.Equal(t, tt.wantDirect, gotDirection)
		})
	}
}

func TestStoreSwim(t *testing.T) {
	tests := []struct {
		name             string
		formData         url.Values
		setupMock        func(*testutils.MockSwimModel)
		expectedStatus   int
		expectedLocation string
	}{
		{
			name: "successful swim creation",
			formData: url.Values{
				"date":       []string{"2024-01-15"},
				"distance_m": []string{"1500"},
				"assessment": []string{"2"},
			},
			setupMock: func(m *testutils.MockSwimModel) {
				m.InsertFunc = func(date time.Time, distanceM int, assessment int, userId int) error {
					return nil
				}
			},
			expectedStatus:   http.StatusSeeOther,
			expectedLocation: "/",
		},
		{
			name: "invalid date format",
			formData: url.Values{
				"date":       []string{"invalid-date"},
				"distance_m": []string{"1500"},
				"assessment": []string{"2"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid distance",
			formData: url.Values{
				"date":       []string{"2024-01-15"},
				"distance_m": []string{"not-a-number"},
				"assessment": []string{"2"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid assessment",
			formData: url.Values{
				"date":       []string{"2024-01-15"},
				"distance_m": []string{"1500"},
				"assessment": []string{"invalid"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "database error on insert",
			formData: url.Values{
				"date":       []string{"2024-01-15"},
				"distance_m": []string{"1500"},
				"assessment": []string{"2"},
			},
			setupMock: func(m *testutils.MockSwimModel) {
				m.InsertFunc = func(date time.Time, distanceM int, assessment int, userId int) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "zero distance",
			formData: url.Values{
				"date":       []string{"2024-01-15"},
				"distance_m": []string{"0"},
				"assessment": []string{"2"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "negative distance",
			formData: url.Values{
				"date":       []string{"2024-01-15"},
				"distance_m": []string{"-100"},
				"assessment": []string{"2"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "negative assessment",
			formData: url.Values{
				"date":       []string{"2024-01-15"},
				"distance_m": []string{"1500"},
				"assessment": []string{"-1"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "assessment too large",
			formData: url.Values{
				"date":       []string{"2024-01-15"},
				"distance_m": []string{"1500"},
				"assessment": []string{"3"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()
			mockSwims := &testutils.MockSwimModel{}
			tt.setupMock(mockSwims)
			app.swims = mockSwims

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/swim", strings.NewReader(tt.formData.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			ctx, _ := app.sessionManager.Load(r.Context(), "")
			app.sessionManager.Put(ctx, "authenticatedUserID", 1)
			r = r.WithContext(ctx)

			app.storeSwim(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedLocation != "" {
				assert.Equal(t, tt.expectedLocation, rr.Header().Get("Location"))
			}
		})
	}
}

func TestEditSwim(t *testing.T) {
	tests := []struct {
		name           string
		swimID         string
		setupMock      func(*testutils.MockSwimModel)
		expectedStatus int
	}{
		{
			name:   "renders edit page",
			swimID: "5",
			setupMock: func(m *testutils.MockSwimModel) {
				m.GetByIDFunc = func(userId int, swimId int) (*models.Swim, error) {
					assert.Equal(t, 1, userId)
					assert.Equal(t, 5, swimId)
					return &models.Swim{Id: 5}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "swim not found",
			swimID: "2",
			setupMock: func(m *testutils.MockSwimModel) {
				m.GetByIDFunc = func(userId int, swimId int) (*models.Swim, error) {
					return &models.Swim{}, models.ErrNoRecord
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid swim id",
			swimID:         "abc",
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()
			app.templateCache["swim-edit.tmpl"] = createTestTemplate("base", `{{define "base"}}Edit{{end}}`)

			mockSwims := &testutils.MockSwimModel{}
			tt.setupMock(mockSwims)
			app.swims = mockSwims

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/swims/edit/"+tt.swimID, nil)

			ctx, _ := app.sessionManager.Load(r.Context(), "")
			app.sessionManager.Put(ctx, "authenticatedUserID", 1)
			ctx = context.WithValue(ctx, httprouter.ParamsKey, httprouter.Params{{Key: "id", Value: tt.swimID}})
			r = r.WithContext(ctx)

			app.editSwim(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestUpdateSwim(t *testing.T) {
	validForm := url.Values{
		"date":       []string{"2024-02-01"},
		"distance_m": []string{"2000"},
		"assessment": []string{"2"},
		"sort":       []string{models.SwimSortDistance},
		"direction":  []string{models.SortDirectionAsc},
	}

	tests := []struct {
		name             string
		swimID           string
		form             url.Values
		setupMock        func(*testutils.MockSwimModel)
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:   "successful update",
			swimID: "5",
			form:   validForm,
			setupMock: func(m *testutils.MockSwimModel) {
				m.UpdateFunc = func(id int, userId int, date time.Time, distanceM int, assessment int) error {
					assert.Equal(t, 5, id)
					assert.Equal(t, 1, userId)
					assert.Equal(t, 2000, distanceM)
					assert.Equal(t, 2, assessment)
					return nil
				}
			},
			expectedStatus:   http.StatusSeeOther,
			expectedLocation: "/swims?direction=asc&sort=distance",
		},
		{
			name:   "successful update with missing sort parameters",
			swimID: "9",
			form: url.Values{
				"date":       []string{"2024-02-01"},
				"distance_m": []string{"2000"},
				"assessment": []string{"2"},
			},
			setupMock: func(m *testutils.MockSwimModel) {
				m.UpdateFunc = func(id int, userId int, date time.Time, distanceM int, assessment int) error {
					assert.Equal(t, 9, id)
					return nil
				}
			},
			expectedStatus:   http.StatusSeeOther,
			expectedLocation: "/swims?direction=desc&sort=date",
		},
		{
			name:   "swim not found on update",
			swimID: "10",
			form:   validForm,
			setupMock: func(m *testutils.MockSwimModel) {
				m.UpdateFunc = func(id int, userId int, date time.Time, distanceM int, assessment int) error {
					return models.ErrNoRecord
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "invalid form data",
			swimID: "3",
			form: url.Values{
				"date":       []string{"invalid-date"},
				"distance_m": []string{"2000"},
				"assessment": []string{"2"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "update error",
			swimID: "6",
			form:   validForm,
			setupMock: func(m *testutils.MockSwimModel) {
				m.UpdateFunc = func(id int, userId int, date time.Time, distanceM int, assessment int) error {
					return errors.New("db error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "negative swim ID",
			swimID:         "-5",
			form:           validForm,
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "zero swim ID",
			swimID:         "0",
			form:           validForm,
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "non-numeric swim ID",
			swimID:         "abc",
			form:           validForm,
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "invalid distance format",
			swimID: "5",
			form: url.Values{
				"date":       []string{"2024-02-01"},
				"distance_m": []string{"not-a-number"},
				"assessment": []string{"2"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid assessment format",
			swimID: "5",
			form: url.Values{
				"date":       []string{"2024-02-01"},
				"distance_m": []string{"2000"},
				"assessment": []string{"invalid"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "extremely large distance",
			swimID: "5",
			form: url.Values{
				"date":       []string{"2024-02-01"},
				"distance_m": []string{"999999"},
				"assessment": []string{"2"},
			},
			setupMock: func(m *testutils.MockSwimModel) {
				m.UpdateFunc = func(id int, userId int, date time.Time, distanceM int, assessment int) error {
					assert.Equal(t, 999999, distanceM)
					return nil
				}
			},
			expectedStatus:   http.StatusSeeOther,
			expectedLocation: "/swims?direction=desc&sort=date",
		},
		{
			name:   "zero distance",
			swimID: "5",
			form: url.Values{
				"date":       []string{"2024-02-01"},
				"distance_m": []string{"0"},
				"assessment": []string{"2"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "negative distance",
			swimID: "5",
			form: url.Values{
				"date":       []string{"2024-02-01"},
				"distance_m": []string{"-100"},
				"assessment": []string{"2"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "negative assessment",
			swimID: "5",
			form: url.Values{
				"date":       []string{"2024-02-01"},
				"distance_m": []string{"2000"},
				"assessment": []string{"-1"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "assessment too large",
			swimID: "5",
			form: url.Values{
				"date":       []string{"2024-02-01"},
				"distance_m": []string{"2000"},
				"assessment": []string{"3"},
			},
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()

			mockSwims := &testutils.MockSwimModel{}
			tt.setupMock(mockSwims)
			app.swims = mockSwims

			rr := httptest.NewRecorder()
			form := tt.form.Encode()
			r := httptest.NewRequest(http.MethodPost, "/swims/edit/"+tt.swimID, strings.NewReader(form))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			ctx, _ := app.sessionManager.Load(r.Context(), "")
			app.sessionManager.Put(ctx, "authenticatedUserID", 1)
			ctx = context.WithValue(ctx, httprouter.ParamsKey, httprouter.Params{{Key: "id", Value: tt.swimID}})
			r = r.WithContext(ctx)

			app.updateSwim(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.expectedLocation != "" {
				assert.Equal(t, tt.expectedLocation, rr.Header().Get("Location"))
			}
		})
	}
}

func TestDeleteSwim(t *testing.T) {
	tests := []struct {
		name             string
		swimID           string
		setupMock        func(*testutils.MockSwimModel)
		expectedStatus   int
		expectedLocation string
		expectFlash      bool
	}{
		{
			name:   "successful delete",
			swimID: "5",
			setupMock: func(m *testutils.MockSwimModel) {
				m.DeleteFunc = func(id int, userId int) error {
					assert.Equal(t, 5, id)
					assert.Equal(t, 1, userId)
					return nil
				}
			},
			expectedStatus:   http.StatusSeeOther,
			expectedLocation: "/swims?direction=desc&sort=date",
			expectFlash:      true,
		},
		{
			name:   "swim not found",
			swimID: "999",
			setupMock: func(m *testutils.MockSwimModel) {
				m.DeleteFunc = func(id int, userId int) error {
					return models.ErrNoRecord
				}
			},
			expectedStatus: http.StatusNotFound,
			expectFlash:    false,
		},
		{
			name:   "swim belongs to different user",
			swimID: "5",
			setupMock: func(m *testutils.MockSwimModel) {
				m.DeleteFunc = func(id int, userId int) error {
					return models.ErrNoRecord
				}
			},
			expectedStatus: http.StatusNotFound,
			expectFlash:    false,
		},
		{
			name:   "database error on delete",
			swimID: "5",
			setupMock: func(m *testutils.MockSwimModel) {
				m.DeleteFunc = func(id int, userId int) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectFlash:    false,
		},
		{
			name:           "invalid swim ID",
			swimID:         "abc",
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusNotFound,
			expectFlash:    false,
		},
		{
			name:           "negative swim ID",
			swimID:         "-5",
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusNotFound,
			expectFlash:    false,
		},
		{
			name:           "zero swim ID",
			swimID:         "0",
			setupMock:      func(m *testutils.MockSwimModel) {},
			expectedStatus: http.StatusNotFound,
			expectFlash:    false,
		},
	}

	// Test with query parameters
	t.Run("successful delete with sort parameters", func(t *testing.T) {
		app := newTestApplication()

		mockSwims := &testutils.MockSwimModel{}
		mockSwims.DeleteFunc = func(id int, userId int) error {
			return nil
		}
		app.swims = mockSwims

		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/swims/5?sort=distance&direction=asc", nil)

		ctx, _ := app.sessionManager.Load(r.Context(), "")
		app.sessionManager.Put(ctx, "authenticatedUserID", 1)
		ctx = context.WithValue(ctx, httprouter.ParamsKey, httprouter.Params{{Key: "id", Value: "5"}})
		r = r.WithContext(ctx)

		app.deleteSwim(rr, r)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/swims?direction=asc&sort=distance", rr.Header().Get("Location"))
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()

			mockSwims := &testutils.MockSwimModel{}
			tt.setupMock(mockSwims)
			app.swims = mockSwims

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodDelete, "/swims/"+tt.swimID, nil)

			ctx, _ := app.sessionManager.Load(r.Context(), "")
			app.sessionManager.Put(ctx, "authenticatedUserID", 1)
			ctx = context.WithValue(ctx, httprouter.ParamsKey, httprouter.Params{{Key: "id", Value: tt.swimID}})
			r = r.WithContext(ctx)

			app.deleteSwim(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedLocation != "" {
				assert.Equal(t, tt.expectedLocation, rr.Header().Get("Location"))
			}

			if tt.expectFlash {
				flashText := app.sessionManager.GetString(ctx, "flashText")
				assert.NotEmpty(t, flashText)
				assert.Contains(t, flashText, "deleted")
			}
		})
	}
}
