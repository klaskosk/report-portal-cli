package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityEventsScope_Search_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/v1/activities/searches")
		assert.Equal(t, "20", r.URL.Query().Get("limit"))
		assert.Equal(t, "0", r.URL.Query().Get("offset"))
		assert.Equal(t, "DESC", r.URL.Query().Get("order"))
		assert.Equal(t, "createdAt", r.URL.Query().Get("sort"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
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
	}))

	result, err := c.ActivityEvents().Search(t.Context(), types.SearchCriteriaRQ{})
	require.NoError(t, err)
	require.NotNil(t, result.Items)
	require.Len(t, *result.Items, 1)

	event := (*result.Items)[0]
	assert.Equal(t, int64(1), event.Id)
	assert.Equal(t, "startLaunch", event.EventName)
	assert.Equal(t, "launch", event.ObjectType)
	assert.Equal(t, "admin", event.SubjectName)
}

func TestActivityEventsScope_Search_WithCriteria(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		criteria, ok := payload["search_criterias"].([]any) //nolint:misspell // API field name
		require.True(t, ok, "request body should contain search criteria array")
		require.Len(t, criteria, 1)

		first := criteria[0].(map[string]any)
		assert.Equal(t, "subjectName", first["filter_key"])
		assert.Equal(t, "EQ", first["operation"])
		assert.Equal(t, "admin", first["value"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"items": [],
			"limit": 20,
			"offset": 0,
			"total_count": 0
		}`))
	}))

	criteria := client.NewSearchCriteria(client.FilterByUser("admin"))

	_, err := c.ActivityEvents().Search(t.Context(), criteria)
	require.NoError(t, err)
}

func TestActivityEventsScope_Search_EmptyResults(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"items": [],
			"limit": 20,
			"offset": 0,
			"total_count": 0
		}`))
	}))

	result, err := c.ActivityEvents().Search(t.Context(), types.SearchCriteriaRQ{})
	require.NoError(t, err)
	require.NotNil(t, result.Items)
	assert.Empty(t, *result.Items)
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

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		assert.Equal(t, "50", query.Get("limit"))
		assert.Equal(t, "10", query.Get("offset"))
		assert.Equal(t, "createdAt", query.Get("sort"))
		assert.Equal(t, "DESC", query.Get("order"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"items": [],
			"limit": 50,
			"offset": 10,
			"total_count": 0
		}`))
	}))

	_, err := c.ActivityEvents().Search(t.Context(), types.SearchCriteriaRQ{},
		client.WithEventLimit(50),
		client.WithEventOffset(10),
		client.WithEventSort("createdAt"),
		client.WithEventOrder(client.OrderDesc),
	)
	require.NoError(t, err)
}

func TestActivityEventsScope_GetProjectSubjectNames_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/activities/my_project/subjectName")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["admin", "user1", "user2"]`))
	}))

	names, err := c.ActivityEvents().GetProjectSubjectNames(t.Context(), "my_project")
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
			name:     "FilterByUser",
			criteria: client.FilterByUser("admin"),
			wantKey:  "subjectName",
			wantOp:   types.EQ,
			wantVal:  "admin",
		},
		{
			name:     "FilterByObjectType",
			criteria: client.FilterByObjectType("launch"),
			wantKey:  "objectType",
			wantOp:   types.EQ,
			wantVal:  "launch",
		},
		{
			name:     "FilterByEventName",
			criteria: client.FilterByEventName("startLaunch"),
			wantKey:  "eventName",
			wantOp:   types.EQ,
			wantVal:  "startLaunch",
		},
		{
			name:     "FilterByProjectName",
			criteria: client.FilterByProjectName("my_project"),
			wantKey:  "projectName",
			wantOp:   types.EQ,
			wantVal:  "my_project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantKey, tt.criteria.FilterKey)
			require.NotNil(t, tt.criteria.Operation)
			assert.Equal(t, tt.wantOp, *tt.criteria.Operation)
			assert.Equal(t, tt.wantVal, tt.criteria.Value)
		})
	}
}

func TestNewSearchCriteria(t *testing.T) {
	t.Parallel()

	criteria := client.NewSearchCriteria(
		client.FilterByUser("admin"),
		client.FilterByObjectType("launch"),
	)

	require.NotNil(t, criteria.SearchCriterias)
	require.Len(t, *criteria.SearchCriterias, 2)

	assert.Equal(t, "subjectName", (*criteria.SearchCriterias)[0].FilterKey)
	assert.Equal(t, "objectType", (*criteria.SearchCriterias)[1].FilterKey)
}
