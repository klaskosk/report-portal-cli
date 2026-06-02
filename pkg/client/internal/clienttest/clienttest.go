// Package clienttest provides helpers for unit tests of pkg/client and its subpackages.
package clienttest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBearerToken = "test-token"

// NewClient creates a Client pointed at an httptest server using handler.
func NewClient(t *testing.T, handler http.Handler) *client.Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := client.New(srv.URL, testBearerToken)
	require.NoError(t, err)

	return c
}

// JSONHandler returns a handler that serves a fixed JSON response.
func JSONHandler(statusCode int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}
}

// RunBulkInfoUpdateBadRequestUnauthorizedErrorTests runs shared bulk-update error tests.
func RunBulkInfoUpdateBadRequestUnauthorizedErrorTests(
	t *testing.T,
	request types.BulkInfoUpdateRQ,
	call func(*client.Client, context.Context, types.BulkInfoUpdateRQ) error,
) {
	t.Helper()

	cases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "BadRequest",
			statusCode: http.StatusBadRequest,
			body:       `{"errorCode": 4001, "message": "Invalid bulk update payload."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := NewClient(t, JSONHandler(testCase.statusCode, testCase.body))
			err := call(testClient, t.Context(), request)
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}
