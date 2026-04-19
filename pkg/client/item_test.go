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

func TestTestItemScope_GetByID_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/42")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"uuid": "item-uuid-42",
			"name": "Suite 42",
			"status": "PASSED",
			"type": "SUITE",
			"launchId": 1001,
			"parent": 1000,
			"hasChildren": true,
			"path": "1.2.42"
		}`))
	}))

	item, err := c.Project("my_project").Items().GetByID(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, item)
	require.NotNil(t, item.Id)
	require.NotNil(t, item.Uuid)
	require.NotNil(t, item.Name)
	require.NotNil(t, item.Type)

	assert.Equal(t, int64(42), *item.Id)
	assert.Equal(t, "item-uuid-42", *item.Uuid)
	assert.Equal(t, "Suite 42", *item.Name)
	assert.Equal(t, "SUITE", *item.Type)
}

func TestTestItemScope_GetByID_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		itemID     int64
		statusCode int
		body       string
		check      func(*testing.T, error)
	}{
		{
			name:       "NotFound",
			itemID:     999,
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40440, "message": "Test item '999' not found."}`,
			check: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.True(t, client.IsNotFound(err))
			},
		},
		{
			name:       "Unauthorized",
			itemID:     42,
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.True(t, client.IsUnauthorized(err))
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))

			_, err := testClient.Project("my_project").Items().GetByID(t.Context(), testCase.itemID)
			testCase.check(t, err)
		})
	}
}

func TestTestItemScope_GetByUUID_PrefersV2Response(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/uuid/item-uuid-42")
		assertMethod(t, r, http.MethodGet)
		assertHeader(t, r, "Accept", "application/x.reportportal.test.v2+json")

		w.Header().Set("Content-Type", "application/x.reportportal.test.v2+json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"uuid": "item-uuid-42",
			"name": "Suite 42",
			"status": "FAILED",
			"type": "SUITE",
			"startTime": "2023-11-14T22:13:20Z"
		}`))
	}))

	item, err := c.Project("my_project").Items().GetByUUID(t.Context(), "item-uuid-42")
	require.NoError(t, err)
	require.NotNil(t, item)
	require.NotNil(t, item.Uuid)
	require.NotNil(t, item.Status)

	assert.Equal(t, "item-uuid-42", *item.Uuid)
	assert.Equal(t, "FAILED", *item.Status)
}

func TestTestItemScope_GetByUUID_LegacyJSONReturnsError(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/uuid/item-uuid-42")
		assertMethod(t, r, http.MethodGet)
		assertHeader(t, r, "Accept", "application/x.reportportal.test.v2+json")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"uuid": "item-uuid-42",
			"name": "Suite 42",
			"status": "FAILED",
			"type": "SUITE"
		}`))
	}))

	_, err := c.Project("my_project").Items().GetByUUID(t.Context(), "item-uuid-42")
	require.Error(t, err)
	assert.ErrorContains(t, err, "legacy JSON shape")
}

func TestTestItemScope_GetByUUID_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40440, "message": "Test item not found."}`))

	_, err := c.Project("my_project").Items().GetByUUID(t.Context(), "unknown-uuid")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestTestItemScope_StartRoot_Success(t *testing.T) {
	t.Parallel()

	startTime := time.UnixMilli(1700000000000).UTC()
	itemType := types.StartTestItemRQType("SUITE")
	hasStats := true

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item")
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "launch-uuid-1", payload["launchUuid"])
		assert.Equal(t, "Authentication Suite", payload["name"])
		assert.Equal(t, startTime.UTC().Format(time.RFC3339Nano), payload["startTime"])
		assert.Equal(t, string(itemType), payload["type"])
		assert.Equal(t, true, payload["hasStats"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"suite-uuid-1"}`))
	}))

	result, err := c.Project("my_project").Items().StartRoot(
		t.Context(),
		types.StartTestItemRQ{
			LaunchUuid: "launch-uuid-1",
			Name:       "Authentication Suite",
			StartTime:  startTime,
			Type:       itemType,
			HasStats:   &hasStats,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, "suite-uuid-1", *result.Id)
}

