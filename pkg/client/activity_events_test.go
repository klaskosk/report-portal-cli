package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityEventsScope_Search_Success(t *testing.T) {
	t.Parallel()

	searchHandler := http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodPost)
			assertPath(t, httpReq, "/api/v1/activities/searches")
			assert.Equal(t, "20", httpReq.URL.Query().Get("limit"))
			assert.Equal(t, "0", httpReq.URL.Query().Get("offset"))
			assert.Equal(t, "DESC", httpReq.URL.Query().Get("order"))
			assert.Equal(t, "createdAt", httpReq.URL.Query().Get("sort"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"items": [
				{
					"id": 1,
					"event_name": "startLaunch",
					"object_type": "launch",
					"object_name": "Test Launch",
					"object_id": 10,
					"project_id": 1,
					"project_name": "my_project",
					"subject_id": "1",
					"subject_name": "admin",
					"subject_type": "user",
					"created_at": "2024-01-01T00:00:00Z"
				}
			],
			"limit": 20,
			"offset": 0,
			"total_count": 1
		}`))
		},
	)

	testClient := newTestClient(t, searchHandler)

	result, err := testClient.ActivityEvents().Search(t.Context(), types.SearchCriteriaRQ{})
	require.NoError(t, err)
	require.NotNil(t, result.Items)
	require.Len(t, result.Items, 1)

	event := result.Items[0]
	assert.Equal(t, int64(1), event.Id)
	assert.Equal(t, "startLaunch", event.EventName)
	assert.Equal(t, "launch", event.ObjectType)
	assert.Equal(t, "admin", event.SubjectName)
}

func TestActivityEventsScope_Search_WithCriteria(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertMethod(t, httpReq, http.MethodPost)

			payload, ok := decodeRequestJSONMap(t, httpReq)
			if !ok {
				return
			}

			criteria, okCriteria := payload["search_criterias"].([]any) //nolint:misspell // API field name
			if !assert.True(t, okCriteria, "request body should contain search criteria array") {
				return
			}

			if !assert.Len(t, criteria, 1) {
				return
			}

			first, okFirst := criteria[0].(map[string]any)
			if !assert.True(t, okFirst, "first criterion should be an object") {
				return
			}

			assert.Equal(t, "subjectName", first["filter_key"])
			assert.Equal(t, "EQ", first["operation"])
			assert.Equal(t, "admin", first["value"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"items": [],
			"limit": 20,
			"offset": 0,
			"total_count": 0
		}`))
		},
	))

	criteria := client.NewEventSearchCriteria(client.EventFilterByUser("admin"))

	_, err := testClient.ActivityEvents().Search(t.Context(), criteria)
	require.NoError(t, err)
}

func TestActivityEventsScope_Search_EmptyResults(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"items": [],
			"limit": 20,
			"offset": 0,
			"total_count": 0
		}`))
		},
	))

	result, err := testClient.ActivityEvents().Search(t.Context(), types.SearchCriteriaRQ{})
	require.NoError(t, err)
	require.NotNil(t, result.Items)
	assert.Empty(t, result.Items)
}

func TestActivityEventsScope_Search_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.ActivityEvents().Search(t.Context(), types.SearchCriteriaRQ{})
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestActivityEventsScope_Search_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			query := httpReq.URL.Query()

			assert.Equal(t, "50", query.Get("limit"))
			assert.Equal(t, "10", query.Get("offset"))
			assert.Equal(t, "createdAt", query.Get("sort"))
			assert.Equal(t, "DESC", query.Get("order"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"items": [],
			"limit": 50,
			"offset": 10,
			"total_count": 0
		}`))
		},
	))

	_, err := testClient.ActivityEvents().Search(t.Context(), types.SearchCriteriaRQ{},
		client.WithEventLimit(50),
		client.WithEventOffset(10),
		client.WithEventSort("createdAt"),
		client.WithEventOrder(client.EventOrderDesc),
	)
	require.NoError(t, err)
}

func TestActivityEventsScope_GetProjectSubjectNames_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/v1/activities/my_project/subjectName")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`["admin", "user1", "user2"]`))
		},
	))

	names, err := testClient.ActivityEvents().GetProjectSubjectNames(t.Context(), "my_project")
	require.NoError(t, err)
	assert.Equal(t, []string{"admin", "user1", "user2"}, names)
}

func TestActivityEventsScope_GetProjectSubjectNames_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.ActivityEvents().GetProjectSubjectNames(t.Context(), "my_project")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestSearchCriteriaHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		criteria types.SearchCriteria
		wantKey  string
		wantOp   types.SearchCriteriaOperation
		wantVal  string
	}{
		{
			name:     "EventFilterByUser",
			criteria: client.EventFilterByUser("admin"),
			wantKey:  "subjectName",
			wantOp:   types.EQ,
			wantVal:  "admin",
		},
		{
			name:     "EventFilterByObjectType",
			criteria: client.EventFilterByObjectType("launch"),
			wantKey:  "objectType",
			wantOp:   types.EQ,
			wantVal:  "launch",
		},
		{
			name:     "EventFilterByEventName",
			criteria: client.EventFilterByEventName("startLaunch"),
			wantKey:  "eventName",
			wantOp:   types.EQ,
			wantVal:  "startLaunch",
		},
		{
			name:     "EventFilterByProjectName",
			criteria: client.EventFilterByProjectName("my_project"),
			wantKey:  "projectName",
			wantOp:   types.EQ,
			wantVal:  "my_project",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.wantKey, testCase.criteria.FilterKey)
			require.NotNil(t, testCase.criteria.Operation)
			assert.Equal(t, testCase.wantOp, *testCase.criteria.Operation)
			assert.Equal(t, testCase.wantVal, testCase.criteria.Value)
		})
	}
}

func TestNewEventSearchCriteria(t *testing.T) {
	t.Parallel()

	criteria := client.NewEventSearchCriteria(
		client.EventFilterByUser("admin"),
		client.EventFilterByObjectType("launch"),
	)

	require.NotNil(t, criteria.SearchCriterias)
	require.Len(t, criteria.SearchCriterias, 2)

	assert.Equal(t, "subjectName", criteria.SearchCriterias[0].FilterKey)
	assert.Equal(t, "objectType", criteria.SearchCriterias[1].FilterKey)
}
