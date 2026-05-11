package client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient creates an httptest.Server backed by the given handler, then returns a Client
// pointed at that server. The server is automatically closed when the test finishes.
func newTestClient(t *testing.T, handler http.Handler) *client.Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := client.New(srv.URL, testBearerToken)
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

const testBearerToken = "test-token"

// assertBearerToken checks that the request uses the test bearer token.
func assertBearerToken(t *testing.T, req *http.Request) {
	t.Helper()

	assert.Equal(
		t,
		"Bearer "+testBearerToken,
		req.Header.Get("Authorization"),
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

// decodeRequestJSONMap reads and unmarshals the request body as a JSON object.
// It uses assert (not require) so it is safe to call from http.HandlerFunc handlers.
func decodeRequestJSONMap(t *testing.T, r *http.Request) (map[string]any, bool) {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	if !assert.NoError(t, err) {
		return nil, false
	}

	var payload map[string]any
	if !assert.NoError(t, json.Unmarshal(body, &payload)) {
		return nil, false
	}

	return payload, true
}

func bulkInfoUpdateBadRequestUnauthorizedCases() []struct {
	name       string
	statusCode int
	body       string
	check      func(error) bool
} {
	return []struct {
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
}

func runBulkInfoUpdateBadRequestUnauthorizedErrorTests(
	t *testing.T,
	request types.BulkInfoUpdateRQ,
	call func(*client.Client, context.Context, types.BulkInfoUpdateRQ) error,
) {
	t.Helper()

	for _, testCase := range bulkInfoUpdateBadRequestUnauthorizedCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			err := call(testClient, t.Context(), request)
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

const testItemListSearchTwoEntryPageJSON = `{
			"content": [
				{"id": 101, "uuid": "suite-101", "name": "Suite A", "status": "PASSED", "type": "SUITE"},
				{"id": 102, "uuid": "test-102", "name": "Test A", "status": "FAILED", "type": "TEST"}
			],
			"page": {"number": 0, "size": 10, "totalElements": 2, "totalPages": 1}
		}`

func newTestClientItemListOrSearchGET(t *testing.T, path string) *client.Client {
	t.Helper()

	return newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, path)
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(testItemListSearchTwoEntryPageJSON))
		},
	))
}

func newTestClientBulkInfoPUTSuccess(
	t *testing.T,
	path string,
	responseJSON string,
) *client.Client {
	t.Helper()

	return newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, path)
			assertMethod(t, httpReq, http.MethodPut)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			ids, ok := payload["ids"].([]any)
			if !assert.True(t, ok) {
				return
			}

			assert.Len(t, ids, 2)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(responseJSON))
		},
	))
}

func newTestClientItemAttributeKeysGET(
	t *testing.T,
	path string,
	query map[string]string,
	responseJSON string,
) *client.Client {
	t.Helper()

	return newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, path)
			assertMethod(t, httpReq, http.MethodGet)

			q := httpReq.URL.Query()
			for key, want := range query {
				assert.Equal(t, want, q.Get(key))
			}

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(responseJSON))
		},
	))
}

func newTestClientLaunchStartPOSTSuccess(
	t *testing.T,
	path, expectLaunchName string,
	startTime time.Time,
	mode types.StartLaunchRQMode,
	responseJSON string,
) *client.Client {
	t.Helper()

	return newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, path)
			assertMethod(t, httpReq, http.MethodPost)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, expectLaunchName, payload["name"])
			assert.Equal(t, startTime.UTC().Format(time.RFC3339Nano), payload["startTime"])
			assert.Equal(t, string(mode), payload["mode"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(responseJSON))
		},
	))
}

func newTestClientDescriptionPUTUpdateSuccess(
	t *testing.T,
	path string,
	expectedDescription string,
	responseJSON string,
) *client.Client {
	t.Helper()

	return newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, path)
			assertMethod(t, httpReq, http.MethodPut)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, expectedDescription, payload["description"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(responseJSON))
		},
	))
}