func TestTestItemScope_StartRoot_Errors(t *testing.T) {
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			itemType := types.StartTestItemRQType("SUITE")
			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().StartRoot(
				t.Context(),
				types.StartTestItemRQ{
					LaunchUuid: "launch-uuid-1",
					Name:       "Authentication Suite",
					StartTime:  time.Now(),
					Type:       itemType,
				},
			)
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_StartChild_Success(t *testing.T) {
	t.Parallel()

	startTime := time.UnixMilli(1700000001000).UTC()
	itemType := types.StartTestItemRQType("TEST")
	hasStats := true

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/suite-uuid-1")
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "launch-uuid-1", payload["launchUuid"])
		assert.Equal(t, "Login Test", payload["name"])
		assert.Equal(t, startTime.UTC().Format(time.RFC3339Nano), payload["startTime"])
		assert.Equal(t, string(itemType), payload["type"])
		assert.Equal(t, true, payload["hasStats"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"test-uuid-1"}`))
	}))

	result, err := c.Project("my_project").Items().StartChild(
		t.Context(),
		"suite-uuid-1",
		types.StartTestItemRQ{
			LaunchUuid: "launch-uuid-1",
			Name:       "Login Test",
			StartTime:  startTime,
			Type:       itemType,
			HasStats:   &hasStats,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, "test-uuid-1", *result.Id)
}

func TestTestItemScope_StartChild_Errors(t *testing.T) {
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			itemType := types.StartTestItemRQType("TEST")
			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().StartChild(
				t.Context(),
				"missing-parent",
				types.StartTestItemRQ{
					LaunchUuid: "launch-uuid-1",
					Name:       "Login Test",
					StartTime:  time.Now(),
					Type:       itemType,
				},
			)
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_Finish_Success(t *testing.T) {
	t.Parallel()

	endTime := time.UnixMilli(1700000005000).UTC()
	status := types.FinishTestItemRQStatusPASSED

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/test-uuid-1")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "launch-uuid-1", payload["launchUuid"])
		assert.Equal(t, endTime.UTC().Format(time.RFC3339Nano), payload["endTime"])
		assert.Equal(t, string(status), payload["status"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Test item finished."}`))
	}))

	result, err := c.Project("my_project").Items().Finish(
		t.Context(),
		"test-uuid-1",
		types.FinishTestItemRQ{
			LaunchUuid: "launch-uuid-1",
			EndTime:    endTime,
			Status:     &status,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Test item finished.", *result.Message)
}

func TestTestItemScope_Finish_Errors(t *testing.T) {
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			status := types.FinishTestItemRQStatusPASSED
			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().Finish(
				t.Context(),
				"missing-item",
				types.FinishTestItemRQ{
					LaunchUuid: "launch-uuid-1",
					EndTime:    time.Now(),
					Status:     &status,
				},
			)
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_StartRootAsync_Success(t *testing.T) {
	t.Parallel()

	startTime := time.UnixMilli(1700000000000).UTC()
	itemType := types.StartTestItemRQType("SUITE")
	hasStats := true

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v2/my_project/item")
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "launch-uuid-1", payload["launchUuid"])
		assert.Equal(t, "Async Suite", payload["name"])
		assert.Equal(t, startTime.UTC().Format(time.RFC3339Nano), payload["startTime"])
		assert.Equal(t, string(itemType), payload["type"])
		assert.Equal(t, true, payload["hasStats"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"async-suite-uuid-1"}`))
	}))

	result, err := c.Project("my_project").Items().StartRootAsync(
		t.Context(),
		types.StartTestItemRQ{
			LaunchUuid: "launch-uuid-1",
			Name:       "Async Suite",
			StartTime:  startTime,
			Type:       itemType,
			HasStats:   &hasStats,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, "async-suite-uuid-1", *result.Id)
}

func TestTestItemScope_StartRootAsync_Errors(t *testing.T) {
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			itemType := types.StartTestItemRQType("SUITE")
			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().StartRootAsync(
				t.Context(),
				types.StartTestItemRQ{
					LaunchUuid: "launch-uuid-1",
					Name:       "Async Suite",
					StartTime:  time.Now(),
					Type:       itemType,
				},
			)
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_StartChildAsync_Success(t *testing.T) {
	t.Parallel()

	startTime := time.UnixMilli(1700000001000).UTC()
	itemType := types.StartTestItemRQType("TEST")
	hasStats := true

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v2/my_project/item/suite-uuid-1")
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "launch-uuid-1", payload["launchUuid"])
		assert.Equal(t, "Async Test", payload["name"])
		assert.Equal(t, startTime.UTC().Format(time.RFC3339Nano), payload["startTime"])
		assert.Equal(t, string(itemType), payload["type"])
		assert.Equal(t, true, payload["hasStats"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"async-test-uuid-1"}`))
	}))

	result, err := c.Project("my_project").Items().StartChildAsync(
		t.Context(),
		"suite-uuid-1",
		types.StartTestItemRQ{
			LaunchUuid: "launch-uuid-1",
			Name:       "Async Test",
			StartTime:  startTime,
			Type:       itemType,
			HasStats:   &hasStats,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, "async-test-uuid-1", *result.Id)
}

