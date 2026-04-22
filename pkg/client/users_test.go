package client_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsersScope_GetMyself_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 1,
			"userId": "superadmin",
			"email": "admin@example.com",
			"fullName": "Super Admin",
			"userRole": "ADMINISTRATOR",
			"assignedProjects": {
				"default_personal": {"entryType": "PERSONAL", "projectRole": "PROJECT_MANAGER"}
			}
		}`))
	}))

	user, err := c.Users().GetMyself(t.Context())
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, int64(1), user.Id)
	assert.Equal(t, "superadmin", user.UserId)
	assert.Equal(t, "admin@example.com", user.Email)
}

func TestUsersScope_GetMyself_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Users().GetMyself(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestUsersScope_GetByLogin_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 2,
			"userId": "testuser",
			"email": "test@example.com",
			"userRole": "USER"
		}`))
	}))

	user, err := c.Users().GetByLogin(t.Context(), "testuser")
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, "testuser", user.UserId)
}

func TestUsersScope_GetByLogin_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/testuser")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errorCode": 4031, "message": "forbidden"}`))
	}))

	_, err := c.Users().GetByLogin(t.Context(), "testuser")
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestUsersScope_GetByLogin_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User not found."}`))

	_, err := c.Users().GetByLogin(t.Context(), "nonexistent")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestUsersScope_ListProjects_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"default_personal": {"entryType": "PERSONAL", "projectRole": "PROJECT_MANAGER"},
			"my_project": {"entryType": "INTERNAL", "projectRole": "MEMBER"}
		}`))
	}))

	projects, err := c.Users().ListProjects(t.Context(), "superadmin")
	require.NoError(t, err)
	require.Len(t, projects, 2)
	assert.Contains(t, projects, "default_personal")
	assert.Contains(t, projects, "my_project")
}

func TestUsersScope_ListProjects_EmptyList(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{}`))
	}))

	projects, err := c.Users().ListProjects(t.Context(), "newuser")
	require.NoError(t, err)
	assert.Empty(t, projects)
}

func TestUsersScope_ListProjects_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Users().ListProjects(t.Context(), "superadmin")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestUsersScope_List_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/all")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": 1, "userId": "admin", "email": "admin@example.com", "userRole": "ADMINISTRATOR"},
				{"id": 2, "userId": "user1", "email": "user1@example.com", "userRole": "USER"}
			],
			"page": {"number": 0, "size": 20, "totalElements": 2, "totalPages": 1}
		}`))
	}))

	page, err := c.Users().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, page)
	require.NotNil(t, page.Content)
	assert.Len(t, *page.Content, 2)
	assert.Equal(t, "admin", (*page.Content)[0].UserId)
	assert.Equal(t, "user1", (*page.Content)[1].UserId)
}

func TestUsersScope_List_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/users/all")

		assert.Equal(t, "ADMINISTRATOR", r.URL.Query().Get("filter.eq.role"))
		assert.Equal(t, "0", r.URL.Query().Get("page.page"))
		assert.Equal(t, "10", r.URL.Query().Get("page.size"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [{"id": 1, "userId": "admin", "email": "admin@example.com", "userRole": "ADMINISTRATOR"}],
			"page": {"number": 0, "size": 10, "totalElements": 1, "totalPages": 1}
		}`))
	}))

	page, err := c.Users().List(t.Context(),
		client.WithUsersRole("ADMINISTRATOR"),
		client.WithUsersPageNumber(0),
		client.WithUsersPageSize(10),
	)
	require.NoError(t, err)
	require.NotNil(t, page)
	require.NotNil(t, page.Content)
	assert.Len(t, *page.Content, 1)
}

