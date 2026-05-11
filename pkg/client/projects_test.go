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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"projectId": 1,
			"projectName": "my_project",
			"entryType": "INTERNAL",
			"configuration": {}
		}`))
		},
	))

	project, err := testClient.Projects().Get(t.Context(), "my_project")
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(
				`["default_personal", "my_project", "another_project"]`,
			))
		},
	))

	names, err := testClient.Projects().ListNames(t.Context())
	require.NoError(t, err)
	require.Len(t, names, 3)
	assert.Equal(t, "default_personal", names[0])
	assert.Equal(t, "my_project", names[1])
}

func TestProjectsScope_ListNames_EmptyList(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[]`))
		},
	))

	names, err := testClient.Projects().ListNames(t.Context())
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestProjectsScope_ListNames_DoesNotRetryRawArrayResponse(t *testing.T) {
	t.Parallel()

	callCount := 0

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/v1/project/names")

			callCount++

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`["default_personal","my_project"]`))
		},
	))

	names, err := testClient.Projects().ListNames(t.Context())
	require.NoError(t, err)
	require.Len(t, names, 2)
	assert.Equal(t, "default_personal", names[0])
	assert.Equal(t, "my_project", names[1])
	assert.Equal(t, 1, callCount, "expected a single typed call")
}

func TestProjectsScope_List_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/v1/project/list")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
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
		},
	))

	result, err := testClient.Projects().List(t.Context())
	require.NoError(t, err)
	require.Len(t, *result.Content, 1)

	assert.Equal(t, "my_project", (*result.Content)[0].ProjectName)
	require.NotNil(t, (*result.Content)[0].EntryType)
	assert.Equal(t, "INTERNAL", *(*result.Content)[0].EntryType)
}

func TestProjectsScope_List_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			query := httpReq.URL.Query()
			assert.Equal(t, "1", query.Get("page.page"))
			assert.Equal(t, "5", query.Get("page.size"))
			assert.Equal(t, "test_project", query.Get("filter.eq.name"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
		},
	))

	_, err := testClient.Projects().List(t.Context(),
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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, http.HandlerFunc(
				func(responseWriter http.ResponseWriter, httpReq *http.Request) {
					query := httpReq.URL.Query()
					for key, want := range testCase.expected {
						assert.Equal(t, want, query.Get(key), "query param %s", key)
					}

					responseWriter.Header().Set("Content-Type", "application/json")
					responseWriter.WriteHeader(http.StatusOK)

					_, _ = responseWriter.Write([]byte(`{
					"content": [],
					"page": {"number": 0, "size": 20, "totalElements": 0, "totalPages": 0}
				}`))
				},
			))

			_, err := testClient.Projects().List(t.Context(), testCase.option)
			require.NoError(t, err)
		})
	}
}

func TestProjectsScope_List_WithSortCleared(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assert.Empty(t, httpReq.URL.Query().Get("page.sort"))

			responseWriter.Header().Set("Content-Type", "application/json")
			_, _ = responseWriter.Write([]byte(`{
				"content":[],
				"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}
			}`))
		},
	))

	_, err := testClient.Projects().List(t.Context(), client.WithProjectSort())
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assert.Equal(t, "my", httpReq.URL.Query().Get("term"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`["my_project", "my_other_project"]`))
		},
	))

	names, err := testClient.Projects().SearchNames(t.Context(), "my")
	require.NoError(t, err)
	require.Len(t, names, 2)
	assert.Equal(t, "my_project", names[0])
	assert.Equal(t, "my_other_project", names[1])
}

func TestProjectsScope_SearchNames_EmptyResults(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[]`))
		},
	))

	names, err := testClient.Projects().SearchNames(t.Context(), "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestProjectsScope_SearchNames_DoesNotRetryRawArrayResponse(t *testing.T) {
	t.Parallel()

	callCount := 0

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/v1/project/names/search")
			assert.Equal(t, "my", httpReq.URL.Query().Get("term"))

			callCount++

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`["my_project","my_other_project"]`))
		},
	))

	names, err := testClient.Projects().SearchNames(t.Context(), "my")
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assert.Equal(t, "3M", httpReq.URL.Query().Get("interval"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 1,
			"projectName": "my_project",
			"usersQuantity": 5,
			"launchesQuantity": 42,
			"entryType": "INTERNAL"
		}`))
		},
	))

	info, err := testClient.Projects().GetInfo(t.Context(), "my_project",
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodPost)
			assertPath(t, httpReq, "/api/v1/project")

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "new_project", payload["projectName"])
			assert.Equal(t, "INTERNAL", payload["entryType"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(`{"id":42}`))
		},
	))

	result, err := testClient.Projects().Create(t.Context(), types.CreateProjectRQ{
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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, jsonResponse(testCase.status, testCase.body))
			_, err := c.Projects().Create(t.Context(), types.CreateProjectRQ{
				ProjectName: "new_project",
				EntryType:   types.CreateProjectRQEntryTypeINTERNAL,
			})
			require.Error(t, err)
			assert.True(t, testCase.assertErr(err))
		})
	}
}

