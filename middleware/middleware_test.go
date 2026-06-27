package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gapi-org/gapi/middleware"
)

func TestRequestIDUsesIncomingHeader(t *testing.T) {
	handler := middleware.RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := middleware.RequestIDFromContext(r.Context()); got != "request-123" {
			t.Fatalf("expected context request ID, got %q", got)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "request-123")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if got := res.Header().Get("X-Request-ID"); got != "request-123" {
		t.Fatalf("expected response request ID header, got %q", got)
	}
}

func TestRecoverWritesProblemJSON(t *testing.T) {
	handler := middleware.Recover()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", res.Code)
	}
	if got := res.Header().Get("Content-Type"); got != "application/problem+json; charset=utf-8" {
		t.Fatalf("expected problem JSON content type, got %q", got)
	}
}

func TestSecureHeaders(t *testing.T) {
	handler := middleware.SecureHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if got := res.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}
	if got := res.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected frame options header, got %q", got)
	}
}

func TestCORSPreflight(t *testing.T) {
	handler := middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("preflight should not call next handler")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", res.Code)
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("expected CORS origin header, got %q", got)
	}
}

func TestAPIKeyHeaderRejectsMissingKey(t *testing.T) {
	handler := middleware.APIKeyHeader("X-API-Key", func(value string) bool {
		return value == "secret"
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unauthorized request should not call next handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", res.Code)
	}
	if got := res.Header().Get("Content-Type"); got != "application/problem+json; charset=utf-8" {
		t.Fatalf("expected problem JSON content type, got %q", got)
	}
}

func TestBearerTokenAllowsValidToken(t *testing.T) {
	handler := middleware.BearerToken(func(value string) bool {
		return value == "secret"
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", res.Code)
	}
}

func TestBearerTokenRejectsEmptyTokenBeforeAllowFunc(t *testing.T) {
	called := false
	handler := middleware.BearerToken(func(value string) bool {
		called = true
		return true
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("empty bearer token should not call next handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", res.Code)
	}
	if called {
		t.Fatal("allow function should not be called for an empty bearer token")
	}
}