func TestUsersScope_List_WithAdditionalOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		option   client.ListUsersOption
		expected map[string]string
	}{
		{
			name:   "WithUsersSort",
			option: client.WithUsersSort("login", "desc"),
			expected: map[string]string{
				"page.sort": "login,desc",
			},
		},
		{
			name:   "WithUsersType",
			option: client.WithUsersType("INTERNAL"),
			expected: map[string]string{
				"filter.eq.type": "INTERNAL",
			},
		},
		{
			name:   "WithUsersActive",
			option: client.WithUsersActive(true),
			expected: map[string]string{
				"filter.eq.active": "true",
			},
		},
		{
			name:   "WithUsersEmail",
			option: client.WithUsersEmail("user@example.com"),
			expected: map[string]string{
				"filter.eq.email": "user@example.com",
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

			_, err := c.Users().List(t.Context(), tt.option)
			require.NoError(t, err)
		})
	}
}

func TestUsersScope_List_WithSortCleared(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[],"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}}`))
	}))

	_, err := c.Users().List(t.Context(), client.WithUsersSort())
	require.NoError(t, err)
}

func TestUsersScope_List_EmptyPage(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/users/all")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 20, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	page, err := c.Users().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, page)
	require.NotNil(t, page.Content)
	assert.Empty(t, *page.Content)
}

func TestUsersScope_List_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4031, "message": "forbidden"}`))

	_, err := c.Users().List(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestUsersScope_Search_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/search")

		assert.Equal(t, "john", r.URL.Query().Get("term"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": 5, "userId": "john.doe", "email": "john@example.com", "userRole": "USER"}
			],
			"page": {"number": 0, "size": 20, "totalElements": 1, "totalPages": 1}
		}`))
	}))

	page, err := c.Users().Search(t.Context(), "john")
	require.NoError(t, err)
	require.NotNil(t, page)
	require.NotNil(t, page.Content)
	assert.Len(t, *page.Content, 1)
	assert.Equal(t, "john.doe", (*page.Content)[0].UserId)
}

func TestUsersScope_Search_WithPagination(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/users/search")

		assert.Equal(t, "admin", r.URL.Query().Get("term"))
		assert.Equal(t, "5", r.URL.Query().Get("page.size"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	page, err := c.Users().Search(t.Context(), "admin",
		client.WithSearchUsersPageSize(5),
	)
	require.NoError(t, err)
	require.NotNil(t, page)
	require.NotNil(t, page.Content)
	assert.Empty(t, *page.Content)
}

func TestUsersScope_Search_WithAdditionalOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		option   client.SearchUsersOption
		expected map[string]string
	}{
		{
			name:   "WithSearchUsersPageNumber",
			option: client.WithSearchUsersPageNumber(2),
			expected: map[string]string{
				"page.page": "2",
			},
		},
		{
			name:   "WithSearchUsersSort",
			option: client.WithSearchUsersSort("login", "asc"),
			expected: map[string]string{
				"page.sort": "login,asc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				assert.Equal(t, "test", query.Get("term"))

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

			_, err := c.Users().Search(t.Context(), "test", tt.option)
			require.NoError(t, err)
		})
	}
}

func TestUsersScope_Search_WithSortCleared(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("page.sort"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[],"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}}`))
	}))

	_, err := c.Users().Search(t.Context(), "test", client.WithSearchUsersSort())
	require.NoError(t, err)
}

func TestUsersScope_Search_EmptyPage(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/users/search")
		assert.Equal(t, "nonexistent", r.URL.Query().Get("term"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 0, "size": 20, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	page, err := c.Users().Search(t.Context(), "nonexistent")
	require.NoError(t, err)
	require.NotNil(t, page)
	require.NotNil(t, page.Content)
	assert.Empty(t, *page.Content)
}

func TestUsersScope_Search_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4031, "message": "forbidden"}`))

	_, err := c.Users().Search(t.Context(), "test")
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestUsersScope_Export_Success(t *testing.T) {
	t.Parallel()

	csvData := []byte("id,userId,email,role\n1,admin,admin@example.com,ADMINISTRATOR\n2,user1,user1@example.com,USER\n")

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/export")

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(csvData)
	}))

	data, err := c.Users().Export(t.Context())
	require.NoError(t, err)
	assert.Equal(t, csvData, data)
}

func TestUsersScope_Export_WithFilter(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/users/export")
		assert.Equal(t, "ADMINISTRATOR", r.URL.Query().Get("filter.eq.role"))

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("id,userId\n1,admin\n"))
	}))

	data, err := c.Users().Export(t.Context(), client.WithUsersRole("ADMINISTRATOR"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "admin")
}

func TestUsersScope_Export_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4031, "message": "forbidden"}`))

	_, err := c.Users().Export(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestUsersScope_Create_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/users")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req types.CreateUserRQFull
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "newuser", req.Login)
		assert.Equal(t, "new@example.com", req.Email)
		assert.Equal(t, "New User", req.FullName)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		_, _ = w.Write([]byte(`{
			"id": 10,
			"login": "newuser",
			"email": "new@example.com",
			"fullName": "New User"
		}`))
	}))

	result, err := c.Users().Create(t.Context(), types.CreateUserRQFull{
		Login:    "newuser",
		Email:    "new@example.com",
		FullName: "New User",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, int64(10), *result.Id)
	require.NotNil(t, result.Login)
	assert.Equal(t, "newuser", *result.Login)
}