func TestTestItemScope_StartChildAsync_Errors(t *testing.T) {
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			itemType := types.StartTestItemRQType("TEST")
			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().StartChildAsync(
				t.Context(),
				"missing-parent",
				types.StartTestItemRQ{
					LaunchUuid: "launch-uuid-1",
					Name:       "Async Test",
					StartTime:  time.Now(),
					Type:       itemType,
				},
			)
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_FinishAsync_Success(t *testing.T) {
	t.Parallel()

	endTime := time.UnixMilli(1700000005000).UTC()
	status := types.FinishTestItemRQStatusPASSED

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v2/my_project/item/test-uuid-1")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "launch-uuid-1", payload["launchUuid"])
		assert.Equal(t, endTime.UTC().Format(time.RFC3339Nano), payload["endTime"])
		assert.Equal(t, string(status), payload["status"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Test item finished."}`))
	}))

	result, err := c.Project("my_project").Items().FinishAsync(
		t.Context(),
		"test-uuid-1",
		types.FinishTestItemRQ{
			LaunchUuid: "launch-uuid-1",
			EndTime:    endTime,
			Status:     &status,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Test item finished.", *result.Message)
}

func TestTestItemScope_FinishAsync_Errors(t *testing.T) {
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			status := types.FinishTestItemRQStatusPASSED
			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().FinishAsync(
				t.Context(),
				"missing-item",
				types.FinishTestItemRQ{
					LaunchUuid: "launch-uuid-1",
					EndTime:    time.Now(),
					Status:     &status,
				},
			)
			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_List_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": 101, "uuid": "suite-101", "name": "Suite A", "status": "PASSED", "type": "SUITE"},
				{"id": 102, "uuid": "test-102", "name": "Test A", "status": "FAILED", "type": "TEST"}
			],
			"page": {"number": 0, "size": 10, "totalElements": 2, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Items().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Content)
	require.Len(t, *result.Content, 2)
	require.NotNil(t, (*result.Content)[0].Name)

	assert.Equal(t, "Suite A", *(*result.Content)[0].Name)
	assert.EqualValues(t, 2, *result.Page.TotalElements)
}

func TestTestItemScope_List_WithFilterOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/v1/my_project/item")

		query := r.URL.Query()
		assert.Equal(t, "123", query.Get("filter.eq.launchId"))
		assert.Equal(t, "456", query.Get("filter.eq.parentId"))
		assert.Equal(t, "SUITE", query.Get("filter.eq.type"))
		assert.Equal(t, "Auth Suite", query.Get("filter.eq.name"))
		assert.Equal(t, "FAILED", query.Get("filter.eq.status"))
		assert.Equal(t, "true", query.Get("filter.eq.hasChildren"))
		assert.Equal(t, "789", query.Get("filter.eq.path"))
		assert.Equal(t, "2", query.Get("page.page"))
		assert.Equal(t, "5", query.Get("page.size"))
		assert.Equal(t, "startTime,desc", query.Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 2, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Project("my_project").Items().List(
		t.Context(),
		client.WithItemLaunchID(123),
		client.WithItemParentID(456),
		client.WithItemType("SUITE"),
		client.WithItemName("Auth Suite"),
		client.WithItemStatus("FAILED"),
		client.WithItemHasChildren(true),
		client.WithItemPath(789),
		client.WithItemPageNumber(2),
		client.WithItemPageSize(5),
		client.WithItemSort("startTime,desc"),
	)
	require.NoError(t, err)
}

type getByIDsTestCase struct {
	name             string
	ids              []int64
	expectedQueryIDs []string
	responseBody     string
	expectedLength   int
	expectedFirstID  int64
	expectedLastType string
}

// runGetByIDsTestCase runs a single GetByIDs table-driven test case against an httptest server.
func runGetByIDsTestCase(t *testing.T, testCase getByIDsTestCase) {
	t.Helper()

	testClient := newTestClient(
		t,
		http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			assertBearerToken(t, request, "test-token")
			assertPath(t, request, "/api/v1/my_project/item/items")
			assertMethod(t, request, http.MethodGet)

			query := request.URL.Query()["ids"]
			assert.Equal(t, testCase.expectedQueryIDs, query)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(testCase.responseBody))
		}),
	)

	items, err := testClient.Project("my_project").
		Items().
		GetByIDs(t.Context(), testCase.ids...)
	require.NoError(t, err)
	require.Len(t, items, testCase.expectedLength)
	require.NotNil(t, items[0].Id)
	require.NotNil(t, items[testCase.expectedLength-1].Type)

	assert.Equal(t, testCase.expectedFirstID, *items[0].Id)
	assert.Equal(t, testCase.expectedLastType, *items[testCase.expectedLength-1].Type)
}

func TestTestItemScope_GetByIDs(t *testing.T) {
	t.Parallel()

	testCases := []getByIDsTestCase{
		{
			name:             "MultipleIDs",
			ids:              []int64{101, 202},
			expectedQueryIDs: []string{"101", "202"},
			responseBody: `[
				{"id": 101, "uuid": "suite-101", "name": "Suite A", "type": "SUITE"},
				{"id": 202, "uuid": "test-202", "name": "Test B", "type": "TEST"}
			]`,
			expectedLength:   2,
			expectedFirstID:  101,
			expectedLastType: "TEST",
		},
		{
			name:             "SingleID",
			ids:              []int64{303},
			expectedQueryIDs: []string{"303"},
			responseBody: `[
				{"id": 303, "uuid": "step-303", "name": "Step C", "type": "STEP"}
			]`,
			expectedLength:   1,
			expectedFirstID:  303,
			expectedLastType: "STEP",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runGetByIDsTestCase(t, testCase)
		})
	}
}

