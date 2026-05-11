package client_test

import (
	"context"
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

func assertCreateBatchMultipartRequest(
	t *testing.T,
	responseWriter http.ResponseWriter,
	httpReq *http.Request,
	logPath string,
	expectedMessage string,
) bool {
	t.Helper()

	assertBearerToken(t, httpReq)
	assertPath(t, httpReq, logPath)
	assertMethod(t, httpReq, http.MethodPost)

	contentType := httpReq.Header.Get("Content-Type")
	assert.Contains(t, contentType, "multipart/form-data")
	assert.Contains(t, contentType, "boundary=")

	httpReq.Body = http.MaxBytesReader(responseWriter, httpReq.Body, 10<<20)

	err := httpReq.ParseMultipartForm(10 << 20)
	//nolint:testifylint // assert keeps this helper usable from HTTP handlers
	if !assert.NoError(t, err) {
		return false
	}

	jsonPart := httpReq.FormValue("json_request_part")
	if !assert.NotEmpty(t, jsonPart) {
		return false
	}

	var entries []map[string]any
	//nolint:testifylint // assert keeps this helper usable from HTTP handlers
	if !assert.NoError(t, json.Unmarshal([]byte(jsonPart), &entries)) {
		return false
	}

	if !assert.Len(t, entries, 1) {
		return false
	}

	assert.Equal(t, "launch-uuid-1", entries[0]["launchUuid"])
	assert.Equal(t, "item-uuid-1", entries[0]["itemUuid"])
	assert.Equal(t, expectedMessage, entries[0]["message"])

	return true
}

func createBatchAttachmentMultipartHandler(
	t *testing.T,
	logPath string,
	responseJSON string,
) http.HandlerFunc {
	t.Helper()

	return func(responseWriter http.ResponseWriter, httpReq *http.Request) {
		assertBearerToken(t, httpReq)
		assertPath(t, httpReq, logPath)
		assertMethod(t, httpReq, http.MethodPost)

		httpReq.Body = http.MaxBytesReader(responseWriter, httpReq.Body, 10<<20)

		err := httpReq.ParseMultipartForm(10 << 20)
		if !assert.NoError(t, err) {
			return
		}

		jsonPart := httpReq.FormValue("json_request_part")
		if !assert.NotEmpty(t, jsonPart) {
			return
		}

		var entries []map[string]any
		if !assert.NoError(t, json.Unmarshal([]byte(jsonPart), &entries)) {
			return
		}

		if !assert.Len(t, entries, 1) {
			return
		}

		file, ok := entries[0]["file"].(map[string]any)
		if !assert.True(t, ok, "entry should have a file field") {
			return
		}

		assert.Equal(t, "test.txt", file["name"])

		_, header, err := httpReq.FormFile("file")
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "test.txt", header.Filename)

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusOK)
		_, _ = responseWriter.Write([]byte(responseJSON))
	}
}

func logEntryCreateSuccessHandler(
	t *testing.T,
	logPath string,
	wantMessage string,
	responseBody string,
) http.HandlerFunc {
	t.Helper()

	return func(responseWriter http.ResponseWriter, httpReq *http.Request) {
		assertBearerToken(t, httpReq)
		assertPath(t, httpReq, logPath)
		assertMethod(t, httpReq, http.MethodPost)

		body, err := io.ReadAll(httpReq.Body)
		if !assert.NoError(t, err) {
			return
		}

		var payload map[string]any
		if !assert.NoError(t, json.Unmarshal(body, &payload)) {
			return
		}

		assert.Equal(t, "launch-uuid-1", payload["launchUuid"])
		assert.Equal(t, "item-uuid-1", payload["itemUuid"])
		assert.Equal(t, wantMessage, payload["message"])
		assert.Equal(t, "INFO", payload["level"])

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusCreated)
		_, _ = responseWriter.Write([]byte(responseBody))
	}
}