func TestProjectsScope_Delete_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodDelete)
			assertPath(t, httpReq, "/api/v1/project")
			assert.Equal(t, []string{"42"}, httpReq.URL.Query()["ids"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"successfullyDeleted":[42]}`))
		},
	))

	result, err := testClient.Projects().Delete(t.Context(), 42)
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodPut)
			assertPath(t, httpReq, "/api/v1/project/my_project")

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Contains(t, payload, "configuration")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Project updated."}`))
		},
	))

	result, err := testClient.Projects().Update(t.Context(), "my_project", types.UpdateProjectRQ{
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodPut)
			assertPath(t, httpReq, "/api/v1/project/my_project/assign")

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			userNames, ok := payload["userNames"].(map[string]any)
			if !assert.True(t, ok) {
				return
			}

			assert.Equal(t, "MEMBER", userNames["user1"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Users assigned."}`))
		},
	))

	result, err := testClient.Projects().AssignUsers(t.Context(), "my_project", types.AssignUsersRQ{
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodPut)
			assertPath(t, httpReq, "/api/v1/project/my_project/unassign")

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			userNames, ok := payload["userNames"].([]any)
			if !assert.True(t, ok) {
				return
			}

			assert.Contains(t, userNames, "user1")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Users unassigned."}`))
		},
	))

	result, err := testClient.Projects().UnassignUsers(
		t.Context(),
		"my_project",
		types.UnassignUsersRQ{
			UserNames: []string{"user1"},
		},
	)
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
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
		},
	))

	result, err := testClient.Projects().ListUsers(t.Context(), "my_project")
	require.NoError(t, err)
	require.Len(t, *result.Content, 1)

	assert.Equal(t, int64(1), (*result.Content)[0].Id)
	assert.Equal(t, "admin", (*result.Content)[0].UserId)
	assert.Equal(t, "admin@example.com", (*result.Content)[0].Email)
}

func TestProjectsScope_ListUsers_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			query := httpReq.URL.Query()
			assert.Equal(t, "0", query.Get("page.page"))
			assert.Equal(t, "10", query.Get("page.size"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
		},
	))

	_, err := testClient.Projects().ListUsers(t.Context(), "my_project",
		client.WithUserPageNumber(0),
		client.WithUserPageSize(10),
	)
	require.NoError(t, err)
}

func TestProjectsScope_ListUsers_WithUserSort(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assert.Equal(t, "login,desc", httpReq.URL.Query().Get("page.sort"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 20, "totalElements": 0, "totalPages": 0}
		}`))
		},
	))

	_, err := testClient.Projects().ListUsers(t.Context(), "my_project",
		client.WithUserSort("login", "desc"),
	)
	require.NoError(t, err)
}

func TestProjectsScope_ListUsers_WithSortCleared(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assert.Empty(t, httpReq.URL.Query().Get("page.sort"))

			responseWriter.Header().Set("Content-Type", "application/json")
			_, _ = responseWriter.Write(
				[]byte(
					`{"content":[],"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}}`,
				),
			)
		},
	))

	_, err := testClient.Projects().ListUsers(t.Context(), "my_project", client.WithUserSort())
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/v1/project/export")

			responseWriter.Header().Set("Content-Type", "application/octet-stream")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write(expectedBody)
		},
	))

	data, err := testClient.Projects().Export(t.Context())
	require.NoError(t, err)
	assert.Equal(t, expectedBody, data)
}

func TestProjectsScope_Export_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			query := httpReq.URL.Query()
			assert.Equal(t, "csv", query.Get("view"))
			assert.Equal(t, "my_project", query.Get("filter.eq.name"))
			assert.Equal(t, "INTERNAL", query.Get("filter.eq.type"))

			responseWriter.Header().Set("Content-Type", "application/octet-stream")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte("data"))
		},
	))

	_, err := testClient.Projects().Export(t.Context(),
		client.WithExportView("csv"),
		client.WithExportFilterName("my_project"),
		client.WithExportFilterType("INTERNAL"),
	)
	require.NoError(t, err)
}

