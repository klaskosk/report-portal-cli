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

func TestGroupsScope_List_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/groups")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"items": [
				{"id": 1, "name": "group-one", "slug": "group-one"},
				{"id": 2, "name": "group-two", "slug": "group-two"}
			],
			"limit": 20,
			"offset": 0,
			"total_count": 2
		}`))
	}))

	result, err := c.Groups().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Items)
	require.Len(t, *result.Items, 2)
	assert.Equal(t, int64(1), *(*result.Items)[0].Id)
	assert.Equal(t, "group-one", *(*result.Items)[0].Name)
}

func TestGroupsScope_List_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Equal(t, "10", query.Get("limit"))
		assert.Equal(t, "5", query.Get("offset"))
		assert.Equal(t, "name", query.Get("sort"))
		assert.Equal(t, "asc", query.Get("order"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[],"limit":10,"offset":5,"total_count":0}`))
	}))

	_, err := c.Groups().List(t.Context(),
		client.WithGroupPageSize(10),
		client.WithGroupOffset(5),
		client.WithGroupSort("name"),
		client.WithGroupOrder("asc"),
	)
	require.NoError(t, err)
}

func TestGroupsScope_List_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Groups().List(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestGroupsScope_Get_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/groups/42")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"name": "my-group",
			"slug": "my-group",
			"uuid": "550e8400-e29b-41d4-a716-446655440000",
			"stats": {"projects_count": 3, "users_count": 5}
		}`))
	}))

	group, err := c.Groups().Get(t.Context(), 42)
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

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().Get(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_Create_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/groups")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "new-group", payload["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": 10, "name": "new-group", "slug": "new-group"}`))
	}))

	result, err := c.Groups().Create(t.Context(), types.CreateGroupRequest{
		Name: "new-group",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(10), *result.Id)
	assert.Equal(t, "new-group", *result.Name)
}

func TestGroupsScope_Create_Conflict(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusConflict,
		`{"errorCode": 40910, "message": "Group already exists."}`))

	_, err := c.Groups().Create(t.Context(), types.CreateGroupRequest{
		Name: "existing-group",
	})
	require.Error(t, err)
	assert.True(t, client.IsConflict(err))
}

func TestGroupsScope_Update_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPut)
		assertPath(t, r, "/api/groups/42")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "updated-group", payload["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Group updated."}`))
	}))

	result, err := c.Groups().Update(t.Context(), 42, types.UpdateGroupRequest{
		Name: "updated-group",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Group updated.", *result.Message)
}

func TestGroupsScope_Update_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().Update(t.Context(), 999, types.UpdateGroupRequest{
		Name: "updated-group",
	})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_Delete_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/api/groups/42")

		w.WriteHeader(http.StatusNoContent)
	}))

	err := c.Groups().Delete(t.Context(), 42)
	require.NoError(t, err)
}

func TestGroupsScope_Delete_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	err := c.Groups().Delete(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_ListProjects_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/groups/1/projects")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"items": [
				{"id": 10, "name": "project-alpha", "permission": "MEMBER"},
				{"id": 20, "name": "project-beta", "permission": "ADMIN"}
			],
			"limit": 20,
			"offset": 0,
			"total_count": 2
		}`))
	}))

	result, err := c.Groups().ListProjects(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Items)
	require.Len(t, *result.Items, 2)
	assert.Equal(t, int64(10), *(*result.Items)[0].Id)
	assert.Equal(t, "project-alpha", *(*result.Items)[0].Name)
}

func TestGroupsScope_ListProjects_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Equal(t, "5", query.Get("limit"))
		assert.Equal(t, "10", query.Get("offset"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[],"limit":5,"offset":10,"total_count":0}`))
	}))

	_, err := c.Groups().ListProjects(t.Context(), 1,
		client.WithGroupProjectPageSize(5),
		client.WithGroupProjectOffset(10),
	)
	require.NoError(t, err)
}

func TestGroupsScope_ListProjects_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().ListProjects(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_GetProject_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/groups/1/projects/10")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 10,
			"name": "project-alpha",
			"permission": "MEMBER"
		}`))
	}))

	result, err := c.Groups().GetProject(t.Context(), 1, 10)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(10), *result.Id)
	assert.Equal(t, "project-alpha", *result.Name)
}

func TestGroupsScope_GetProject_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found in group."}`))

	_, err := c.Groups().GetProject(t.Context(), 1, 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_AddProject_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPut)
		assertPath(t, r, "/api/groups/1/projects/10")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "MEMBER", payload["permission"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Project added."}`))
	}))

	permission := types.AddProjectToGroupByIdRequestPermission("MEMBER")
	result, err := c.Groups().AddProject(t.Context(), 1, 10, types.AddProjectToGroupByIdRequest{
		Permission: &permission,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Project added.", *result.Message)
}

func TestGroupsScope_AddProject_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().AddProject(t.Context(), 999, 10, types.AddProjectToGroupByIdRequest{})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_RemoveProject_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/api/groups/1/projects/10")

		w.WriteHeader(http.StatusNoContent)
	}))

	err := c.Groups().RemoveProject(t.Context(), 1, 10)
	require.NoError(t, err)
}

func TestGroupsScope_RemoveProject_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found in group."}`))

	err := c.Groups().RemoveProject(t.Context(), 1, 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_ListUsers_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/groups/1/users")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"items": [
				{"id": 100, "full_name": "Alice", "email": "alice@example.com"},
				{"id": 200, "full_name": "Bob", "email": "bob@example.com"}
			],
			"limit": 20,
			"offset": 0,
			"total_count": 2
		}`))
	}))

	result, err := c.Groups().ListUsers(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Items)
	require.Len(t, *result.Items, 2)
	assert.Equal(t, int64(100), *(*result.Items)[0].Id)
	assert.Equal(t, "Alice", *(*result.Items)[0].FullName)
}

func TestGroupsScope_ListUsers_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Equal(t, "15", query.Get("limit"))
		assert.Equal(t, "30", query.Get("offset"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[],"limit":15,"offset":30,"total_count":0}`))
	}))

	_, err := c.Groups().ListUsers(t.Context(), 1,
		client.WithGroupUserPageSize(15),
		client.WithGroupUserOffset(30),
	)
	require.NoError(t, err)
}

func TestGroupsScope_ListUsers_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().ListUsers(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_GetUser_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/groups/1/users/100")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 100,
			"full_name": "Alice",
			"email": "alice@example.com",
			"instance_role": "USER",
			"active": true
		}`))
	}))

	result, err := c.Groups().GetUser(t.Context(), 1, 100)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(100), *result.Id)
	assert.Equal(t, "Alice", *result.FullName)
	assert.True(t, *result.Active)
}

func TestGroupsScope_GetUser_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User not found in group."}`))

	_, err := c.Groups().GetUser(t.Context(), 1, 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_AddUser_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPut)
		assertPath(t, r, "/api/groups/1/users/100")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "User added."}`))
	}))

	result, err := c.Groups().AddUser(t.Context(), 1, 100)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "User added.", *result.Message)
}

func TestGroupsScope_AddUser_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Group not found."}`))

	_, err := c.Groups().AddUser(t.Context(), 999, 100)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestGroupsScope_RemoveUser_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/api/groups/1/users/100")

		w.WriteHeader(http.StatusNoContent)
	}))

	err := c.Groups().RemoveUser(t.Context(), 1, 100)
	require.NoError(t, err)
}

func TestGroupsScope_RemoveUser_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User not found in group."}`))

	err := c.Groups().RemoveUser(t.Context(), 1, 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