func TestLogScope_GetByID_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log/42")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 42,
			"uuid": "log-uuid-42",
			"message": "Test step completed",
			"level": "INFO",
			"itemId": 100,
			"launchId": 10
		}`))
		},
	))

	log, err := testClient.Project("my_project").Logs().GetByID(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, log)

	assert.Equal(t, int64(42), log.Id)
	assert.Equal(t, "log-uuid-42", log.Uuid)
	require.NotNil(t, log.Message)
	assert.Equal(t, "Test step completed", *log.Message)
}

//nolint:dupl // structured like sibling tests; shared HTTP/table scaffolding
func TestLogScope_GetByID_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		logID      int64
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			logID:      999,
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40440, "message": "Log '999' not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Unauthorized",
			logID:      42,
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log/uuid/log-uuid-42")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 42,
			"uuid": "log-uuid-42",
			"message": "Test passed",
			"level": "INFO"
		}`))
		},
	))

	log, err := testClient.Project("my_project").Logs().GetByUUID(t.Context(), "log-uuid-42")
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [
				{"id": 1, "uuid": "log-1", "message": "Step 1", "level": "INFO", "itemId": 100},
				{"id": 2, "uuid": "log-2", "message": "Step 2 failed", "level": "ERROR", "itemId": 100}
			],
			"page": {"number": 0, "size": 10, "totalElements": 2, "totalPages": 1}
		}`))
		},
	))

	result, err := testClient.Project("my_project").Logs().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, *result.Content, 2)

	require.NotNil(t, (*result.Content)[0].Message)
	assert.Equal(t, "Step 1", *(*result.Content)[0].Message)
	assert.EqualValues(t, 2, *result.Page.TotalElements)
}

//nolint:dupl // structured like sibling tests; shared HTTP/table scaffolding
func TestLogScope_List_WithFilterOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, r *http.Request) {
			assertPath(t, r, "/api/v1/my_project/log")

			query := r.URL.Query()
			assert.Equal(t, "100", query.Get("filter.eq.item"))
			assert.Equal(t, "10", query.Get("filter.eq.launchId"))
			assert.Equal(t, "ERROR", query.Get("filter.eq.level"))
			assert.Equal(t, "2", query.Get("page.page"))
			assert.Equal(t, "5", query.Get("page.size"))
			assert.Equal(t, "logTime,desc", query.Get("page.sort"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [],
			"page": {"number": 2, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
		},
	))

	_, err := testClient.Project("my_project").Logs().List(
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

	testClient := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	result, err := testClient.Project("my_project").Logs().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Content)
}

func TestLogScope_Create_Success(t *testing.T) {
	t.Parallel()

	message := "Test log message"
	level := "INFO"
	itemUUID := "item-uuid-1"

	testClient := newTestClient(t, logEntryCreateSuccessHandler(t,
		"/api/v1/my_project/log/entry",
		"Test log message",
		`{"id":"log-entry-uuid-1"}`,
	))

	result, err := testClient.Project("my_project").Logs().Create(
		t.Context(),
		types.SaveLogRQ{
			LaunchUuid: "launch-uuid-1",
			ItemUuid:   &itemUUID,
			Message:    &message,
			Level:      &level,
			Time:       time.UnixMilli(1700000000000).UTC(),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, "log-entry-uuid-1", *result.Id)
}

//nolint:dupl // structured like sibling tests; shared HTTP/table scaffolding
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
					Time:       time.Now(),
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLogScope_CreateAsync_Success(t *testing.T) {
	t.Parallel()

	message := "Async log message"
	level := "INFO"
	itemUUID := "item-uuid-1"

	testClient := newTestClient(t, logEntryCreateSuccessHandler(t,
		"/api/v2/my_project/log/entry",
		"Async log message",
		`{"id":"async-log-entry-uuid"}`,
	))

	result, err := testClient.Project("my_project").Logs().CreateAsync(
		t.Context(),
		types.SaveLogRQ{
			LaunchUuid: "launch-uuid-1",
			ItemUuid:   &itemUUID,
			Message:    &message,
			Level:      &level,
			Time:       time.UnixMilli(1700000000000).UTC(),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, "async-log-entry-uuid", *result.Id)
}

//nolint:dupl // structured like sibling tests; shared HTTP/table scaffolding
func TestLogScope_CreateAsync_Errors(t *testing.T) {
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
			_, err := testClient.Project("my_project").Logs().CreateAsync(
				t.Context(),
				types.SaveLogRQ{
					LaunchUuid: "launch-uuid-1",
					Time:       time.Now(),
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLogScope_CreateBatchMultipartJSONSuccess(t *testing.T) {
	t.Parallel()

	entryTime := time.UnixMilli(1700000000000).UTC()
	baseEntry := client.BatchLogEntry{
		LaunchUUID: "launch-uuid-1",
		ItemUUID:   "item-uuid-1",
		Level:      "INFO",
		Time:       entryTime,
	}

	tests := []struct {
		name         string
		logPath      string
		message      string
		responseBody string
		wantID       string
		call         func(context.Context, *client.Client, string) (*types.BatchSaveOperatingRS, error)
	}{
		{
			name:         "CreateBatch",
			logPath:      "/api/v1/my_project/log",
			message:      "Test log message",
			responseBody: `{"responses":[{"id":"log-batch-1"}]}`,
			wantID:       "log-batch-1",
			call: func(ctx context.Context, c *client.Client, msg string) (*types.BatchSaveOperatingRS, error) {
				return c.Project("my_project").Logs().CreateBatch(ctx, client.BatchLogEntry{
					LaunchUUID: baseEntry.LaunchUUID,
					ItemUUID:   baseEntry.ItemUUID,
					Level:      baseEntry.Level,
					Message:    msg,
					Time:       baseEntry.Time,
				})
			},
		},
		{
			name:         "CreateBatchAsync",
			logPath:      "/api/v2/my_project/log",
			message:      "Async batch log",
			responseBody: `{"responses":[{"id":"async-log-batch-1"}]}`,
			wantID:       "async-log-batch-1",
			call: func(ctx context.Context, c *client.Client, msg string) (*types.BatchSaveOperatingRS, error) {
				return c.Project("my_project").Logs().CreateBatchAsync(ctx, client.BatchLogEntry{
					LaunchUUID: baseEntry.LaunchUUID,
					ItemUUID:   baseEntry.ItemUUID,
					Level:      baseEntry.Level,
					Message:    msg,
					Time:       baseEntry.Time,
				})
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, http.HandlerFunc(
				func(responseWriter http.ResponseWriter, httpReq *http.Request) {
					if !assertCreateBatchMultipartRequest(
						t, responseWriter, httpReq, testCase.logPath, testCase.message,
					) {
						return
					}

					responseWriter.Header().Set("Content-Type", "application/json")
					responseWriter.WriteHeader(http.StatusOK)
					_, _ = responseWriter.Write([]byte(testCase.responseBody))
				},
			))

			result, err := testCase.call(t.Context(), testClient, testCase.message)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.Responses)
			require.Len(t, *result.Responses, 1)
			assert.Equal(t, testCase.wantID, *(*result.Responses)[0].Id)
		})
	}
}

func TestLogScope_CreateBatchAsync_WithAttachment(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, createBatchAttachmentMultipartHandler(t,
		"/api/v2/my_project/log",
		`{"responses":[{"id":"async-log-batch-attachment"}]}`,
	))

	result, err := testClient.Project("my_project").Logs().CreateBatchAsync(
		t.Context(),
		client.BatchLogEntry{
			LaunchUUID:     "launch-uuid-1",
			ItemUUID:       "item-uuid-1",
			Level:          "INFO",
			Message:        "Async log with attachment",
			Time:           time.UnixMilli(1700000000000).UTC(),
			AttachmentName: "test.txt",
			AttachmentData: []byte("file content here"),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Responses)
	require.Len(t, *result.Responses, 1)
}

func TestLogScope_CreateBatchOperationErrors(t *testing.T) {
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
			body:       `{"errorCode": 4001, "message": "Invalid log entry."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	entry := client.BatchLogEntry{
		LaunchUUID: "launch-uuid-1",
		Level:      "INFO",
		Message:    "Test",
		Time:       time.Now(),
	}

	apis := []struct {
		name string
		call func(context.Context, *client.Client) error
	}{
		{
			name: "CreateBatch",
			call: func(ctx context.Context, c *client.Client) error {
				_, err := c.Project("my_project").Logs().CreateBatch(ctx, entry)

				return err //nolint:wrapcheck // test exercises raw API errors
			},
		},
		{
			name: "CreateBatchAsync",
			call: func(ctx context.Context, c *client.Client) error {
				_, err := c.Project("my_project").Logs().CreateBatchAsync(ctx, entry)

				return err //nolint:wrapcheck // test exercises raw API errors
			},
		},
	}

	for _, api := range apis {
		t.Run(api.name, func(t *testing.T) {
			t.Parallel()

			for _, testCase := range testCases {
				t.Run(testCase.name, func(t *testing.T) {
					t.Parallel()

					testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
					err := api.call(t.Context(), testClient)
					require.Error(t, err)
					assert.True(t, testCase.check(err))
				})
			}
		})
	}
}

