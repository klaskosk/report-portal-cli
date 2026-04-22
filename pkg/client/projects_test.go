package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	openapi_types "github.com/oapi-codegen/runtime/types"
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

		_, _ = w.Write([]byte(`["default_personal", "my_project", "another_project"]`))
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

		_, _ = w.Write([]byte(`[]`))
	}))

	names, err := c.Projects().ListNames(t.Context())
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestProjectsScope_ListNames_DoesNotRetryRawArrayResponse(t *testing.T) {
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
	assert.Equal(t, 1, callCount, "expected a single typed call")
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
	require.Len(t, *result.Content, 1)

	assert.Equal(t, "my_project", (*result.Content)[0].ProjectName)
	require.NotNil(t, (*result.Content)[0].EntryType)
	assert.Equal(t, "INTERNAL", *(*result.Content)[0].EntryType)
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

func TestProjectsScope_List_WithAdditionalOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		option   client.ListProjectsOption
		expected map[string]string
	}{
		{
			name:   "WithProjectType",
			option: client.WithProjectType("INTERNAL"),
			expected: map[string]string{
				"filter.eq.type": "INTERNAL",
			},
		},
		{
			name:   "WithProjectSort",
			option: client.WithProjectSort("name", "asc"),
			expected: map[string]string{
				"page.sort": "name,asc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				for key, want := range tt.expected {
					assert.Equal(t, want, query.Get(key), "query param %s", key)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				_, _ = w.Write([]byte(`{
					"content": [],
					"page": {"number": 0, "size": 20, "totalElements": 0, "totalPages": 0}
				}`))
			}))

			_, err := c.Projects().List(t.Context(), tt.option)
			require.NoError(t, err)
		})
	}
}

func TestProjectsScope_List_WithSortCleared(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[],"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}}`))
	}))

	_, err := c.Projects().List(t.Context(), client.WithProjectSort())
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

		_, _ = w.Write([]byte(`["my_project", "my_other_project"]`))
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

		_, _ = w.Write([]byte(`[]`))
	}))

	names, err := c.Projects().SearchNames(t.Context(), "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestProjectsScope_SearchNames_DoesNotRetryRawArrayResponse(t *testing.T) {
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
	assert.Equal(t, 1, callCount, "expected a single typed call")
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
	assert.Equal(t, "my_project", info.ProjectName)
	assert.Equal(t, int32(5), info.UsersQuantity)
	assert.Equal(t, int32(42), info.LaunchesQuantity)
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
		EntryType:   types.CreateProjectRQEntryTypeINTERNAL,
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
				EntryType:   types.CreateProjectRQEntryTypeINTERNAL,
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
		assertPath(t, r, "/api/v1/project")
		assert.Equal(t, []string{"42"}, r.URL.Query()["ids"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"successfullyDeleted":[42]}`))
	}))

	result, err := c.Projects().Delete(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.SuccessfullyDeleted)
	assert.Equal(t, []int64{42}, *result.SuccessfullyDeleted)
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
	require.Len(t, *result.Content, 1)

	assert.Equal(t, int64(1), (*result.Content)[0].Id)
	assert.Equal(t, "admin", (*result.Content)[0].UserId)
	assert.Equal(t, "admin@example.com", (*result.Content)[0].Email)
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

func TestProjectsScope_ListUsers_WithUserSort(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "login,desc", r.URL.Query().Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 20, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Projects().ListUsers(t.Context(), "my_project",
		client.WithUserSort("login", "desc"),
	)
	require.NoError(t, err)
}

func TestProjectsScope_ListUsers_WithSortCleared(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[],"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}}`))
	}))

	_, err := c.Projects().ListUsers(t.Context(), "my_project", client.WithUserSort())
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

func TestProjectsScope_Export_Success(t *testing.T) {
	t.Parallel()

	expectedBody := []byte("project-id,project-name\n1,my_project\n")

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/project/export")

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(expectedBody)
	}))

	data, err := c.Projects().Export(t.Context())
	require.NoError(t, err)
	assert.Equal(t, expectedBody, data)
}

