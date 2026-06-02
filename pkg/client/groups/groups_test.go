package groups_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/clienttest"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupsScope_List_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodGet)
			testutil.AssertPath(t, httpReq, "/api/groups")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"items": [
				{"id": 1, "name": "group-one", "slug": "group-one"},
				{"id": 2, "name": "group-two", "slug": "group-two"}
			],
			"limit": 20,
			"offset": 0,
			"total_count": 2
		}`))
		},
	))

	result, err := testClient.Groups().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Items)
	require.Len(t, *result.Items, 2)
	assert.Equal(t, int64(1), *(*result.Items)[0].Id)
	assert.Equal(t, "group-one", *(*result.Items)[0].Name)
}

func TestGroupsScope_List_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			query := httpReq.URL.Query()
			assert.Equal(t, "10", query.Get("limit"))
			assert.Equal(t, "5", query.Get("offset"))
			assert.Equal(t, "name", query.Get("sort"))
			assert.Equal(t, "asc", query.Get("order"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			const listOptsBody = `{"items":[],"limit":10,"offset":5,"total_count":0}`

			_, _ = responseWriter.Write([]byte(listOptsBody))
		},
	))

	_, err := testClient.Groups().List(t.Context(),
		client.WithGroupPageSize(10),
		client.WithGroupOffset(5),
		client.WithGroupSort("name"),
		client.WithGroupOrder("asc"),
	)
	require.NoError(t, err)
}

func TestGroupsScope_List_Unauthorized(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Groups().List(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestGroupsScope_Get_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodGet)
			testutil.AssertPath(t, httpReq, "/api/groups/42")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 42,
			"name": "my-group",
			"slug": "my-group",
			"uuid": "550e8400-e29b-41d4-a716-446655440000",
			"stats": {"projects_count": 3, "users_count": 5}
		}`))
		},
	))

	group, err := testClient.Groups().Get(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, group)
	assert.Equal(t, int64(42), *group.Id)
	assert.Equal(t, "my-group", *group.Name)
	require.NotNil(t, group.Stats)
	assert.Equal(t, int32(3), *group.Stats.ProjectsCount)
	assert.Equal(t, int32(5), *group.Stats.UsersCount)
}

func TestGroupsScope_Get_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().Get(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_Create_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodPost)
			testutil.AssertPath(t, httpReq, "/api/groups")

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "new-group", payload["name"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)

			const createGroupBody = `{"id": 10, "name": "new-group", "slug": "new-group"}`

			_, _ = responseWriter.Write([]byte(createGroupBody))
		},
	))

	result, err := testClient.Groups().Create(t.Context(), types.CreateGroupRequest{
		Name: "new-group",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(10), *result.Id)
	assert.Equal(t, "new-group", *result.Name)
}

func TestGroupsScope_Create_Conflict(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusConflict,
		`{"errorCode": 40910, "message": "Group already exists."}`))

	_, err := c.Groups().Create(t.Context(), types.CreateGroupRequest{
		Name: "existing-group",
	})
	require.Error(t, err)
	assert.True(t, client.IsConflict(err))
}

func TestGroupsScope_Update_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodPut)
			testutil.AssertPath(t, httpReq, "/api/groups/42")

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "updated-group", payload["name"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message": "Group updated."}`))
		},
	))

	result, err := testClient.Groups().Update(t.Context(), 42, types.UpdateGroupRequest{
		Name: "updated-group",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Group updated.", *result.Message)
}

func TestGroupsScope_Update_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().Update(t.Context(), 999, types.UpdateGroupRequest{
		Name: "updated-group",
	})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_Delete_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodDelete)
			testutil.AssertPath(t, httpReq, "/api/groups/42")

			responseWriter.WriteHeader(http.StatusNoContent)
		},
	))

	err := testClient.Groups().Delete(t.Context(), 42)
	require.NoError(t, err)
}