func TestUsersScope_Create_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4031, "message": "forbidden"}`))

	_, err := c.Users().Create(t.Context(), types.CreateUserRQFull{
		Login:    "newuser",
		Email:    "new@example.com",
		FullName: "New User",
	})
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestUsersScope_Edit_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPut)
		assertPath(t, r, "/api/users/testuser")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req types.EditUserRQ
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "Updated Name", req.FullName)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"message": "User with login = 'testuser' successfully updated."}`))
	}))

	newEmail := "updated@example.com"

	result, err := c.Users().Edit(t.Context(), "testuser", types.EditUserRQ{
		FullName: "Updated Name",
		Email:    &newEmail,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Contains(t, *result.Message, "testuser")
}

func TestUsersScope_Edit_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User not found."}`))

	_, err := c.Users().Edit(t.Context(), "nonexistent", types.EditUserRQ{
		FullName: "Name",
	})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestUsersScope_Edit_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4031, "message": "forbidden"}`))

	_, err := c.Users().Edit(t.Context(), "admin", types.EditUserRQ{
		FullName: "Name",
	})
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestUsersScope_Delete_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/api/users/42")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"message": "User with id = '42' successfully deleted."}`))
	}))

	result, err := c.Users().Delete(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Contains(t, *result.Message, "42")
}

func TestUsersScope_Delete_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User not found."}`))

	_, err := c.Users().Delete(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestUsersScope_Delete_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4031, "message": "forbidden"}`))

	_, err := c.Users().Delete(t.Context(), 1)
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestUsersScope_DeleteBulk_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/api/users")

		ids := r.URL.Query()["ids"]
		assert.Contains(t, ids, "10")
		assert.Contains(t, ids, "20")
		assert.Contains(t, ids, "30")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"successfullyDeleted": [10, 20, 30],
			"notFound": [],
			"errors": []
		}`))
	}))

	result, err := c.Users().DeleteBulk(t.Context(), []int64{10, 20, 30})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.SuccessfullyDeleted)
	assert.Len(t, *result.SuccessfullyDeleted, 3)
}

func TestUsersScope_DeleteBulk_PartialFailure(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/api/users")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"successfullyDeleted": [10],
			"notFound": [999],
			"errors": []
		}`))
	}))

	result, err := c.Users().DeleteBulk(t.Context(), []int64{10, 999})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.SuccessfullyDeleted)
	assert.Len(t, *result.SuccessfullyDeleted, 1)
	require.NotNil(t, result.NotFound)
	assert.Len(t, *result.NotFound, 1)
}

func TestUsersScope_DeleteBulk_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4031, "message": "forbidden"}`))

	_, err := c.Users().DeleteBulk(t.Context(), []int64{1, 2})
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestUsersScope_CreateInvitation_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/users/bid")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req types.CreateUserRQ
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "invite@example.com", req.Email)
		assert.Equal(t, "my_project", req.DefaultProject)
		assert.Equal(t, "MEMBER", req.Role)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		_, _ = w.Write([]byte(`{
			"bid": "abc-123-def",
			"backLink": "http://example.com/register?uuid=abc-123-def",
			"message": "Invitation sent."
		}`))
	}))

	result, err := c.Users().CreateInvitation(t.Context(), types.CreateUserRQ{
		Email:          "invite@example.com",
		DefaultProject: "my_project",
		Role:           "MEMBER",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Bid)
	assert.Equal(t, "abc-123-def", *result.Bid)
}