func TestProjectsScope_Export_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Equal(t, "csv", query.Get("view"))
		assert.Equal(t, "my_project", query.Get("filter.eq.name"))
		assert.Equal(t, "INTERNAL", query.Get("filter.eq.type"))

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("data"))
	}))

	_, err := c.Projects().Export(t.Context(),
		client.WithExportView("csv"),
		client.WithExportFilterName("my_project"),
		client.WithExportFilterType("INTERNAL"),
	)
	require.NoError(t, err)
}

func TestProjectsScope_Export_WithExportFilterLastRun(t *testing.T) {
	t.Parallel()

	date := openapi_types.Date{Time: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)}

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2025-06-15", r.URL.Query().Get("filter.eq.lastRun"))

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("data"))
	}))

	_, err := c.Projects().Export(t.Context(),
		client.WithExportFilterLastRun(date),
	)
	require.NoError(t, err)
}

func TestProjectsScope_Export_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Projects().Export(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectsScope_GetAnalyzerStatus_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/project/analyzer/status")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"my_project": true, "other_project": false}`))
	}))

	result, err := c.Projects().GetAnalyzerStatus(t.Context())
	require.NoError(t, err)
	assert.True(t, result["my_project"])
	assert.False(t, result["other_project"])
}

func TestProjectsScope_GetAnalyzerStatus_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Projects().GetAnalyzerStatus(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectsScope_UpdateNotificationConfig_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPut)
		assertPath(t, r, "/api/v1/project/my_project/notification")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Contains(t, payload, "enabled")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Notification config updated."}`))
	}))

	enabled := true
	result, err := c.Projects().UpdateNotificationConfig(t.Context(), "my_project",
		types.ProjectNotificationConfigDTO{Enabled: &enabled},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Notification config updated.", *result.Message)
}

func TestProjectsScope_UpdateNotificationConfig_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found."}`))

	_, err := c.Projects().UpdateNotificationConfig(t.Context(), "nonexistent",
		types.ProjectNotificationConfigDTO{},
	)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectsScope_GetPreference_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/project/my_project/preference")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"projectId": 1,
			"filters": [{"id": 10, "name": "my-filter"}]
		}`))
	}))

	result, err := c.Projects().GetPreference(t.Context(), "my_project")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(1), result.ProjectId)
	require.NotNil(t, result.Filters)
	require.Len(t, *result.Filters, 1)
}

func TestProjectsScope_GetPreference_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found."}`))

	_, err := c.Projects().GetPreference(t.Context(), "nonexistent")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectsScope_AddPreference_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPut)
		assertPath(t, r, "/api/v1/project/my_project/preference/42")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Preference added."}`))
	}))

	result, err := c.Projects().AddPreference(t.Context(), "my_project", 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Preference added.", *result.Message)
}

func TestProjectsScope_AddPreference_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Filter not found."}`))

	_, err := c.Projects().AddPreference(t.Context(), "my_project", 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectsScope_RemovePreference_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/api/v1/project/my_project/preference/42")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Preference removed."}`))
	}))

	result, err := c.Projects().RemovePreference(t.Context(), "my_project", 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Preference removed.", *result.Message)
}

func TestProjectsScope_RemovePreference_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Filter not found."}`))

	_, err := c.Projects().RemovePreference(t.Context(), "my_project", 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectsScope_DeleteIndex_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/api/v1/project/my_project/index")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Index deleted."}`))
	}))

	result, err := c.Projects().DeleteIndex(t.Context(), "my_project")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Index deleted.", *result.Message)
}

func TestProjectsScope_DeleteIndex_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found."}`))

	_, err := c.Projects().DeleteIndex(t.Context(), "nonexistent")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectsScope_IndexData_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPut)
		assertPath(t, r, "/api/v1/project/my_project/index")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Index started."}`))
	}))

	result, err := c.Projects().IndexData(t.Context(), "my_project")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Index started.", *result.Message)
}