func TestLogScope_Delete_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log/42")
			assertMethod(t, httpReq, http.MethodDelete)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Log entry successfully deleted."}`))
		},
	))

	result, err := testClient.Project("my_project").Logs().Delete(t.Context(), 42)
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

func TestLogScope_CreateBatch_WithAttachment(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, createBatchAttachmentMultipartHandler(t,
		"/api/v1/my_project/log",
		`{"responses":[{"id":"log-batch-attachment"}]}`,
	))

	result, err := testClient.Project("my_project").Logs().CreateBatch(
		t.Context(),
		client.BatchLogEntry{
			LaunchUUID:     "launch-uuid-1",
			ItemUUID:       "item-uuid-1",
			Level:          "INFO",
			Message:        "Log with attachment",
			Time:           time.UnixMilli(1700000000000).UTC(),
			AttachmentName: "test.txt",
			AttachmentData: []byte("file content here"),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Responses)
	require.Len(t, *result.Responses, 1)
}

func TestLogScope_Search_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log/search/100")
			assertMethod(t, httpReq, http.MethodPost)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var request types.SearchLogRq
			if !assert.NoError(t, json.Unmarshal(body, &request)) {
				return
			}

			assert.Equal(t, types.CurrentLaunch, request.SearchMode)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[
				{
					"itemId": 200,
					"itemName": "Similar Test",
					"launchId": 10,
					"status": "FAILED",
					"duration": 1.5
				}
			]`))
		},
	))

	result, err := testClient.Project("my_project").Logs().Search(
		t.Context(),
		100,
		types.SearchLogRq{},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].ItemName)
	assert.Equal(t, "Similar Test", *result[0].ItemName)
}

//nolint:dupl // structured like sibling tests; shared HTTP/table scaffolding
func TestLogScope_Search_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40440, "message": "Test item '100' not found."}`,
			check:      client.IsNotFound,
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
			_, err := testClient.Project("my_project").Logs().Search(
				t.Context(),
				100,
				types.SearchLogRq{},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLogScope_Search_PreservesProvidedSearchMode(t *testing.T) {
	t.Parallel()

	expectedMode := types.LaunchName
	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log/search/100")
			assertMethod(t, httpReq, http.MethodPost)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var request types.SearchLogRq
			if !assert.NoError(t, json.Unmarshal(body, &request)) {
				return
			}

			assert.Equal(t, expectedMode, request.SearchMode)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`[]`))
		},
	))

	result, err := testClient.Project("my_project").Logs().Search(
		t.Context(),
		100,
		types.SearchLogRq{SearchMode: expectedMode},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestLogScope_GetLogsUnder_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log/under")
			assertMethod(t, httpReq, http.MethodPost)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var request types.GetLogsUnderRq
			if !assert.NoError(t, json.Unmarshal(body, &request)) {
				return
			}

			assert.Equal(t, []int64{100, 200}, request.ItemIds)
			assert.Equal(t, "INFO", request.LogLevel)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"100": [
				{"id": 1, "uuid": "log-1", "message": "Step 1", "level": "INFO"},
				{"id": 2, "uuid": "log-2", "message": "Step 2", "level": "ERROR"}
			],
			"200": [
				{"id": 3, "uuid": "log-3", "message": "Step 3", "level": "INFO"}
			]
		}`))
		},
	))

	itemIDs := []int64{100, 200}
	result, err := testClient.Project("my_project").Logs().GetLogsUnder(
		t.Context(),
		types.GetLogsUnderRq{ItemIds: itemIDs},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result["100"], 2)
	assert.Len(t, result["200"], 1)
}

func TestLogScope_GetLogsUnder_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "Forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"errorCode": 4003, "message": "Access denied."}`,
			check:      client.IsForbidden,
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
			itemIDs := []int64{100}
			_, err := testClient.Project("my_project").Logs().GetLogsUnder(
				t.Context(),
				types.GetLogsUnderRq{ItemIds: itemIDs},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLogScope_GetLogsUnder_PreservesProvidedLogLevel(t *testing.T) {
	t.Parallel()

	expectedLevel := "ERROR"
	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log/under")
			assertMethod(t, httpReq, http.MethodPost)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var request types.GetLogsUnderRq
			if !assert.NoError(t, json.Unmarshal(body, &request)) {
				return
			}

			assert.Equal(t, []int64{100}, request.ItemIds)
			assert.Equal(t, expectedLevel, request.LogLevel)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"100":[]}`))
		},
	))

	itemIDs := []int64{100}
	result, err := testClient.Project("my_project").Logs().GetLogsUnder(
		t.Context(),
		types.GetLogsUnderRq{
			ItemIds:  itemIDs,
			LogLevel: expectedLevel,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestLogScope_GetPageNumber_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log/42/page")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{"number": 3}`))
		},
	))

	result, err := testClient.Project("my_project").Logs().GetPageNumber(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.InEpsilon(t, float64(3), result["number"], 0.001)
}