func TestGroupsScope_Delete_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	err := c.Groups().Delete(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_ListProjects_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodGet)
			testutil.AssertPath(t, httpReq, "/api/groups/1/projects")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"items": [
				{"id": 10, "name": "project-alpha", "permission": "MEMBER"},
				{"id": 20, "name": "project-beta", "permission": "ADMIN"}
			],
			"limit": 20,
			"offset": 0,
			"total_count": 2
		}`))
		},
	))

	result, err := testClient.Groups().ListProjects(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Items)
	require.Len(t, *result.Items, 2)
	assert.Equal(t, int64(10), *(*result.Items)[0].Id)
	assert.Equal(t, "project-alpha", *(*result.Items)[0].Name)
}

func TestGroupsScope_ListProjects_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			query := httpReq.URL.Query()
			assert.Equal(t, "5", query.Get("limit"))
			assert.Equal(t, "10", query.Get("offset"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(
				`{"items":[],"limit":5,"offset":10,"total_count":0}`,
			))
		},
	))

	_, err := testClient.Groups().ListProjects(t.Context(), 1,
		client.WithGroupProjectPageSize(5),
		client.WithGroupProjectOffset(10),
	)
	require.NoError(t, err)
}

func TestGroupsScope_ListProjects_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().ListProjects(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_GetProject_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodGet)
			testutil.AssertPath(t, httpReq, "/api/groups/1/projects/10")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 10,
			"name": "project-alpha",
			"permission": "MEMBER"
		}`))
		},
	))

	result, err := testClient.Groups().GetProject(t.Context(), 1, 10)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(10), *result.Id)
	assert.Equal(t, "project-alpha", *result.Name)
}

func TestGroupsScope_GetProject_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found in group."}`))

	_, err := c.Groups().GetProject(t.Context(), 1, 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_AddProject_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodPut)
			testutil.AssertPath(t, httpReq, "/api/groups/1/projects/10")

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "MEMBER", payload["permission"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message": "Project added."}`))
		},
	))

	permission := types.AddProjectToGroupByIdRequestPermission("MEMBER")
	result, err := testClient.Groups().AddProject(
		t.Context(), 1, 10,
		types.AddProjectToGroupByIdRequest{
			Permission: &permission,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Project added.", *result.Message)
}

func TestGroupsScope_AddProject_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().AddProject(t.Context(), 999, 10, types.AddProjectToGroupByIdRequest{})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_RemoveProject_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodDelete)
			testutil.AssertPath(t, httpReq, "/api/groups/1/projects/10")

			responseWriter.WriteHeader(http.StatusNoContent)
		},
	))

	err := testClient.Groups().RemoveProject(t.Context(), 1, 10)
	require.NoError(t, err)
}

func TestGroupsScope_RemoveProject_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found in group."}`))

	err := c.Groups().RemoveProject(t.Context(), 1, 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_ListUsers_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodGet)
			testutil.AssertPath(t, httpReq, "/api/groups/1/users")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"items": [
				{"id": 100, "full_name": "Alice", "email": "alice@example.com"},
				{"id": 200, "full_name": "Bob", "email": "bob@example.com"}
			],
			"limit": 20,
			"offset": 0,
			"total_count": 2
		}`))
		},
	))

	result, err := testClient.Groups().ListUsers(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Items)
	require.Len(t, *result.Items, 2)
	assert.Equal(t, int64(100), *(*result.Items)[0].Id)
	assert.Equal(t, "Alice", *(*result.Items)[0].FullName)
}

func TestGroupsScope_ListUsers_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			query := httpReq.URL.Query()
			assert.Equal(t, "15", query.Get("limit"))
			assert.Equal(t, "30", query.Get("offset"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(
				`{"items":[],"limit":15,"offset":30,"total_count":0}`,
			))
		},
	))

	_, err := testClient.Groups().ListUsers(t.Context(), 1,
		client.WithGroupUserPageSize(15),
		client.WithGroupUserOffset(30),
	)
	require.NoError(t, err)
}

func TestGroupsScope_ListUsers_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().ListUsers(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_GetUser_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodGet)
			testutil.AssertPath(t, httpReq, "/api/groups/1/users/100")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 100,
			"full_name": "Alice",
			"email": "alice@example.com",
			"instance_role": "USER",
			"active": true
		}`))
		},
	))

	result, err := testClient.Groups().GetUser(t.Context(), 1, 100)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(100), *result.Id)
	assert.Equal(t, "Alice", *result.FullName)
	assert.True(t, *result.Active)
}

func TestGroupsScope_GetUser_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User not found in group."}`))

	_, err := c.Groups().GetUser(t.Context(), 1, 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_AddUser_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodPut)
			testutil.AssertPath(t, httpReq, "/api/groups/1/users/100")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message": "User added."}`))
		},
	))

	result, err := testClient.Groups().AddUser(t.Context(), 1, 100)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "User added.", *result.Message)
}

func TestGroupsScope_AddUser_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().AddUser(t.Context(), 999, 100)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_RemoveUser_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertMethod(t, httpReq, http.MethodDelete)
			testutil.AssertPath(t, httpReq, "/api/groups/1/users/100")

			responseWriter.WriteHeader(http.StatusNoContent)
		},
	))

	err := testClient.Groups().RemoveUser(t.Context(), 1, 100)
	require.NoError(t, err)
}

func TestGroupsScope_RemoveUser_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User not found in group."}`))

	err := c.Groups().RemoveUser(t.Context(), 1, 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
