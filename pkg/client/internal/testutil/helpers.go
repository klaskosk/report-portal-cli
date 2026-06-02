package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
)

// DecodeRequestJSONMap reads and unmarshals the request body as a JSON object.
func DecodeRequestJSONMap(t *testing.T, r *http.Request) (map[string]any, bool) {
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

// TestItemListSearchTwoEntryPageJSON is a fixture for list/search item tests.
const TestItemListSearchTwoEntryPageJSON = `{
			"content": [
				{"id": 101, "uuid": "suite-101", "name": "Suite A", "status": "PASSED", "type": "SUITE"},
				{"id": 102, "uuid": "test-102", "name": "Test A", "status": "FAILED", "type": "TEST"}
			],
			"page": {"number": 0, "size": 10, "totalElements": 2, "totalPages": 1}
		}`

// ItemListOrSearchGETHandler returns a handler for item list/search GET endpoints.
func ItemListOrSearchGETHandler(t *testing.T, path string) http.HandlerFunc {
	t.Helper()

	return func(responseWriter http.ResponseWriter, httpReq *http.Request) {
		AssertBearerToken(t, httpReq)
		AssertPath(t, httpReq, path)
		AssertMethod(t, httpReq, http.MethodGet)

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusOK)
		_, _ = responseWriter.Write([]byte(TestItemListSearchTwoEntryPageJSON))
	}
}

// BulkInfoPUTSuccessHandler returns a handler for bulk info PUT endpoints.
func BulkInfoPUTSuccessHandler(t *testing.T, path, responseJSON string) http.HandlerFunc {
	t.Helper()

	return func(responseWriter http.ResponseWriter, httpReq *http.Request) {
		AssertBearerToken(t, httpReq)
		AssertPath(t, httpReq, path)
		AssertMethod(t, httpReq, http.MethodPut)

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
	}
}

// ItemAttributeKeysGETHandler returns a handler for attribute key GET endpoints.
func ItemAttributeKeysGETHandler(
	t *testing.T,
	path string,
	query map[string]string,
	responseJSON string,
) http.HandlerFunc {
	t.Helper()

	return func(responseWriter http.ResponseWriter, httpReq *http.Request) {
		AssertBearerToken(t, httpReq)
		AssertPath(t, httpReq, path)
		AssertMethod(t, httpReq, http.MethodGet)

		q := httpReq.URL.Query()
		for key, want := range query {
			assert.Equal(t, want, q.Get(key))
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusOK)
		_, _ = responseWriter.Write([]byte(responseJSON))
	}
}

// LaunchStartPOSTSuccessHandler returns a handler for launch start POST endpoints.
func LaunchStartPOSTSuccessHandler(
	t *testing.T,
	path, expectLaunchName string,
	startTime time.Time,
	mode types.StartLaunchRQMode,
	responseJSON string,
) http.HandlerFunc {
	t.Helper()

	return func(responseWriter http.ResponseWriter, httpReq *http.Request) {
		AssertBearerToken(t, httpReq)
		AssertPath(t, httpReq, path)
		AssertMethod(t, httpReq, http.MethodPost)

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
	}
}

// DescriptionPUTUpdateSuccessHandler returns a handler for description PUT endpoints.
func DescriptionPUTUpdateSuccessHandler(
	t *testing.T,
	path string,
	expectedDescription string,
	responseJSON string,
) http.HandlerFunc {
	t.Helper()

	return func(responseWriter http.ResponseWriter, httpReq *http.Request) {
		AssertBearerToken(t, httpReq)
		AssertPath(t, httpReq, path)
		AssertMethod(t, httpReq, http.MethodPut)

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
	}
}
