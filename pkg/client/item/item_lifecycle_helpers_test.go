package item_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/clienttest"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	finishItemCaller = func(
		*client.Client,
		context.Context,
		string,
		types.FinishTestItemRQ,
	) (*types.OperationCompletionRS, error)
	finishItemErrorCall = func(
		*client.Client,
		context.Context,
		string,
		types.FinishTestItemRQ,
	) error
	startChildItemCaller = func(
		*client.Client,
		context.Context,
		string,
		types.StartTestItemRQ,
	) (*types.EntryCreatedAsyncRS, error)
	startChildItemErrorCall = func(
		*client.Client,
		context.Context,
		string,
		types.StartTestItemRQ,
	) error
	startRootItemCaller = func(
		*client.Client,
		context.Context,
		types.StartTestItemRQ,
	) (*types.EntryCreatedAsyncRS, error)
	startRootItemErrorCall = func(
		*client.Client,
		context.Context,
		types.StartTestItemRQ,
	) error
)

func startRootItemJSONPostHandler(
	t *testing.T,
	apiPath string,
	suiteName string,
	startTime time.Time,
	itemType types.StartTestItemRQType,
	responseJSON string,
) http.HandlerFunc {
	t.Helper()

	return func(responseWriter http.ResponseWriter, httpReq *http.Request) {
		testutil.AssertBearerToken(t, httpReq)
		testutil.AssertPath(t, httpReq, apiPath)
		testutil.AssertMethod(t, httpReq, http.MethodPost)

		body, err := io.ReadAll(httpReq.Body)
		if !assert.NoError(t, err) {
			return
		}

		var payload map[string]any
		if !assert.NoError(t, json.Unmarshal(body, &payload)) {
			return
		}

		assert.Equal(t, "launch-uuid-1", payload["launchUuid"])
		assert.Equal(t, suiteName, payload["name"])
		assert.Equal(t, startTime.UTC().Format(time.RFC3339Nano), payload["startTime"])
		assert.Equal(t, string(itemType), payload["type"])
		assert.Equal(t, true, payload["hasStats"])

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusCreated)
		_, _ = responseWriter.Write([]byte(responseJSON))
	}
}

func runStartRootItemSuccess(
	t *testing.T,
	apiPath string,
	suiteName string,
	responseJSON string,
	wantUUID string,
	call startRootItemCaller,
) {
	t.Helper()

	startTime := time.UnixMilli(1700000000000).UTC()
	itemType := types.StartTestItemRQType("SUITE")
	hasStats := true

	handler := startRootItemJSONPostHandler(
		t,
		apiPath,
		suiteName,
		startTime,
		itemType,
		responseJSON,
	)
	testClient := clienttest.NewClient(t, handler)

	request := types.StartTestItemRQ{
		LaunchUuid: "launch-uuid-1",
		Name:       suiteName,
		StartTime:  startTime,
		Type:       itemType,
		HasStats:   &hasStats,
	}

	result, err := call(testClient, t.Context(), request)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, wantUUID, *result.Id)
}

func startRootItemBadRequestUnauthorizedCases() []struct {
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
			body:       `{"errorCode": 4001, "message": "Invalid test item payload."}`,
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

func runStartRootItemErrorTests(
	t *testing.T,
	suiteName string,
	call startRootItemErrorCall,
) {
	t.Helper()

	for _, testCase := range startRootItemBadRequestUnauthorizedCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			itemType := types.StartTestItemRQType("SUITE")
			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)

			err := call(testClient, t.Context(), types.StartTestItemRQ{
				LaunchUuid: "launch-uuid-1",
				Name:       suiteName,
				StartTime:  time.Now(),
				Type:       itemType,
			})
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func startChildItemJSONHandler(
	t *testing.T,
	apiPath string,
	wantChildName string,
	startTime time.Time,
	itemType types.StartTestItemRQType,
	responseJSON string,
) http.HandlerFunc {
	t.Helper()

	return func(responseWriter http.ResponseWriter, httpReq *http.Request) {
		testutil.AssertBearerToken(t, httpReq)
		testutil.AssertPath(t, httpReq, apiPath)
		testutil.AssertMethod(t, httpReq, http.MethodPost)

		payload, ok := testutil.DecodeRequestJSONMap(t, httpReq)
		if !ok {
			return
		}

		assertStartChildPostPayload(t, payload, wantChildName, startTime, itemType)

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusCreated)
		_, _ = responseWriter.Write([]byte(responseJSON))
	}
}

