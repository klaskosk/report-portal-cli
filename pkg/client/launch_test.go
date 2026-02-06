package client_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLaunchScope_GetByID_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/42")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"uuid": "abc-def-123",
			"name": "Test Launch",
			"status": "PASSED",
			"startTime": 1700000000000,
			"number": 1
		}`))
	}))

	launch, err := c.Project("my_project").Launches().GetByID(context.Background(), "42")
	require.NoError(t, err)

	assert.Equal(t, int64(42), launch.Id)
	assert.Equal(t, "Test Launch", launch.Name)
	assert.Equal(t, "PASSED", launch.Status)
}

func TestLaunchScope_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Launch '999' not found."}`))

	_, err := c.Project("my_project").Launches().GetByID(context.Background(), "999")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestLaunchScope_GetByID_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Launches().GetByID(context.Background(), "42")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestLaunchScope_GetByUUID_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/uuid/abc-def-123")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"uuid": "abc-def-123",
			"name": "Test Launch",
			"status": "FAILED",
			"startTime": 1700000000000,
			"number": 3
		}`))
	}))

	launch, err := c.Project("my_project").Launches().GetByUUID(context.Background(), "abc-def-123")
	require.NoError(t, err)

	assert.Equal(t, "abc-def-123", launch.Uuid)
	assert.Equal(t, "FAILED", launch.Status)
}

func TestLaunchScope_GetByUUID_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Launch not found."}`))

	_, err := c.Project("my_project").Launches().GetByUUID(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestLaunchScope_List_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": 1, "uuid": "uuid-1", "name": "Launch 1", "status": "PASSED", "startTime": 1700000000000, "number": 1},
				{"id": 2, "uuid": "uuid-2", "name": "Launch 2", "status": "FAILED", "startTime": 1700000001000, "number": 2}
			],
			"page": {"number": 1, "size": 10, "totalElements": 2, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Launches().List(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Content, 2)

	assert.Equal(t, "Launch 1", result.Content[0].Name)
	assert.Equal(t, 2, result.Page.TotalElements)
}

func TestLaunchScope_List_WithFilterOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		assert.Equal(t, "My Launch", query.Get("filter.eq.name"))
		assert.Equal(t, "PASSED", query.Get("filter.eq.status"))
		assert.Equal(t, "2", query.Get("page.page"))
		assert.Equal(t, "5", query.Get("page.size"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 2, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Project("my_project").Launches().List(context.Background(),
		client.WithName("My Launch"),
		client.WithStatus("PASSED"),
		client.WithPageNumber(2),
		client.WithPageSize(5),
	)
	require.NoError(t, err)
}

func TestLaunchScope_List_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4003, "message": "Access denied."}`))

	_, err := c.Project("my_project").Launches().List(context.Background())
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestLaunchScope_AuthorizationHeader(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Project("test").Launches().List(context.Background())
	require.NoError(t, err)
}
