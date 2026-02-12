package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogScope_GetByID_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/log/42")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"uuid": "log-uuid-42",
			"message": "Test step completed",
			"level": "INFO",
			"itemId": 100,
			"launchId": 10
		}`))
	}))

	log, err := c.Project("my_project").Logs().GetByID(t.Context(), "42")
	require.NoError(t, err)
	require.NotNil(t, log)

	assert.Equal(t, int64(42), log.Id)
	assert.Equal(t, "log-uuid-42", log.Uuid)
	require.NotNil(t, log.Message)
	assert.Equal(t, "Test step completed", *log.Message)
}

func TestLogScope_GetByID_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		logID      string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			logID:      "999",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40440, "message": "Log '999' not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Unauthorized",
			logID:      "42",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))

			_, err := testClient.Project("my_project").Logs().GetByID(t.Context(), testCase.logID)
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLogScope_GetByUUID_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/log/uuid/log-uuid-42")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"uuid": "log-uuid-42",
			"message": "Test passed",
			"level": "INFO"
		}`))
	}))

	log, err := c.Project("my_project").Logs().GetByUUID(t.Context(), "log-uuid-42")
	require.NoError(t, err)
	require.NotNil(t, log)

	assert.Equal(t, "log-uuid-42", log.Uuid)
	require.NotNil(t, log.Message)
	assert.Equal(t, "Test passed", *log.Message)
}

func TestLogScope_GetByUUID_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40440, "message": "Log not found."}`))

	_, err := c.Project("my_project").Logs().GetByUUID(t.Context(), "unknown-uuid")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestLogScope_List_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/log")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": 1, "uuid": "log-1", "message": "Step 1", "level": "INFO", "itemId": 100},
				{"id": 2, "uuid": "log-2", "message": "Step 2 failed", "level": "ERROR", "itemId": 100}
			],
			"page": {"number": 0, "size": 10, "totalElements": 2, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Logs().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 2)

	require.NotNil(t, result.Content[0].Message)
	assert.Equal(t, "Step 1", *result.Content[0].Message)
	assert.Equal(t, 2, result.Page.TotalElements)
}

func TestLogScope_List_WithFilterOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/v1/my_project/log")

		query := r.URL.Query()
		assert.Equal(t, "100", query.Get("filter.eq.item"))
		assert.Equal(t, "10", query.Get("filter.eq.launchId"))
		assert.Equal(t, "ERROR", query.Get("filter.eq.level"))
		assert.Equal(t, "2", query.Get("page.page"))
		assert.Equal(t, "5", query.Get("page.size"))
		assert.Equal(t, "logTime,desc", query.Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 2, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Project("my_project").Logs().List(
		t.Context(),
		client.WithLogItemID(100),
		client.WithLogLaunchID(10),
		client.WithLogLevel("ERROR"),
		client.WithLogPageNumber(2),
		client.WithLogPageSize(5),
		client.WithLogSort("logTime,desc"),
	)
	require.NoError(t, err)
}

func TestLogScope_List_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Logs().List(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestLogScope_List_EmptyResults(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	result, err := c.Project("my_project").Logs().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Content)
}

func TestLogScope_Create_Success(t *testing.T) {
	t.Parallel()

	message := "Test log message"
	level := types.SaveLogRQLevel("INFO")
	itemUUID := "item-uuid-1"

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/log/entry")
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "launch-uuid-1", payload["launchUuid"])
		assert.Equal(t, "item-uuid-1", payload["itemUuid"])
		assert.Equal(t, "Test log message", payload["message"])
		assert.Equal(t, "INFO", payload["level"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"log-entry-uuid-1"}`))
	}))

	result, err := c.Project("my_project").Logs().Create(
		t.Context(),
		types.SaveLogRQ{
			LaunchUuid: "launch-uuid-1",
			ItemUuid:   &itemUUID,
			Message:    &message,
			Level:      &level,
			Time:       types.EpochMillis(time.UnixMilli(1700000000000)),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, "log-entry-uuid-1", *result.Id)
}

func TestLogScope_Create_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "BadRequest",
			statusCode: http.StatusBadRequest,
			body:       `{"errorCode": 4001, "message": "Missing required fields."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Logs().Create(
				t.Context(),
				types.SaveLogRQ{
					LaunchUuid: "launch-uuid-1",
					Time:       types.EpochMillis(time.Now()),
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLogScope_Delete_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/log/42")
		assertMethod(t, r, http.MethodDelete)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Log entry successfully deleted."}`))
	}))

	result, err := c.Project("my_project").Logs().Delete(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Log entry successfully deleted.", *result.Message)
}

func TestLogScope_Delete_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40440, "message": "Log '999' not found."}`,
			check:      client.IsNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Logs().Delete(t.Context(), 999)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}