func TestUsersScope_CreateInvitation_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4031, "message": "forbidden"}`))

	_, err := c.Users().CreateInvitation(t.Context(), types.CreateUserRQ{
		Email:          "invite@example.com",
		DefaultProject: "my_project",
		Role:           "MEMBER",
	})
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestUsersScope_GetInvitation_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/registration")

		assert.Equal(t, "abc-123-def", r.URL.Query().Get("uuid"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"uuid": "abc-123-def",
			"email": "invite@example.com",
			"isActive": true
		}`))
	}))

	result, err := c.Users().GetInvitation(t.Context(), "abc-123-def")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Uuid)
	assert.Equal(t, "abc-123-def", *result.Uuid)
	require.NotNil(t, result.Email)
	assert.Equal(t, "invite@example.com", *result.Email)
}

func TestUsersScope_GetInvitation_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Invitation not found."}`))

	_, err := c.Users().GetInvitation(t.Context(), "nonexistent-uuid")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestUsersScope_Register_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/users/registration")

		assert.Equal(t, "abc-123-def", r.URL.Query().Get("uuid"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req types.CreateUserRQConfirm
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "newuser", req.Login)
		assert.Equal(t, "New User", req.FullName)
		assert.Equal(t, "new@example.com", req.Email)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		_, _ = w.Write([]byte(`{
			"id": 15,
			"login": "newuser",
			"email": "new@example.com",
			"fullName": "New User"
		}`))
	}))

	result, err := c.Users().Register(t.Context(), "abc-123-def", types.CreateUserRQConfirm{
		Login:    "newuser",
		FullName: "New User",
		Email:    "new@example.com",
		Password: "secure123",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, int64(15), *result.Id)
	require.NotNil(t, result.Login)
	assert.Equal(t, "newuser", *result.Login)
}

func TestUsersScope_Register_BadRequest(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 40016, "message": "Bad request."}`))

	_, err := c.Users().Register(t.Context(), "invalid-uuid", types.CreateUserRQConfirm{
		Login:    "newuser",
		FullName: "New User",
		Email:    "new@example.com",
		Password: "short",
	})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

// --- Password management tests ---

func TestUsersScope_ChangePassword_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/users/password/change")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "oldPass123", payload["oldPassword"])
		assert.Equal(t, "newPass456", payload["newPassword"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Password has been changed successfully."}`))
	}))

	newPass := "newPass456"
	result, err := c.Users().ChangePassword(t.Context(), types.ChangePasswordRQ{
		OldPassword: "oldPass123",
		NewPassword: &newPass,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, *result.Message, "successfully")
}

func TestUsersScope_ChangePassword_BadRequest(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 40010, "message": "Old password is incorrect."}`))

	newPass := "newPass456"
	_, err := c.Users().ChangePassword(t.Context(), types.ChangePasswordRQ{
		OldPassword: "wrongOldPass",
		NewPassword: &newPass,
	})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestUsersScope_RestorePassword_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/users/password/restore")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "user@example.com", payload["email"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Password restore instruction has been sent."}`))
	}))

	result, err := c.Users().RestorePassword(t.Context(), types.RestorePasswordRQ{
		Email: "user@example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, *result.Message, "restore")
}

func TestUsersScope_RestorePassword_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User with email not found."}`))

	_, err := c.Users().RestorePassword(t.Context(), types.RestorePasswordRQ{
		Email: "nonexistent@example.com",
	})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestUsersScope_IsRestorePasswordBidExist_Exists(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/password/reset/abc-123-def")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"is": true}`))
	}))

	exists, err := c.Users().IsRestorePasswordBidExist(t.Context(), "abc-123-def")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUsersScope_IsRestorePasswordBidExist_DoesNotExist(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"is": false}`))
	}))

	exists, err := c.Users().IsRestorePasswordBidExist(t.Context(), "invalid-uuid")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestUsersScope_IsRestorePasswordBidExist_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Restore password bid not found."}`))

	_, err := c.Users().IsRestorePasswordBidExist(t.Context(), "invalid-uuid")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestUsersScope_ResetPassword_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/users/password/reset")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "newSecurePass", payload["password"])
		assert.Equal(t, "abc-123-def", payload["uuid"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Password has been reset successfully."}`))
	}))

	result, err := c.Users().ResetPassword(t.Context(), types.ResetPasswordRQ{
		Password: "newSecurePass",
		Uuid:     "abc-123-def",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, *result.Message, "reset")
}