func TestLogScope_GetPageNumber_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertPath(t, httpReq, "/api/v1/my_project/log/42/page")

			query := httpReq.URL.Query()
			assert.Equal(t, "100", query.Get("filter.eq.item"))
			assert.Equal(t, "20", query.Get("page.size"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{"number": 5}`))
		},
	))

	result, err := testClient.Project("my_project").Logs().GetPageNumber(
		t.Context(),
		42,
		client.WithLogPageItemID(100),
		client.WithLogPageNumberPageSize(20),
	)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(5), result["number"], 0.001)
}

func TestLogScope_GetPageNumber_WithSort(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertPath(t, httpReq, "/api/v1/my_project/log/42/page")

			query := httpReq.URL.Query()
			assert.Equal(t, "logTime,desc", query.Get("page.sort"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{"number": 1}`))
		},
	))

	result, err := testClient.Project("my_project").Logs().GetPageNumber(
		t.Context(),
		42,
		client.WithLogPageNumberSort("logTime,desc"),
	)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(1), result["number"], 0.001)
}

func TestLogScope_GetPageNumber_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Logs().GetPageNumber(t.Context(), 42)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestLogScope_GetPageNumber_EmptyResponse(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{}`))
	}))

	result, err := testClient.Project("my_project").Logs().GetPageNumber(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result)
}

func TestLogScope_GetPageNumber_EmptyJSONBody(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))

	result, err := testClient.Project("my_project").Logs().GetPageNumber(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result)
}

func TestLogScope_GetLogLocations_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log/locations/100")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[
			{
				"id": 1,
				"uuid": "loc-uuid-1",
				"level": "ERROR",
				"message": "failure detected",
				"itemId": 100,
				"launchId": 10
			}
		]`))
		},
	))

	result, err := testClient.Project("my_project").Logs().GetLogLocations(t.Context(), 100)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, int64(1), result[0].Id)
	assert.Equal(t, "loc-uuid-1", result[0].Uuid)
	require.NotNil(t, result[0].Message)
	assert.Equal(t, "failure detected", *result[0].Message)
}