func TestTestItemScope_GetByIDs_EmptyInput(t *testing.T) {
	t.Parallel()

	var called bool

	testClient := newTestClient(
		t,
		http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			called = true

			responseWriter.WriteHeader(http.StatusInternalServerError)
		}),
	)

	items, err := testClient.Project("my_project").Items().GetByIDs(t.Context())
	require.NoError(t, err)
	assert.False(t, called)
	assert.Empty(t, items)
}

func TestTestItemScope_Update_Success(t *testing.T) {
	t.Parallel()

	description := "Updated item description"

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/42/update")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "Updated item description", payload["description"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Test item successfully updated."}`))
	}))

	result, err := c.Project("my_project").Items().Update(
		t.Context(),
		42,
		types.UpdateTestItemRQ{
			Description: &description,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Test item successfully updated.", *result.Message)
}

func TestTestItemScope_Update_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40440, "message": "Test item '999' not found."}`,
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
			_, err := testClient.Project("my_project").Items().Update(
				t.Context(),
				999,
				types.UpdateTestItemRQ{},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_Delete_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/42")
		assertMethod(t, r, http.MethodDelete)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Test item successfully deleted."}`))
	}))

	result, err := c.Project("my_project").Items().Delete(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Test item successfully deleted.", *result.Message)
}

func TestTestItemScope_Delete_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40440, "message": "Test item '999' not found."}`,
			check:      client.IsNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().Delete(t.Context(), 999)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_DefineIssueType_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		issues := payload["issues"].([]any)
		assert.Len(t, issues, 1)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"issueType":"pb001","comment":"Product bug"}]`))
	}))

	issues, err := c.Project("my_project").Items().DefineIssueType(
		t.Context(),
		types.DefineIssueRQ{
			Issues: []types.IssueDefinition{
				{
					TestItemId: 42,
					Issue: types.Issue{
						IssueType: "pb001",
					},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "pb001", issues[0].IssueType)
}

func TestTestItemScope_DefineIssueType_Errors(t *testing.T) {
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
			body:       `{"errorCode": 4001, "message": "Invalid issue definition."}`,
			check:      client.IsBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().DefineIssueType(
				t.Context(),
				types.DefineIssueRQ{
					Issues: []types.IssueDefinition{},
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_DeleteBulk_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item")
		assertMethod(t, r, http.MethodDelete)

		query := r.URL.Query()["ids"]
		assert.Equal(t, []string{"101", "202"}, query)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"message":"Items successfully deleted."}]`))
	}))

	result, err := c.Project("my_project").Items().DeleteBulk(t.Context(), []int64{101, 202})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].Message)
	assert.Equal(t, "Items successfully deleted.", *result[0].Message)
}

func TestTestItemScope_DeleteBulk_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40440, "message": "Test items not found."}`,
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
			_, err := testClient.Project("my_project").Items().DeleteBulk(
				t.Context(),
				[]int64{101, 202},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_BulkUpdate_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/info")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		ids := payload["ids"].([]any)
		assert.Len(t, ids, 2)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Items successfully updated."}`))
	}))

	comment := "Updated via bulk"
	description := types.Description{Comment: &comment}
	result, err := c.Project("my_project").Items().BulkUpdate(
		t.Context(),
		types.BulkInfoUpdateRQ{
			Ids:         []int64{101, 202},
			Description: &description,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Items successfully updated.", *result.Message)
}

func TestTestItemScope_BulkUpdate_Errors(t *testing.T) {
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().BulkUpdate(
				t.Context(),
				types.BulkInfoUpdateRQ{
					Ids: []int64{101, 202},
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_GetAttributeKeys_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/attribute/keys")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, "env", query.Get("filter.cnt.attributeKey"))
		assert.Equal(t, "42", query.Get("launch"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["environment","env.type"]`))
	}))

	keys, err := c.Project("my_project").Items().GetAttributeKeys(t.Context(), 42, "env")
	require.NoError(t, err)
	require.Equal(t, []string{"environment", "env.type"}, keys)
}

func TestTestItemScope_GetAttributeKeys_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40410, "message": "Project 'unknown' not found."}`,
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
			_, err := testClient.Project("my_project").Items().GetAttributeKeys(
				t.Context(), 1, "env",
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_GetAttributeKeysForProject_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/attribute/keys/all")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, "build", query.Get("filter.cnt.attributeKey"))
		assert.Equal(t, "99", query.Get("filterId"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["build.number","build.branch"]`))
	}))

	keys, err := c.Project("my_project").Items().GetAttributeKeysForProject(
		t.Context(), 99, "build",
	)
	require.NoError(t, err)
	require.Equal(t, []string{"build.number", "build.branch"}, keys)
}

func TestTestItemScope_GetAttributeKeysForProject_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/attribute/keys/all")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, "tag", query.Get("filter.cnt.attributeKey"))
		assert.Equal(t, "50", query.Get("filterId"))
		assert.Equal(t, "true", query.Get("isLatest"))
		assert.Equal(t, "10", query.Get("launchesLimit"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["tag.priority"]`))
	}))

	keys, err := c.Project("my_project").Items().GetAttributeKeysForProject(
		t.Context(), 50, "tag",
		client.WithProjectKeysLatest(true),
		client.WithProjectKeysLaunchesLimit(10),
	)
	require.NoError(t, err)
	require.Equal(t, []string{"tag.priority"}, keys)
}

func TestTestItemScope_GetAttributeValues_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/attribute/values")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, "prod", query.Get("filter.cnt.attributeValue"))
		assert.Empty(t, query.Get("filter.eq.attributeKey"))
		assert.Equal(t, "42", query.Get("launch"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["production","prod-east"]`))
	}))

	values, err := c.Project("my_project").Items().GetAttributeValues(
		t.Context(), 42, "prod", nil,
	)
	require.NoError(t, err)
	require.Equal(t, []string{"production", "prod-east"}, values)
}

func TestTestItemScope_GetAttributeValues_WithAttributeKey(t *testing.T) {
	t.Parallel()

	key := "environment"

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/attribute/values")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, "stag", query.Get("filter.cnt.attributeValue"))
		assert.Equal(t, "environment", query.Get("filter.eq.attributeKey"))
		assert.Equal(t, "42", query.Get("launch"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["staging"]`))
	}))

	values, err := c.Project("my_project").Items().GetAttributeValues(
		t.Context(), 42, "stag", &key,
	)
	require.NoError(t, err)
	require.Equal(t, []string{"staging"}, values)
}

