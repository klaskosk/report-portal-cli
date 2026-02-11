package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectsScope_Get_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"projectId": 1,
			"projectName": "my_project",
			"entryType": "INTERNAL",
			"configuration": {}
		}`))
	}))

	project, err := c.Projects().Get(t.Context(), "my_project")
	require.NoError(t, err)
	require.NotNil(t, project)

	assert.Equal(t, int64(1), project.ProjectId)
	assert.Equal(t, "my_project", project.ProjectName)
	assert.Equal(t, "INTERNAL", project.EntryType)
}

func TestProjectsScope_Get_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found."}`))

	_, err := c.Projects().Get(t.Context(), "nonexistent")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectsScope_Get_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Projects().Get(t.Context(), "my_project")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectsScope_ListNames_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": ["default_personal", "my_project", "another_project"],
			"page": {"number": 0, "size": 50, "totalElements": 3, "totalPages": 1}
		}`))
	}))

	names, err := c.Projects().ListNames(t.Context())
	require.NoError(t, err)
	require.Len(t, names, 3)
	assert.Equal(t, "default_personal", names[0])
	assert.Equal(t, "my_project", names[1])
}

func TestProjectsScope_ListNames_EmptyList(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 50, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	names, err := c.Projects().ListNames(t.Context())
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestProjectsScope_ListNames_FallsBackToRawArrayResponse(t *testing.T) {
	t.Parallel()

	callCount := 0

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/project/names")

		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`["default_personal","my_project"]`))
	}))

	names, err := c.Projects().ListNames(t.Context())
	require.NoError(t, err)
	require.Len(t, names, 2)
	assert.Equal(t, "default_personal", names[0])
	assert.Equal(t, "my_project", names[1])
	assert.Equal(t, 2, callCount, "expected typed call and raw fallback call")
}
