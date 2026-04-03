package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityScope_GetByID_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/my_project/activity/42")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"actionType": "updateItem",
			"objectType": "testItem",
			"objectName": "Login Test",
			"loggedObjectId": 100,
			"projectId": 1,
			"projectName": "my_project",
			"user": "admin",
			"lastModified": "2023-11-14T22:13:20Z"
		}`))
	}))

	activity, err := c.Project("my_project").Activities().GetByID(t.Context(), 42)
	require.NoError(t, err)

	assert.Equal(t, int64(42), activity.Id)
	assert.Equal(t, "updateItem", activity.ActionType)
	assert.Equal(t, "testItem", activity.ObjectType)
	require.NotNil(t, activity.ObjectName)
	assert.Equal(t, "Login Test", *activity.ObjectName)
	assert.Equal(t, "admin", activity.User)
}

func TestActivityScope_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Activity '999' not found."}`))

	_, err := c.Project("my_project").Activities().GetByID(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestActivityScope_GetByID_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Activities().GetByID(t.Context(), 42)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestActivityScope_ListForItem_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/my_project/activity/item/100")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{
					"id": 1,
					"created_at": "2024-03-29T12:29:54.772Z",
					"event_name": "updateItem",
					"object_type": "testItem",
					"object_name": "Login Test",
					"object_id": 100,
					"project_id": 1,
					"project_name": "my_project",
					"subject_id": "1",
					"subject_name": "admin",
					"subject_type": "user"
				}
			],
			"page": {"number": 1, "size": 20, "totalElements": 1, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Activities().ListForItem(t.Context(), 100)
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	event := result.Content[0]
	assert.Equal(t, int64(1), event.Id)
	assert.Equal(t, "updateItem", event.EventName)
	assert.Equal(t, int64(100), event.ObjectId)
}

func TestActivityScope_ListForItem_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		assert.Equal(t, "2", query.Get("page.page"))
		assert.Equal(t, "5", query.Get("page.size"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 2, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Project("my_project").Activities().ListForItem(t.Context(), 100,
		client.WithActivityPageNumber(2),
		client.WithActivityPageSize(5),
	)
	require.NoError(t, err)
}

func TestActivityScope_ListForItem_EmptyResults(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 20, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	result, err := c.Project("my_project").Activities().ListForItem(t.Context(), 100)
	require.NoError(t, err)
	assert.Empty(t, result.Content)
}

func TestActivityScope_ListForItem_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Activities().ListForItem(t.Context(), 100)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}