func TestTestItemScope_GetAttributeValues_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40410, "message": "Project 'unknown' not found."}`,
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
			_, err := testClient.Project("my_project").Items().GetAttributeValues(
				t.Context(), 1, "val", nil,
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_GetStepAttributeKeys_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/step/attribute/keys")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, "retry", query.Get("filter.cnt.attributeKey"))
		assert.Empty(t, query.Get("filter.eq.name"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["retry.count","retry.reason"]`))
	}))

	keys, err := c.Project("my_project").Items().GetStepAttributeKeys(
		t.Context(), "retry", nil,
	)
	require.NoError(t, err)
	require.Equal(t, []string{"retry.count", "retry.reason"}, keys)
}

func TestTestItemScope_GetStepAttributeKeys_WithStepName(t *testing.T) {
	t.Parallel()

	stepName := "Login Step"

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/step/attribute/keys")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, "tag", query.Get("filter.cnt.attributeKey"))
		assert.Equal(t, "Login Step", query.Get("filter.eq.name"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["tag.auth"]`))
	}))

	keys, err := c.Project("my_project").Items().GetStepAttributeKeys(
		t.Context(), "tag", &stepName,
	)
	require.NoError(t, err)
	require.Equal(t, []string{"tag.auth"}, keys)
}

func TestTestItemScope_GetStepAttributeKeys_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40410, "message": "Project 'unknown' not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"errorCode": 4003, "message": "Access denied."}`,
			check:      client.IsForbidden,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().GetStepAttributeKeys(
				t.Context(), "key", nil,
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_GetStepAttributeValues_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/step/attribute/values")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, "pass", query.Get("filter.cnt.attributeValue"))
		assert.Empty(t, query.Get("filter.eq.attributeKey"))
		assert.Empty(t, query.Get("filter.eq.name"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["password","passphrase"]`))
	}))

	values, err := c.Project("my_project").Items().GetStepAttributeValues(
		t.Context(), "pass", nil, nil,
	)
	require.NoError(t, err)
	require.Equal(t, []string{"password", "passphrase"}, values)
}

func TestTestItemScope_GetStepAttributeValues_WithAllFilters(t *testing.T) {
	t.Parallel()

	key := "status"
	stepName := "Verify Step"

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/step/attribute/values")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, "ok", query.Get("filter.cnt.attributeValue"))
		assert.Equal(t, "status", query.Get("filter.eq.attributeKey"))
		assert.Equal(t, "Verify Step", query.Get("filter.eq.name"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["ok","ok_with_warnings"]`))
	}))

	values, err := c.Project("my_project").Items().GetStepAttributeValues(
		t.Context(), "ok", &key, &stepName,
	)
	require.NoError(t, err)
	require.Equal(t, []string{"ok", "ok_with_warnings"}, values)
}

