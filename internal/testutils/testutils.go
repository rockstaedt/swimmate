package testutils

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alexedwards/scs/v2"
)

// NewTestLogger returns a no-op logger for testing
func NewTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Only show errors in tests
	}))
}

// NewTestSessionManager returns a session manager configured for testing
func NewTestSessionManager() *scs.SessionManager {
	sessionManager := scs.New()
	sessionManager.Cookie.Secure = false // Allow for testing
	return sessionManager
}

// NewTestRequest creates a new HTTP request for testing
func NewTestRequest(t *testing.T, method, path string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	return req
}

// NewTestResponseRecorder creates a new ResponseRecorder for testing
func NewTestResponseRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}