func runStartChildItemSuccess(
	t *testing.T,
	apiPath string,
	childName string,
	responseJSON string,
	wantUUID string,
	call startChildItemCaller,
) {
	t.Helper()

	startTime := time.UnixMilli(1700000001000).UTC()
	itemType := types.StartTestItemRQType("TEST")
	hasStats := true

	handler := startChildItemJSONHandler(
		t,
		apiPath,
		childName,
		startTime,
		itemType,
		responseJSON,
	)
	testClient := clienttest.NewClient(t, handler)

	request := types.StartTestItemRQ{
		LaunchUuid: "launch-uuid-1",
		Name:       childName,
		StartTime:  startTime,
		Type:       itemType,
		HasStats:   &hasStats,
	}

	result, err := call(testClient, t.Context(), "suite-uuid-1", request)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, wantUUID, *result.Id)
}

func itemNotFoundUnauthorizedCases() []struct {
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
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40440, "message": "Parent item not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}
}

func runStartChildItemErrorTests(
	t *testing.T,
	childName string,
	call startChildItemErrorCall,
) {
	t.Helper()

	for _, testCase := range itemNotFoundUnauthorizedCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			itemType := types.StartTestItemRQType("TEST")
			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)

			err := call(testClient, t.Context(), "missing-parent", types.StartTestItemRQ{
				LaunchUuid: "launch-uuid-1",
				Name:       childName,
				StartTime:  time.Now(),
				Type:       itemType,
			})
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func finishItemPutHandler(
	t *testing.T,
	apiPath string,
	endTime time.Time,
	status types.FinishTestItemRQStatus,
) http.HandlerFunc {
	t.Helper()

	return func(responseWriter http.ResponseWriter, httpReq *http.Request) {
		testutil.AssertBearerToken(t, httpReq)
		testutil.AssertPath(t, httpReq, apiPath)
		testutil.AssertMethod(t, httpReq, http.MethodPut)

		payload, ok := testutil.DecodeRequestJSONMap(t, httpReq)
		if !ok {
			return
		}

		assertFinishPutPayload(t, payload, endTime, status)

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusOK)
		_, _ = responseWriter.Write([]byte(`{"message":"Test item finished."}`))
	}
}

func runFinishItemSuccess(
	t *testing.T,
	apiPath string,
	call finishItemCaller,
) {
	t.Helper()

	endTime := time.UnixMilli(1700000005000).UTC()
	status := types.FinishTestItemRQStatusPASSED

	testClient := clienttest.NewClient(t, finishItemPutHandler(t, apiPath, endTime, status))

	result, err := call(testClient, t.Context(), "test-uuid-1", types.FinishTestItemRQ{
		LaunchUuid: "launch-uuid-1",
		EndTime:    endTime,
		Status:     &status,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Test item finished.", *result.Message)
}

func itemFinishNotFoundUnauthorizedCases() []struct {
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
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40440, "message": "Test item not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}
}

func runFinishItemErrorTests(
	t *testing.T,
	call finishItemErrorCall,
) {
	t.Helper()

	for _, testCase := range itemFinishNotFoundUnauthorizedCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			status := types.FinishTestItemRQStatusPASSED
			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)

			err := call(testClient, t.Context(), "missing-item", types.FinishTestItemRQ{
				LaunchUuid: "launch-uuid-1",
				EndTime:    time.Now(),
				Status:     &status,
			})
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}