func TestTestItemScope_GetStepAttributeValues_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40410, "message": "Project 'unknown' not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"errorCode": 4003, "message": "Access denied."}`,
			check:      client.IsForbidden,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().GetStepAttributeValues(
				t.Context(), "val", nil, nil,
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_GetHistory_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/history")
		assertMethod(t, r, http.MethodGet)

		assert.Equal(t, "5", r.URL.Query().Get("historyDepth"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{
					"groupingField": "test-case-1",
					"resources": [
						{"id": 100, "uuid": "item-uuid-100", "name": "TestCase1", "status": "PASSED", "type": "TEST"},
						{"id": 101, "uuid": "item-uuid-101", "name": "TestCase1", "status": "FAILED", "type": "TEST"}
					]
				}
			],
			"page": {
				"number": 0,
				"size": 20,
				"totalElements": 1,
				"totalPages": 1
			}
		}`))
	}))

	result, err := c.Project("my_project").Items().GetHistory(
		t.Context(),
		client.WithHistoryDepth(5),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Content)
	require.Len(t, *result.Content, 1)
	require.NotNil(t, (*result.Content)[0].GroupingField)
	assert.Equal(t, "test-case-1", *(*result.Content)[0].GroupingField)
	require.NotNil(t, (*result.Content)[0].Resources)
	assert.Len(t, *(*result.Content)[0].Resources, 2)
}

func TestTestItemScope_GetHistory_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/history")
		assertMethod(t, r, http.MethodGet)

		q := r.URL.Query()
		assert.Equal(t, "10", q.Get("historyDepth"))
		assert.Equal(t, "42", q.Get("filter.eq.launchId"))
		assert.Equal(t, "FAILED", q.Get("filter.eq.status"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 20, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	result, err := c.Project("my_project").Items().GetHistory(
		t.Context(),
		client.WithHistoryDepth(10),
		client.WithHistoryLaunchID(42),
		client.WithHistoryItemStatus("FAILED"),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Content)
}

func TestTestItemScope_GetHistory_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40410, "message": "Project 'unknown' not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"errorCode": 4003, "message": "Access denied."}`,
			check:      client.IsForbidden,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().GetHistory(
				t.Context(),
				client.WithHistoryDepth(5),
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_GetHistory_WithAdditionalOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		option   client.HistoryOption
		expected map[string]string
	}{
		{
			name:   "WithHistoryFilterID",
			option: client.WithHistoryFilterID(42),
			expected: map[string]string{
				"filterId": "42",
			},
		},
		{
			name:   "WithHistoryLaunchesLimit",
			option: client.WithHistoryLaunchesLimit(10),
			expected: map[string]string{
				"launchesLimit": "10",
			},
		},
		{
			name:   "WithHistoryType",
			option: client.WithHistoryType("line"),
			expected: map[string]string{
				"type": "line",
			},
		},
		{
			name:   "WithHistoryItemName",
			option: client.WithHistoryItemName("login test"),
			expected: map[string]string{
				"filter.eq.name": "login test",
			},
		},
		{
			name:   "WithHistoryPageNumber",
			option: client.WithHistoryPageNumber(3),
			expected: map[string]string{
				"page.page": "3",
			},
		},
		{
			name:   "WithHistoryPageSize",
			option: client.WithHistoryPageSize(25),
			expected: map[string]string{
				"page.size": "25",
			},
		},
		{
			name:   "WithHistoryLatest",
			option: client.WithHistoryLatest(true),
			expected: map[string]string{
				"isLatest": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()

				for key, expectedValue := range tt.expected {
					assert.Equal(t, expectedValue, query.Get(key),
						"query parameter mismatch for "+key)
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"content": [],
					"page": {"number": 0, "size": 20, "totalElements": 0, "totalPages": 0}
				}`))
			}))

			_, err := c.Project("my_project").Items().GetHistory(t.Context(), tt.option)
			require.NoError(t, err)
		})
	}
}

func TestTestItemScope_GetStatistics_WithOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		option   client.StatisticsOption
		expected map[string]string
	}{
		{
			name:   "WithStatisticsParentID",
			option: client.WithStatisticsParentID(99),
			expected: map[string]string{
				"filter.eq.parentId": "99",
			},
		},
		{
			name:   "WithStatisticsStatus",
			option: client.WithStatisticsStatus("PASSED"),
			expected: map[string]string{
				"filter.eq.status": "PASSED",
			},
		},
		{
			name:   "WithStatisticsName",
			option: client.WithStatisticsName("suite A"),
			expected: map[string]string{
				"filter.eq.name": "suite A",
			},
		},
		{
			name:   "WithStatisticsType",
			option: client.WithStatisticsType("SUITE"),
			expected: map[string]string{
				"filter.eq.type": "SUITE",
			},
		},
		{
			name:   "WithStatisticsHasChildren",
			option: client.WithStatisticsHasChildren(true),
			expected: map[string]string{
				"filter.eq.hasChildren": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()

				for key, expectedValue := range tt.expected {
					assert.Equal(t, expectedValue, query.Get(key),
						"query parameter mismatch for "+key)
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"executions": {},
					"defects": {}
				}`))
			}))

			_, err := c.Project("my_project").Items().GetStatistics(t.Context(), tt.option)
			require.NoError(t, err)
		})
	}
}