func TestUsersScope_ResetPassword_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 40010, "message": "Password reset bid has expired."}`))

	_, err := c.Users().ResetPassword(t.Context(), types.ResetPasswordRQ{
		Password: "newPass",
		Uuid:     "expired-uuid",
	})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

// --- API key management tests ---

func TestUsersScope_ListAPIKeys_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/42/api-keys")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"items": [
				{"id": 1, "name": "ci-key", "created_at": "2025-01-15T10:00:00Z", "user_id": 42},
				{"id": 2, "name": "dev-key", "created_at": "2025-02-20T14:30:00Z", "last_used_at": "2025-03-01T09:00:00Z", "user_id": 42}
			]
		}`))
	}))

	keys, err := c.Users().ListAPIKeys(t.Context(), 42)
	require.NoError(t, err)
	require.Len(t, keys, 2)

	assert.Equal(t, int64(1), keys[0].Id)
	assert.Equal(t, "ci-key", keys[0].Name)
	assert.Equal(t, int64(42), keys[0].UserId)

	assert.Equal(t, int64(2), keys[1].Id)
	assert.Equal(t, "dev-key", keys[1].Name)
	assert.NotNil(t, keys[1].LastUsedAt)
}

func TestUsersScope_ListAPIKeys_Empty(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/42/api-keys")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": []}`))
	}))

	keys, err := c.Users().ListAPIKeys(t.Context(), 42)
	require.NoError(t, err)
	assert.Empty(t, keys)
	assert.NotNil(t, keys)
}

func TestUsersScope_ListAPIKeys_NullItems(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))

	keys, err := c.Users().ListAPIKeys(t.Context(), 42)
	require.NoError(t, err)
	assert.Empty(t, keys)
	assert.NotNil(t, keys)
}

func TestUsersScope_ListAPIKeys_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4031, "message": "forbidden"}`))

	_, err := c.Users().ListAPIKeys(t.Context(), 42)
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestUsersScope_CreateAPIKey_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, fmt.Sprintf("/api/users/%d/api-keys", 42))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "my-new-key", payload["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": 99,
			"name": "my-new-key",
			"created_at": "2025-06-01T12:00:00Z",
			"user_id": 42,
			"api_key": "rp_ultra_secret_key_value"
		}`))
	}))

	result, err := c.Users().CreateAPIKey(t.Context(), 42, types.ApiKeyRQ{Name: "my-new-key"})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, int64(99), result.Id)
	assert.Equal(t, "my-new-key", result.Name)
	assert.Equal(t, int64(42), result.UserId)
	require.NotNil(t, result.ApiKey)
	assert.Equal(t, "rp_ultra_secret_key_value", *result.ApiKey)
}

func TestUsersScope_CreateAPIKey_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 40010, "message": "API key name already exists."}`))

	_, err := c.Users().CreateAPIKey(t.Context(), 42, types.ApiKeyRQ{Name: "duplicate-name"})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestUsersScope_DeleteAPIKey_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/api/users/42/api-keys/99")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "API key has been deleted."}`))
	}))

	result, err := c.Users().DeleteAPIKey(t.Context(), 42, 99)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, *result.Message, "deleted")
}

func TestUsersScope_DeleteAPIKey_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "API key not found."}`))

	_, err := c.Users().DeleteAPIKey(t.Context(), 42, 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
