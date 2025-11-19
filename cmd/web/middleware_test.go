package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rockstaedt/swimmate/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestSecureHeaders(t *testing.T) {
	tests := []struct {
		name            string
		expectedHeaders map[string]string
	}{
		{
			name: "all security headers set",
			expectedHeaders: map[string]string{
				"Referrer-Policy":        "origin-when-cross-origin",
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "deny",
				"X-XSS-Protection":       "0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that will be wrapped
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			})

			// Wrap with secureHeaders middleware
			handler := secureHeaders(nextHandler)

			// Create test request and response recorder
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Execute the handler
			handler.ServeHTTP(rr, r)

			// Verify all expected headers are set
			for header, expected := range tt.expectedHeaders {
				actual := rr.Header().Get(header)
				assert.Equal(t, expected, actual, "Header %s should be %s", header, expected)
			}

			// Verify the next handler was called
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "OK", rr.Body.String())
		})
	}
}

func TestLogRequest(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		uri        string
		remoteAddr string
		proto      string
	}{
		{
			name:       "GET request",
			method:     http.MethodGet,
			uri:        "/test",
			remoteAddr: "192.168.1.1:1234",
			proto:      "HTTP/1.1",
		},
		{
			name:       "POST request",
			method:     http.MethodPost,
			uri:        "/api/data",
			remoteAddr: "10.0.0.1:5678",
			proto:      "HTTP/2.0",
		},
		{
			name:       "request with query params",
			method:     http.MethodGet,
			uri:        "/search?q=test&page=1",
			remoteAddr: "127.0.0.1:9999",
			proto:      "HTTP/1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &application{
				logger: testutils.NewTestLogger(),
			}

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := app.logRequest(nextHandler)

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(tt.method, tt.uri, nil)
			r.RemoteAddr = tt.remoteAddr
			r.Proto = tt.proto

			handler.ServeHTTP(rr, r)

			// Verify the next handler was called
			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestRecoverPanic(t *testing.T) {
	tests := []struct {
		name           string
		handlerFunc    http.HandlerFunc
		expectedStatus int
		shouldPanic    bool
		expectedHeader string
	}{
		{
			name: "no panic - normal execution",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			},
			expectedStatus: http.StatusOK,
			shouldPanic:    false,
		},
		{
			name: "panic with string",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				panic("something went wrong")
			},
			expectedStatus: http.StatusInternalServerError,
			shouldPanic:    true,
			expectedHeader: "close",
		},
		{
			name: "panic with error",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				panic("database connection failed")
			},
			expectedStatus: http.StatusInternalServerError,
			shouldPanic:    true,
			expectedHeader: "close",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &application{
				logger: testutils.NewTestLogger(),
			}

			handler := app.recoverPanic(tt.handlerFunc)

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Execute the handler (should not panic even if inner handler panics)
			handler.ServeHTTP(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.shouldPanic {
				// Verify Connection: close header is set on panic
				assert.Equal(t, tt.expectedHeader, rr.Header().Get("Connection"))
				// Verify error response
				assert.Contains(t, rr.Body.String(), "Internal Server Error")
			}
		})
	}
}

func TestRequireAuthentication(t *testing.T) {
	tests := []struct {
		name                 string
		setupRequest         func() *http.Request
		isAuthenticated      bool
		expectedStatus       int
		expectedLocation     string
		shouldCallNext       bool
		shouldSetCacheHeader bool
	}{
		{
			name: "authenticated user - calls next handler",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/protected", nil)
			},
			isAuthenticated:      true,
			expectedStatus:       http.StatusOK,
			shouldCallNext:       true,
			shouldSetCacheHeader: true,
		},
		{
			name: "unauthenticated user - redirects to login",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/protected", nil)
			},
			isAuthenticated:      false,
			expectedStatus:       http.StatusSeeOther,
			expectedLocation:     "/login",
			shouldCallNext:       false,
			shouldSetCacheHeader: false,
		},
		{
			name: "authenticated POST request",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/protected/action", nil)
			},
			isAuthenticated:      true,
			expectedStatus:       http.StatusOK,
			shouldCallNext:       true,
			shouldSetCacheHeader: true,
		},
		{
			name: "unauthenticated POST request redirects",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/protected/action", nil)
			},
			isAuthenticated:      false,
			expectedStatus:       http.StatusSeeOther,
			expectedLocation:     "/login",
			shouldCallNext:       false,
			shouldSetCacheHeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionManager := testutils.NewTestSessionManager()
			app := &application{
				sessionManager: sessionManager,
			}

			nextHandlerCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextHandlerCalled = true
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("Protected content"))
			})

			handler := app.requireAuthentication(nextHandler)

			r := tt.setupRequest()

			// Set up session context
			ctx, _ := sessionManager.Load(r.Context(), "")
			if tt.isAuthenticated {
				sessionManager.Put(ctx, "authenticatedUserID", 1)
			}
			r = r.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, r)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedLocation != "" {
				assert.Equal(t, tt.expectedLocation, rr.Header().Get("Location"))
			}

			assert.Equal(t, tt.shouldCallNext, nextHandlerCalled)

			if tt.shouldSetCacheHeader {
				assert.Equal(t, "no-store", rr.Header().Get("Cache-Control"))
			}

			if tt.shouldCallNext {
				assert.Contains(t, rr.Body.String(), "Protected content")
			}
		})
	}
}

func TestMiddlewareChaining(t *testing.T) {
	// Test that middleware can be chained together
	app := &application{
		logger:         testutils.NewTestLogger(),
		sessionManager: testutils.NewTestSessionManager(),
	}

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Final handler"))
	})

	// Chain all middleware together
	handler := secureHeaders(app.logRequest(app.recoverPanic(finalHandler)))

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler.ServeHTTP(rr, r)

	// Verify all middleware executed
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "Final handler", rr.Body.String())

	// Verify security headers were set
	assert.Equal(t, "deny", rr.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
}

func TestRecoverPanicWithConnectionClose(t *testing.T) {
	app := &application{
		logger: testutils.NewTestLogger(),
	}

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("critical error")
	})

	handler := app.recoverPanic(panicHandler)

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler.ServeHTTP(rr, r)

	// Verify Connection: close header is set
	assert.Equal(t, "close", rr.Header().Get("Connection"))
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
