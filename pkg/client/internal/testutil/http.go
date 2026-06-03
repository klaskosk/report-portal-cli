package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const TestBearerToken = "test-token"

// NewTestServer creates an httptest server that responds with the given status and JSON body.
func NewTestServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(JSONHandler(status, body))
	t.Cleanup(server.Close)

	return server
}

// JSONHandler returns a handler that serves a fixed JSON response.
func JSONHandler(statusCode int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}
}

// AssertBearerToken checks that the request uses the test bearer token.
func AssertBearerToken(t *testing.T, req *http.Request) {
	t.Helper()

	if req.Header.Get("Authorization") != "Bearer "+TestBearerToken {
		t.Errorf("Authorization header mismatch: got %q", req.Header.Get("Authorization"))
	}
}

// AssertPath checks that the request path matches the expected path.
func AssertPath(t *testing.T, r *http.Request, expectedPath string) {
	t.Helper()

	if r.URL.Path != expectedPath {
		t.Errorf("request path mismatch: got %q want %q", r.URL.Path, expectedPath)
	}
}

// AssertMethod checks that the request method matches the expected method.
func AssertMethod(t *testing.T, r *http.Request, expectedMethod string) {
	t.Helper()

	if r.Method != expectedMethod {
		t.Errorf("request method mismatch: got %q want %q", r.Method, expectedMethod)
	}
}

// AssertHeader checks that the request header matches the expected value.
func AssertHeader(t *testing.T, r *http.Request, key, expectedValue string) {
	t.Helper()

	if got := r.Header.Get(key); got != expectedValue {
		t.Errorf("request header mismatch for %s: got %q want %q", key, got, expectedValue)
	}
}
