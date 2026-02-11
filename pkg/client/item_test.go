package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
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

	item, err := c.Project("my_project").Items().GetByID(t.Context(), "42")
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
		itemID     string
		statusCode int
		body       string
		check      func(*testing.T, error)
	}{
		{
			name:       "NotFound",
			itemID:     "999",
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
			itemID:     "42",
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

func TestTestItemScope_GetByUUID_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/item/uuid/item-uuid-42")
		assertMethod(t, r, http.MethodGet)

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

	item, err := c.Project("my_project").Items().GetByUUID(t.Context(), "item-uuid-42")
	require.NoError(t, err)
	require.NotNil(t, item)
	require.NotNil(t, item.Uuid)
	require.NotNil(t, item.Status)

	assert.Equal(t, "item-uuid-42", *item.Uuid)
	assert.Equal(t, "FAILED", *item.Status)
}

func TestTestItemScope_GetByUUID_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40440, "message": "Test item not found."}`))

	_, err := c.Project("my_project").Items().GetByUUID(t.Context(), "unknown-uuid")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
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
	require.Len(t, result.Content, 2)
	require.NotNil(t, result.Content[0].Name)

	assert.Equal(t, "Suite A", *result.Content[0].Name)
	assert.Equal(t, 2, result.Page.TotalElements)
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