func TestLogScope_GetLogLocations_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertPath(t, httpReq, "/api/v1/my_project/log/locations/100")

			query := httpReq.URL.Query()
			assert.Equal(t, "10", query.Get("page.size"))
			assert.Equal(t, "2", query.Get("page.page"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[]`))
		},
	))

	_, err := testClient.Project("my_project").Logs().GetLogLocations(
		t.Context(),
		100,
		client.WithLogLocationsPageSize(10),
		client.WithLogLocationsPageNumber(2),
	)
	require.NoError(t, err)
}

func TestLogScope_GetLogLocations_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Logs().GetLogLocations(t.Context(), 100)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestLogScope_GetLogLocations_EmptyResponse(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[]`))
		},
	))

	result, err := testClient.Project("my_project").Logs().GetLogLocations(t.Context(), 100)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result)
}

func TestLogScope_GetLogLocations_EmptyJSONBody(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))

	result, err := c.Project("my_project").Logs().GetLogLocations(t.Context(), 100)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result)
}

func TestLogScope_SearchLogLocations_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log/locations/search/100")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [
				{
					"id": 1,
					"uuid": "search-uuid-1",
					"level": "ERROR",
					"message": "search hit"
				}
			],
			"page": {"number": 0, "size": 10, "totalElements": 1, "totalPages": 1}
		}`))
		},
	))

	result, err := testClient.Project("my_project").Logs().SearchLogLocations(t.Context(), 100)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Content)
	require.Len(t, *result.Content, 1)

	require.NotNil(t, (*result.Content)[0].Message)
	assert.Equal(t, "search hit", *(*result.Content)[0].Message)
}

func TestLogScope_SearchLogLocations_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertPath(t, httpReq, "/api/v1/my_project/log/locations/search/100")

			query := httpReq.URL.Query()
			assert.Equal(t, "5", query.Get("page.size"))
			assert.Equal(t, "1", query.Get("page.page"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
		},
	))

	_, err := testClient.Project("my_project").Logs().SearchLogLocations(
		t.Context(),
		100,
		client.WithSearchLogLocationsPageSize(5),
		client.WithSearchLogLocationsPageNumber(1),
	)
	require.NoError(t, err)
}

func TestLogScope_SearchLogLocations_EmptyResponse(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
		},
	))

	result, err := testClient.Project("my_project").Logs().SearchLogLocations(t.Context(), 100)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Content)
}

func TestLogScope_SearchLogLocations_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Logs().SearchLogLocations(t.Context(), 100)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestLogScope_GetNestedLogs_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/my_project/log/nested/100")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [{"id": 1, "message": "nested log"}],
			"page": {"number": 0, "size": 10, "totalElements": 1, "totalPages": 1}
		}`))
		},
	))

	result, err := testClient.Project("my_project").Logs().GetNestedLogs(t.Context(), 100)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Content)
	assert.Len(t, *result.Content, 1)
	assert.EqualValues(t, 1, *result.Page.TotalElements)
}