func TestTestItemScope_GetStatistics_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/statistics")
		assertMethod(t, r, http.MethodGet)

		assert.Equal(t, "123", r.URL.Query().Get("filter.eq.launchId"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"executions": {"passed": 10, "failed": 3, "skipped": 1, "total": 14},
			"defects": {
				"product_bug": {"pb001": 2},
				"to_investigate": {"ti001": 1}
			}
		}`))
	}))

	result, err := c.Project("my_project").Items().GetStatistics(
		t.Context(),
		client.WithStatisticsLaunchID(123),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Executions)
	require.NotNil(t, result.Defects)

	assert.Equal(t, int32(10), (*result.Executions)["passed"])
	assert.Equal(t, int32(3), (*result.Executions)["failed"])
	assert.Equal(t, int32(2), (*result.Defects)["product_bug"]["pb001"])
}

func TestTestItemScope_GetStatistics_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40410, "message": "Project 'unknown' not found."}`,
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
			_, err := testClient.Project("my_project").Items().GetStatistics(t.Context())

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_GetSuggestedItems_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/suggest/42")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
			{
				"testItemResource": {"id": 100, "name": "Similar Test"},
				"suggestRs": {"issueType": "pb001", "matchScore": 0.95}
			}
		]`))
	}))

	items, err := c.Project("my_project").Items().GetSuggestedItems(t.Context(), 42)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].TestItemResource)
	require.NotNil(t, items[0].TestItemResource.Name)
	require.NotNil(t, items[0].SuggestRs)
	require.NotNil(t, items[0].SuggestRs.IssueType)

	assert.Equal(t, "Similar Test", *items[0].TestItemResource.Name)
	assert.Equal(t, "pb001", *items[0].SuggestRs.IssueType)
}

func TestTestItemScope_GetSuggestedItems_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40440, "message": "Test item '42' not found."}`,
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
			_, err := testClient.Project("my_project").Items().GetSuggestedItems(t.Context(), 42)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_GetSuggestedClusterItems_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/suggest/cluster/99")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
			{
				"testItemResource": {"id": 200, "name": "Cluster Match A"},
				"suggestRs": {"clusterId": 99, "matchScore": 0.88}
			},
			{
				"testItemResource": {"id": 201, "name": "Cluster Match B"},
				"suggestRs": {"clusterId": 99, "matchScore": 0.75}
			}
		]`))
	}))

	items, err := c.Project("my_project").Items().GetSuggestedClusterItems(t.Context(), 99)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.NotNil(t, items[0].TestItemResource)
	require.NotNil(t, items[0].TestItemResource.Name)

	assert.Equal(t, "Cluster Match A", *items[0].TestItemResource.Name)
	assert.Equal(t, "Cluster Match B", *items[1].TestItemResource.Name)
}

func TestTestItemScope_GetSuggestedClusterItems_Errors(t *testing.T) {
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
			body:       `{"errorCode": 40440, "message": "Cluster '99' not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"errorCode": 4003, "message": "Access denied."}`,
			check:      client.IsForbidden,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").
				Items().
				GetSuggestedClusterItems(t.Context(), 99)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_HandleSuggestChoice_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/suggest/choice")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload []map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Len(t, payload, 1)
		assert.Equal(t, "pb001", payload[0]["issueType"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Suggestion choice applied."}`))
	}))

	issueType := "pb001"
	result, err := c.Project("my_project").Items().HandleSuggestChoice(
		t.Context(),
		[]types.SuggestInfo{
			{IssueType: &issueType},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Suggestion choice applied.", *result.Message)
}

func TestTestItemScope_HandleSuggestChoice_Errors(t *testing.T) {
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
			body:       `{"errorCode": 4001, "message": "Invalid suggest choice payload."}`,
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
			_, err := testClient.Project("my_project").Items().HandleSuggestChoice(
				t.Context(),
				[]types.SuggestInfo{},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_LinkExternalIssues_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/issue/link")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		testItemIds := payload["testItemIds"].([]any)
		assert.Len(t, testItemIds, 1)

		issues, ok := payload["issues"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "JIRA-123", issues["ticketId"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"message":"Issues linked successfully."}]`))
	}))

	ticketID := "JIRA-123"
	btsURL := "https://jira.example.com"
	btsProject := "PROJ"
	url := "https://jira.example.com/browse/JIRA-123"

	result, err := c.Project("my_project").Items().LinkExternalIssues(
		t.Context(),
		types.LinkExternalIssueRQ{
			TestItemIds: []int64{42},
			Issues: types.ExternalSystemIssue{
				TicketId:   ticketID,
				BtsUrl:     btsURL,
				BtsProject: btsProject,
				Url:        url,
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].Message)
	assert.Equal(t, "Issues linked successfully.", *result[0].Message)
}

func TestTestItemScope_LinkExternalIssues_Errors(t *testing.T) {
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
			body:       `{"errorCode": 4001, "message": "Invalid link request."}`,
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
			_, err := testClient.Project("my_project").Items().LinkExternalIssues(
				t.Context(),
				types.LinkExternalIssueRQ{
					TestItemIds: []int64{42},
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_UnlinkExternalIssues_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/issue/unlink")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		testItemIds := payload["testItemIds"].([]any)
		assert.Len(t, testItemIds, 1)

		ticketIds := payload["ticketIds"].([]any)
		assert.Len(t, ticketIds, 1)
		assert.Equal(t, "JIRA-123", ticketIds[0])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"message":"Issues unlinked successfully."}]`))
	}))

	result, err := c.Project("my_project").Items().UnlinkExternalIssues(
		t.Context(),
		types.UnlinkExternalIssueRQ{
			TestItemIds: []int64{42},
			TicketIds:   []string{"JIRA-123"},
		},
	)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].Message)
	assert.Equal(t, "Issues unlinked successfully.", *result[0].Message)
}

