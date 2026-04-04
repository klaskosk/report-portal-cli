package client_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient creates an httptest.Server backed by the given handler, then returns a Client
// pointed at that server. The server is automatically closed when the test finishes.
func newTestClient(t *testing.T, handler http.Handler) *client.Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := client.New(srv.URL, "test-token")
	require.NoError(t, err, "failed to create test client")

	return c
}

// jsonResponse returns an http.HandlerFunc that responds with the given status code and JSON body.
func jsonResponse(statusCode int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		_, _ = w.Write([]byte(body))
	}
}

// assertBearerToken checks that the request has the expected bearer token.
func assertBearerToken(t *testing.T, r *http.Request, expectedToken string) {
	t.Helper()

	assert.Equal(
		t,
		"Bearer "+expectedToken,
		r.Header.Get("Authorization"),
		"Authorization header mismatch",
	)
}

// assertPath checks that the request path matches the expected path.
func assertPath(t *testing.T, r *http.Request, expectedPath string) {
	t.Helper()

	assert.Equal(t, expectedPath, r.URL.Path, "request path mismatch")
}

// assertMethod checks that the request method matches the expected method.
func assertMethod(t *testing.T, r *http.Request, expectedMethod string) {
	t.Helper()

	assert.Equal(t, expectedMethod, r.Method, "request method mismatch")
}

// assertHeader checks that the request header matches the expected value.
func assertHeader(t *testing.T, r *http.Request, key, expectedValue string) {
	t.Helper()

	assert.Equal(t, expectedValue, r.Header.Get(key), "request header mismatch for %s", key)
}