func TestLogScope_GetNestedLogs_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, r *http.Request) {
			assertPath(t, r, "/api/v1/my_project/log/nested/100")

			query := r.URL.Query()
			assert.Equal(t, "25", query.Get("page.size"))
			assert.Equal(t, "3", query.Get("page.page"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [],
			"page": {"number": 3, "size": 25, "totalElements": 0, "totalPages": 0}
		}`))
		},
	))

	_, err := testClient.Project("my_project").Logs().GetNestedLogs(
		t.Context(),
		100,
		client.WithNestedLogsPageSize(25),
		client.WithNestedLogsPageNumber(3),
	)
	require.NoError(t, err)
}

func TestLogScope_GetNestedLogs_EmptyResponse(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	result, err := testClient.Project("my_project").Logs().GetNestedLogs(t.Context(), 100)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Content)
}

func TestLogScope_GetNestedLogs_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Logs().GetNestedLogs(t.Context(), 100)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestLogScope_List_WithSortCleared(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assert.Empty(t, httpReq.URL.Query().Get("page.sort"))

			responseWriter.Header().Set("Content-Type", "application/json")

			const emptyLogListPage = `{"content":[],"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}}`

			_, _ = responseWriter.Write([]byte(emptyLogListPage))
		},
	))

	_, err := testClient.Project("my_project").Logs().List(t.Context(), client.WithLogSort())
	require.NoError(t, err)
}