func TestTestItemScope_UnlinkExternalIssues_Errors(t *testing.T) {
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
			body:       `{"errorCode": 4001, "message": "Invalid unlink request."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40440, "message": "Test item not found."}`,
			check:      client.IsNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Items().UnlinkExternalIssues(
				t.Context(),
				types.UnlinkExternalIssueRQ{
					TestItemIds: []int64{42},
					TicketIds:   []string{"JIRA-123"},
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestTestItemScope_Search_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/search")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": 101, "uuid": "suite-101", "name": "Suite A", "status": "PASSED", "type": "SUITE"},
				{"id": 102, "uuid": "test-102", "name": "Test A", "status": "FAILED", "type": "TEST"}
			],
			"page": {"number": 0, "size": 10, "totalElements": 2, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Items().Search(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Content)
	require.Len(t, *result.Content, 2)
	require.NotNil(t, (*result.Content)[0].Name)

	assert.Equal(t, "Suite A", *(*result.Content)[0].Name)
	assert.EqualValues(t, 2, *result.Page.TotalElements)
}

func TestTestItemScope_Search_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/v1/my_project/item/search")

		query := r.URL.Query()
		assert.Equal(t, "Login", query.Get("filter.cnt.name"))
		assert.Equal(t, "env:prod", query.Get("filter.has.compositeAttribute"))
		assert.Equal(t, "FAILED", query.Get("filter.in.status"))
		assert.Equal(t, "1", query.Get("page.page"))
		assert.Equal(t, "25", query.Get("page.size"))
		assert.Equal(t, "startTime,desc", query.Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 25, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Project("my_project").Items().Search(
		t.Context(),
		client.WithSearchItemName("Login"),
		client.WithSearchItemAttribute("env:prod"),
		client.WithSearchItemStatus("FAILED"),
		client.WithSearchItemPageNumber(1),
		client.WithSearchItemPageSize(25),
		client.WithSearchItemSort("startTime,desc"),
	)
	require.NoError(t, err)
}

func TestTestItemScope_Search_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Items().Search(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestTestItemScope_Search_EmptyPage(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/v1/my_project/item/search")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	result, err := c.Project("my_project").Items().Search(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Content)
	assert.Empty(t, *result.Content)
	assert.EqualValues(t, 0, *result.Page.TotalElements)
}

func TestTestItemScope_ListV2_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/v2")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": 201, "uuid": "suite-201", "name": "Suite B", "status": "PASSED", "type": "SUITE"},
				{"id": 202, "uuid": "test-202", "name": "Test B", "status": "FAILED", "type": "TEST"}
			],
			"page": {"number": 0, "size": 10, "totalElements": 2, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Items().ListV2(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Content)
	require.Len(t, *result.Content, 2)
	require.NotNil(t, (*result.Content)[0].Name)

	assert.Equal(t, "Suite B", *(*result.Content)[0].Name)
	assert.EqualValues(t, 2, *result.Page.TotalElements)
}

func TestTestItemScope_ListV2_WithFilterOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/v1/my_project/item/v2")

		query := r.URL.Query()
		assert.Equal(t, "FAILED", query.Get("filter.eq.status"))
		assert.Equal(t, "Auth Test", query.Get("filter.eq.name"))
		assert.Equal(t, "123", query.Get("filter.eq.launchId"))
		assert.Equal(t, "456", query.Get("filter.eq.parentId"))
		assert.Equal(t, "2", query.Get("page.page"))
		assert.Equal(t, "5", query.Get("page.size"))
		assert.Equal(t, "startTime,desc", query.Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 2, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Project("my_project").Items().ListV2(
		t.Context(),
		client.WithItemV2Status("FAILED"),
		client.WithItemV2Name("Auth Test"),
		client.WithItemV2LaunchID(123),
		client.WithItemV2ParentID(456),
		client.WithItemV2PageNumber(2),
		client.WithItemV2PageSize(5),
		client.WithItemV2Sort("startTime,desc"),
	)
	require.NoError(t, err)
}

func TestTestItemScope_ListV2_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Items().ListV2(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestTestItemScope_ListV2_EmptyPage(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/v1/my_project/item/v2")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	result, err := c.Project("my_project").Items().ListV2(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Content)
	assert.Empty(t, *result.Content)
	assert.EqualValues(t, 0, *result.Page.TotalElements)
}
