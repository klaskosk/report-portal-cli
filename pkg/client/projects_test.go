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

func TestProjectsScope_List_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/project/list")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{
					"id": 1,
					"projectName": "my_project",
					"entryType": "INTERNAL",
					"launchesQuantity": 10,
					"usersQuantity": 3
				}
			],
			"page": {"number": 0, "size": 20, "totalElements": 1, "totalPages": 1}
		}`))
	}))

	result, err := c.Projects().List(t.Context())
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	assert.Equal(t, "my_project", *result.Content[0].ProjectName)
	assert.Equal(t, "INTERNAL", *result.Content[0].EntryType)
}

func TestProjectsScope_List_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Equal(t, "1", query.Get("page.page"))
		assert.Equal(t, "5", query.Get("page.size"))
		assert.Equal(t, "test_project", query.Get("filter.eq.name"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Projects().List(t.Context(),
		client.WithProjectPageNumber(1),
		client.WithProjectPageSize(5),
		client.WithProjectName("test_project"),
	)
	require.NoError(t, err)
}

func TestProjectsScope_List_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Projects().List(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectsScope_SearchNames_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assert.Equal(t, "my", r.URL.Query().Get("term"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": ["my_project", "my_other_project"],
			"page": {"number": 0, "size": 50, "totalElements": 2, "totalPages": 1}
		}`))
	}))

	names, err := c.Projects().SearchNames(t.Context(), "my")
	require.NoError(t, err)
	require.Len(t, names, 2)
	assert.Equal(t, "my_project", names[0])
	assert.Equal(t, "my_other_project", names[1])
}

func TestProjectsScope_SearchNames_EmptyResults(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 50, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	names, err := c.Projects().SearchNames(t.Context(), "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestProjectsScope_SearchNames_FallsBackToRawArrayResponse(t *testing.T) {
	t.Parallel()

	callCount := 0

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/project/names/search")
		assert.Equal(t, "my", r.URL.Query().Get("term"))

		callCount++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`["my_project","my_other_project"]`))
	}))

	names, err := c.Projects().SearchNames(t.Context(), "my")
	require.NoError(t, err)
	require.Len(t, names, 2)
	assert.Equal(t, "my_project", names[0])
	assert.Equal(t, "my_other_project", names[1])
	assert.Equal(t, 2, callCount, "expected typed call and raw fallback call")
}

func TestProjectsScope_SearchNames_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Projects().SearchNames(t.Context(), "my")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectsScope_GetInfo_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assert.Equal(t, "3M", r.URL.Query().Get("interval"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 1,
			"projectName": "my_project",
			"usersQuantity": 5,
			"launchesQuantity": 42,
			"entryType": "INTERNAL"
		}`))
	}))

	info, err := c.Projects().GetInfo(t.Context(), "my_project",
		client.WithProjectInfoInterval("3M"),
	)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "my_project", *info.ProjectName)
	assert.Equal(t, int32(5), *info.UsersQuantity)
	assert.Equal(t, int32(42), *info.LaunchesQuantity)
}

func TestProjectsScope_GetInfo_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found."}`))

	_, err := c.Projects().GetInfo(t.Context(), "nonexistent")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectsScope_Create_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/v1/project")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "new_project", payload["projectName"])
		assert.Equal(t, "INTERNAL", payload["entryType"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":42}`))
	}))

	result, err := c.Projects().Create(t.Context(), types.CreateProjectRQ{
		ProjectName: "new_project",
		EntryType:   types.INTERNAL,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, int64(42), *result.Id)
}

func TestProjectsScope_Create_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    int
		body      string
		assertErr func(error) bool
	}{
		{
			name:      "BadRequest",
			status:    http.StatusBadRequest,
			body:      `{"errorCode": 4001, "message": "Invalid project name."}`,
			assertErr: client.IsBadRequest,
		},
		{
			name:      "Unauthorized",
			status:    http.StatusUnauthorized,
			body:      `{"errorCode": 4001, "message": "Full authentication is required."}`,
			assertErr: client.IsUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, jsonResponse(tt.status, tt.body))
			_, err := c.Projects().Create(t.Context(), types.CreateProjectRQ{
				ProjectName: "new_project",
				EntryType:   types.INTERNAL,
			})
			require.Error(t, err)
			assert.True(t, tt.assertErr(err))
		})
	}
}

func TestProjectsScope_Delete_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/api/v1/project/42")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Project deleted."}`))
	}))

	result, err := c.Projects().Delete(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Project deleted.", *result.Message)
}

func TestProjectsScope_Delete_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found."}`))

	_, err := c.Projects().Delete(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectsScope_Update_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPut)
		assertPath(t, r, "/api/v1/project/my_project")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Contains(t, payload, "configuration")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Project updated."}`))
	}))

	result, err := c.Projects().Update(t.Context(), "my_project", types.UpdateProjectRQ{
		Configuration: &types.ProjectConfigurationUpdate{
			Attributes: map[string]string{"analyzer.isAutoAnalyzerEnabled": "false"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Project updated.", *result.Message)
}

func TestProjectsScope_Update_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found."}`))

	_, err := c.Projects().Update(t.Context(), "nonexistent", types.UpdateProjectRQ{})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectsScope_AssignUsers_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPut)
		assertPath(t, r, "/api/v1/project/my_project/assign")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		userNames := payload["userNames"].(map[string]any)
		assert.Equal(t, "MEMBER", userNames["user1"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Users assigned."}`))
	}))

	result, err := c.Projects().AssignUsers(t.Context(), "my_project", types.AssignUsersRQ{
		UserNames: map[string]string{"user1": "MEMBER"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Users assigned.", *result.Message)
}

func TestProjectsScope_AssignUsers_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Projects().AssignUsers(t.Context(), "my_project", types.AssignUsersRQ{
		UserNames: map[string]string{"user1": "MEMBER"},
	})
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectsScope_UnassignUsers_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPut)
		assertPath(t, r, "/api/v1/project/my_project/unassign")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		userNames := payload["userNames"].([]any)
		assert.Contains(t, userNames, "user1")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Users unassigned."}`))
	}))

	result, err := c.Projects().UnassignUsers(t.Context(), "my_project", types.UnassignUsersRQ{
		UserNames: []string{"user1"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Users unassigned.", *result.Message)
}

func TestProjectsScope_UnassignUsers_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Projects().UnassignUsers(t.Context(), "my_project", types.UnassignUsersRQ{
		UserNames: []string{"user1"},
	})
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectsScope_ListUsers_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{
					"id": 1,
					"userId": "admin",
					"email": "admin@example.com",
					"fullName": "Admin User"
				}
			],
			"page": {"number": 0, "size": 20, "totalElements": 1, "totalPages": 1}
		}`))
	}))

	result, err := c.Projects().ListUsers(t.Context(), "my_project")
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	assert.Equal(t, int64(1), result.Content[0].Id)
	assert.Equal(t, "admin", result.Content[0].UserId)
	assert.Equal(t, "admin@example.com", result.Content[0].Email)
}

func TestProjectsScope_ListUsers_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Equal(t, "0", query.Get("page.page"))
		assert.Equal(t, "10", query.Get("page.size"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Projects().ListUsers(t.Context(), "my_project",
		client.WithUserPageNumber(0),
		client.WithUserPageSize(10),
	)
	require.NoError(t, err)
}

func TestProjectsScope_ListUsers_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Projects().ListUsers(t.Context(), "my_project")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}