func TestProjectsScope_IndexData_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Project not found."}`))

	_, err := c.Projects().IndexData(t.Context(), "nonexistent")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectsScope_ListAssignableUsers_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/project/my_project/assignable")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{
					"id": 1,
					"userId": "user1",
					"email": "user1@example.com",
					"fullName": "User One"
				}
			],
			"page": {"number": 0, "size": 20, "totalElements": 1, "totalPages": 1}
		}`))
	}))

	result, err := c.Projects().ListAssignableUsers(t.Context(), "my_project")
	require.NoError(t, err)
	require.NotNil(t, result.Content)
	require.Len(t, *result.Content, 1)
	assert.Equal(t, "user1", (*result.Content)[0].UserId)
}

func TestProjectsScope_ListAssignableUsers_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Equal(t, "0", query.Get("page.page"))
		assert.Equal(t, "5", query.Get("page.size"))
		assert.Equal(t, "userId,asc", query.Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Projects().ListAssignableUsers(t.Context(), "my_project",
		client.WithAssignableUserPageNumber(0),
		client.WithAssignableUserPageSize(5),
		client.WithAssignableUserSort("userId", "asc"),
	)
	require.NoError(t, err)
}

func TestProjectsScope_ListAssignableUsers_WithSortCleared(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[],"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}}`))
	}))

	_, err := c.Projects().ListAssignableUsers(t.Context(), "my_project", client.WithAssignableUserSort())
	require.NoError(t, err)
}

func TestProjectsScope_ListAssignableUsers_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Projects().ListAssignableUsers(t.Context(), "my_project")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectsScope_ListUsernames_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/project/my_project/usernames")
		assert.Equal(t, "adm", r.URL.Query().Get("filter.cnt.users"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["admin", "admin2"]`))
	}))

	result, err := c.Projects().ListUsernames(t.Context(), "my_project", "adm")
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "admin", result[0])
	assert.Equal(t, "admin2", result[1])
}

func TestProjectsScope_ListUsernames_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Projects().ListUsernames(t.Context(), "my_project", "adm")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectsScope_SearchUsernames_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/project/my_project/usernames/search")
		assert.Equal(t, "admin", r.URL.Query().Get("term"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"login": "admin", "fullName": "Admin User"}
			],
			"page": {"number": 0, "size": 20, "totalElements": 1, "totalPages": 1}
		}`))
	}))

	result, err := c.Projects().SearchUsernames(t.Context(), "my_project", "admin")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Content)
	require.Len(t, *result.Content, 1)
}

func TestProjectsScope_SearchUsernames_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Equal(t, "user", query.Get("term"))
		assert.Equal(t, "0", query.Get("page.page"))
		assert.Equal(t, "10", query.Get("page.size"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Projects().SearchUsernames(t.Context(), "my_project", "user",
		client.WithSearchUsernamesPageNumber(0),
		client.WithSearchUsernamesPageSize(10),
	)
	require.NoError(t, err)
}

func TestProjectsScope_SearchUsernames_WithSort(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "admin", r.URL.Query().Get("term"))
		assert.Equal(t, "login,asc", r.URL.Query().Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 20, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Projects().SearchUsernames(t.Context(), "my_project", "admin",
		client.WithSearchUsernamesSort("login", "asc"),
	)
	require.NoError(t, err)
}

func TestProjectsScope_SearchUsernames_WithSortCleared(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[],"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}}`))
	}))

	_, err := c.Projects().SearchUsernames(t.Context(), "my_project", "admin",
		client.WithSearchUsernamesSort(),
	)
	require.NoError(t, err)
}

func TestProjectsScope_SearchUsernames_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Projects().SearchUsernames(t.Context(), "my_project", "admin")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectsScope_GetWidgetByCode_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/project/my_project/widget/launchesPerUser")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"result": [{"user": "admin", "count": 5}]}`))
	}))

	result, err := c.Projects().GetWidgetByCode(t.Context(), "my_project", "launchesPerUser")
	require.NoError(t, err)
	assert.Contains(t, result, "result")
}

func TestProjectsScope_GetWidgetByCode_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "3M", r.URL.Query().Get("interval"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{}`))
	}))

	_, err := c.Projects().GetWidgetByCode(t.Context(), "my_project", "launchesPerUser",
		client.WithWidgetInterval("3M"),
	)
	require.NoError(t, err)
}

func TestProjectsScope_GetWidgetByCode_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Widget not found."}`))

	_, err := c.Projects().GetWidgetByCode(t.Context(), "my_project", "nonexistent")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
