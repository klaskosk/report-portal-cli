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

func TestProjectGroupsScope_List_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/projects/my_project/groups")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"items": [
				{"id": 1, "name": "alpha", "slug": "alpha", "permission": "MEMBER"},
				{"id": 2, "name": "beta", "slug": "beta", "permission": "PROJECT_MANAGER"}
			],
			"limit": 300,
			"offset": 0,
			"total_count": 2
		}`))
		},
	))

	result, err := testClient.Project("my_project").Groups().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Items)
	require.Len(t, *result.Items, 2)
	assert.Equal(t, int64(1), *(*result.Items)[0].Id)
	assert.Equal(t, "alpha", *(*result.Items)[0].Name)
}

func TestProjectGroupsScope_List_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			query := httpReq.URL.Query()
			assert.Equal(t, "10", query.Get("limit"))
			assert.Equal(t, "20", query.Get("offset"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			const projectGroupsListBody = `{"items":[],"limit":10,"offset":20,"total_count":0}`

			_, _ = responseWriter.Write([]byte(projectGroupsListBody))
		},
	))

	_, err := testClient.Project("my_project").Groups().List(t.Context(),
		client.WithProjectGroupPageSize(10),
		client.WithProjectGroupPageOffset(20),
	)
	require.NoError(t, err)
}

func TestProjectGroupsScope_List_Empty(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			const emptyListBody = `{"items":[],"limit":300,"offset":0,"total_count":0}`

			_, _ = responseWriter.Write([]byte(emptyListBody))
		},
	))

	result, err := testClient.Project("my_project").Groups().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Items)
	assert.Empty(t, *result.Items)
}

func TestProjectGroupsScope_List_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Groups().List(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectGroupsScope_Get_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/projects/my_project/groups/42")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 42,
			"name": "my-group",
			"slug": "my-group",
			"permission": "MEMBER",
			"stats": {"users_count": 5}
		}`))
		},
	))

	group, err := testClient.Project("my_project").Groups().Get(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, group)
	assert.Equal(t, int64(42), *group.Id)
	assert.Equal(t, "my-group", *group.Name)
	require.NotNil(t, group.Stats)
	assert.Equal(t, int32(5), *group.Stats.UsersCount)
}

func TestProjectGroupsScope_Get_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Project("my_project").Groups().Get(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectGroupsScope_AddOrUpdate_Success(t *testing.T) {
	t.Parallel()

	perm := types.AddProjectToGroupByIdRequestPermissionMEMBER

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodPut)
			assertPath(t, httpReq, "/api/projects/my_project/groups/10")

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var req types.AddProjectToGroupByIdRequest
			if !assert.NoError(t, json.Unmarshal(body, &req)) {
				return
			}

			assert.Equal(t, &perm, req.Permission)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(
				`{"message": "Group has been added to project successfully."}`,
			))
		},
	))

	result, err := testClient.Project("my_project").
		Groups().
		AddOrUpdate(t.Context(), 10, types.AddProjectToGroupByIdRequest{
			Permission: &perm,
		})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Group has been added to project successfully.", *result.Message)
}

func TestProjectGroupsScope_AddOrUpdate_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4003, "message": "You do not have enough permissions."}`))

	_, err := c.Project("my_project").
		Groups().
		AddOrUpdate(t.Context(), 10, types.AddProjectToGroupByIdRequest{})
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestProjectGroupsScope_Delete_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodDelete)
			assertPath(t, httpReq, "/api/projects/my_project/groups/42")

			responseWriter.WriteHeader(http.StatusNoContent)
		},
	))

	err := testClient.Project("my_project").Groups().Delete(t.Context(), 42)
	require.NoError(t, err)
}

func TestProjectGroupsScope_Delete_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	err := c.Project("my_project").Groups().Delete(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