func TestProjectsScope_Export_WithExportFilterLastRun(t *testing.T) {
	t.Parallel()

	date := openapi_types.Date{Time: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)}

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assert.Equal(t, "2025-06-15", httpReq.URL.Query().Get("filter.eq.lastRun"))

			responseWriter.Header().Set("Content-Type", "application/octet-stream")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte("data"))
		},
	))

	_, err := testClient.Projects().Export(t.Context(),
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/v1/project/analyzer/status")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{"my_project": true, "other_project": false}`))
		},
	))

	result, err := testClient.Projects().GetAnalyzerStatus(t.Context())
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodPut)
			assertPath(t, httpReq, "/api/v1/project/my_project/notification")

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Contains(t, payload, "enabled")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Notification config updated."}`))
		},
	))

	enabled := true
	result, err := testClient.Projects().UpdateNotificationConfig(t.Context(), "my_project",
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/v1/project/my_project/preference")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"projectId": 1,
			"filters": [{"id": 10, "name": "my-filter"}]
		}`))
		},
	))

	result, err := testClient.Projects().GetPreference(t.Context(), "my_project")
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodPut)
			assertPath(t, httpReq, "/api/v1/project/my_project/preference/42")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Preference added."}`))
		},
	))

	result, err := testClient.Projects().AddPreference(t.Context(), "my_project", 42)
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodDelete)
			assertPath(t, httpReq, "/api/v1/project/my_project/preference/42")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Preference removed."}`))
		},
	))

	result, err := testClient.Projects().RemovePreference(t.Context(), "my_project", 42)
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodDelete)
			assertPath(t, httpReq, "/api/v1/project/my_project/index")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Index deleted."}`))
		},
	))

	result, err := testClient.Projects().DeleteIndex(t.Context(), "my_project")
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

	indexHandler := http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodPut)
			assertPath(t, httpReq, "/api/v1/project/my_project/index")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Index started."}`))
		},
	)

	testClient := newTestClient(t, indexHandler)

	result, err := testClient.Projects().IndexData(t.Context(), "my_project")
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/v1/project/my_project/assignable")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
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
		},
	))

	result, err := testClient.Projects().ListAssignableUsers(t.Context(), "my_project")
	require.NoError(t, err)
	require.NotNil(t, result.Content)
	require.Len(t, *result.Content, 1)
	assert.Equal(t, "user1", (*result.Content)[0].UserId)
}

func TestProjectsScope_ListAssignableUsers_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			query := httpReq.URL.Query()
			assert.Equal(t, "0", query.Get("page.page"))
			assert.Equal(t, "5", query.Get("page.size"))
			assert.Equal(t, "userId,asc", query.Get("page.sort"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
		},
	))

	_, err := testClient.Projects().ListAssignableUsers(t.Context(), "my_project",
		client.WithAssignableUserPageNumber(0),
		client.WithAssignableUserPageSize(5),
		client.WithAssignableUserSort("userId", "asc"),
	)
	require.NoError(t, err)
}

func TestProjectsScope_ListAssignableUsers_WithSortCleared(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assert.Empty(t, httpReq.URL.Query().Get("page.sort"))

			responseWriter.Header().Set("Content-Type", "application/json")
			_, _ = responseWriter.Write([]byte(`{
				"content":[],
				"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}
			}`))
		},
	))

	_, err := testClient.Projects().
		ListAssignableUsers(t.Context(), "my_project", client.WithAssignableUserSort())
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/v1/project/my_project/usernames")
			assert.Equal(t, "adm", httpReq.URL.Query().Get("filter.cnt.users"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`["admin", "admin2"]`))
		},
	))

	result, err := testClient.Projects().ListUsernames(t.Context(), "my_project", "adm")
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/v1/project/my_project/usernames/search")
			assert.Equal(t, "admin", httpReq.URL.Query().Get("term"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [
				{"login": "admin", "fullName": "Admin User"}
			],
			"page": {"number": 0, "size": 20, "totalElements": 1, "totalPages": 1}
		}`))
		},
	))

	result, err := testClient.Projects().SearchUsernames(t.Context(), "my_project", "admin")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Content)
	require.Len(t, *result.Content, 1)
}

func TestProjectsScope_SearchUsernames_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			query := httpReq.URL.Query()
			assert.Equal(t, "user", query.Get("term"))
			assert.Equal(t, "0", query.Get("page.page"))
			assert.Equal(t, "10", query.Get("page.size"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
		},
	))

	_, err := testClient.Projects().SearchUsernames(t.Context(), "my_project", "user",
		client.WithSearchUsernamesPageNumber(0),
		client.WithSearchUsernamesPageSize(10),
	)
	require.NoError(t, err)
}

func TestProjectsScope_SearchUsernames_WithSort(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assert.Equal(t, "admin", httpReq.URL.Query().Get("term"))
			assert.Equal(t, "login,asc", httpReq.URL.Query().Get("page.sort"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 20, "totalElements": 0, "totalPages": 0}
		}`))
		},
	))

	_, err := testClient.Projects().SearchUsernames(t.Context(), "my_project", "admin",
		client.WithSearchUsernamesSort("login", "asc"),
	)
	require.NoError(t, err)
}

func TestProjectsScope_SearchUsernames_WithSortCleared(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(`{"content":[],"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}}`),
		)
	}))

	_, err := testClient.Projects().SearchUsernames(t.Context(), "my_project", "admin",
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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodGet)
			assertPath(t, httpReq, "/api/v1/project/my_project/widget/launchesPerUser")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
				"result": [{"user": "admin", "count": 5}]
			}`))
		},
	))

	result, err := testClient.Projects().GetWidgetByCode(
		t.Context(),
		"my_project",
		"launchesPerUser",
	)
	require.NoError(t, err)
	assert.Contains(t, result, "result")
}

func TestProjectsScope_GetWidgetByCode_WithOptions(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assert.Equal(t, "3M", httpReq.URL.Query().Get("interval"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{}`))
		},
	))

	_, err := testClient.Projects().GetWidgetByCode(t.Context(), "my_project", "launchesPerUser",
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