func TestLogScope_GetPageNumber_WithSortCleared(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertPath(t, httpReq, "/api/v1/my_project/log/42/page")
			assert.Empty(t, httpReq.URL.Query().Get("page.sort"))

			responseWriter.Header().Set("Content-Type", "application/json")
			_, _ = responseWriter.Write([]byte(`{"number": 1}`))
		},
	))

	_, err := testClient.Project("my_project").Logs().GetPageNumber(
		t.Context(), 42, client.WithLogPageNumberSort(""),
	)
	require.NoError(t, err)
}

//nolint:dupl // structured like sibling tests; shared HTTP/table scaffolding
func TestLogScope_CreateBatchAsync_201Response(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertPath(t, httpReq, "/api/v2/my_project/log")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(`{"responses":[{"id":"created-log-1"}]}`))
		},
	))

	result, err := testClient.Project("my_project").Logs().CreateBatchAsync(
		t.Context(),
		client.BatchLogEntry{
			LaunchUUID: "launch-uuid-1",
			Level:      "INFO",
			Message:    "Test",
			Time:       time.UnixMilli(1700000000000).UTC(),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Responses)
	require.Len(t, *result.Responses, 1)
	assert.Equal(t, "created-log-1", *(*result.Responses)[0].Id)
}

//nolint:dupl // structured like sibling tests; shared HTTP/table scaffolding
func TestLogScope_CreateBatch_201Response(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertPath(t, httpReq, "/api/v1/my_project/log")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(`{"responses":[{"id":"created-log-2"}]}`))
		},
	))

	result, err := testClient.Project("my_project").Logs().CreateBatch(
		t.Context(),
		client.BatchLogEntry{
			LaunchUUID: "launch-uuid-1",
			Level:      "INFO",
			Message:    "Test",
			Time:       time.UnixMilli(1700000000000).UTC(),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Responses)
	require.Len(t, *result.Responses, 1)
	assert.Equal(t, "created-log-2", *(*result.Responses)[0].Id)
}

func TestLogScope_CreateBatchAsync_NonJSONError(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>Bad Gateway</html>"))
	}))

	_, err := c.Project("my_project").Logs().CreateBatchAsync(
		t.Context(),
		client.BatchLogEntry{
			LaunchUUID: "launch-uuid-1",
			Level:      "INFO",
			Message:    "Test",
			Time:       time.UnixMilli(1700000000000).UTC(),
		},
	)
	require.Error(t, err)
	assert.True(t, client.HasStatusCode(err, http.StatusBadGateway))
}

func TestLogScope_CreateBatchAsync_201BadJSON(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("not json"))
	}))

	_, err := c.Project("my_project").Logs().CreateBatchAsync(
		t.Context(),
		client.BatchLogEntry{
			LaunchUUID: "launch-uuid-1",
			Level:      "INFO",
			Message:    "Test",
			Time:       time.UnixMilli(1700000000000).UTC(),
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create log batch async")
}

func TestLogScope_CreateBatch_201BadJSON(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("not json"))
	}))

	_, err := c.Project("my_project").Logs().CreateBatch(
		t.Context(),
		client.BatchLogEntry{
			LaunchUUID: "launch-uuid-1",
			Level:      "INFO",
			Message:    "Test",
			Time:       time.UnixMilli(1700000000000).UTC(),
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create log batch")
}

func TestLogScope_CreateBatch_NonJSONError(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>Bad Gateway</html>"))
	}))

	_, err := c.Project("my_project").Logs().CreateBatch(
		t.Context(),
		client.BatchLogEntry{
			LaunchUUID: "launch-uuid-1",
			Level:      "INFO",
			Message:    "Test",
			Time:       time.UnixMilli(1700000000000).UTC(),
		},
	)
	require.Error(t, err)
	assert.True(t, client.HasStatusCode(err, http.StatusBadGateway))
}
