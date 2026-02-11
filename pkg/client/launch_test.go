package client_test

import (
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

	launch, err := c.Project("my_project").Launches().GetByID(t.Context(), "42")
	require.NoError(t, err)

	assert.Equal(t, int64(42), launch.Id)
	assert.Equal(t, "Test Launch", launch.Name)
	assert.Equal(t, "PASSED", launch.Status)
}

func TestLaunchScope_GetByID_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    int
		body      string
		id        string
		assertErr func(error) bool
	}{
		{
			name:      "NotFound",
			status:    http.StatusNotFound,
			body:      `{"errorCode": 40410, "message": "Launch '999' not found."}`,
			id:        "999",
			assertErr: client.IsNotFound,
		},
		{
			name:      "Unauthorized",
			status:    http.StatusUnauthorized,
			body:      `{"errorCode": 4001, "message": "Full authentication is required."}`,
			id:        "42",
			assertErr: client.IsUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, jsonResponse(tt.status, tt.body))

			_, err := c.Project("my_project").Launches().GetByID(t.Context(), tt.id)
			require.Error(t, err)
			assert.True(t, tt.assertErr(err))
		})
	}
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

	launch, err := c.Project("my_project").Launches().GetByUUID(t.Context(), "abc-def-123")
	require.NoError(t, err)

	assert.Equal(t, "abc-def-123", launch.Uuid)
	assert.Equal(t, "FAILED", launch.Status)
}

func TestLaunchScope_GetByUUID_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Launch not found."}`))

	_, err := c.Project("my_project").Launches().GetByUUID(t.Context(), "nonexistent")
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

	result, err := c.Project("my_project").Launches().List(t.Context())
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

	_, err := c.Project("my_project").Launches().List(t.Context(),
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

	_, err := c.Project("my_project").Launches().List(t.Context())
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

	_, err := c.Project("test").Launches().List(t.Context())
	require.NoError(t, err)
}

func TestLaunchScope_GetLatest_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/latest")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": 1, "uuid": "latest-uuid", "name": "Latest Launch", "status": "PASSED", "startTime": 1700000000000, "number": 5}
			],
			"page": {"number": 1, "size": 10, "totalElements": 1, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Launches().GetLatest(t.Context())
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	assert.Equal(t, "Latest Launch", result.Content[0].Name)
}

func TestLaunchScope_GetLatest_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		assert.Equal(t, "My Launch", query.Get("filter.eq.name"))
		assert.Equal(t, "10", query.Get("page.size"))
		assertPath(t, r, "/api/v1/my_project/launch/latest")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Project("my_project").Launches().GetLatest(t.Context(),
		client.WithName("My Launch"),
		client.WithPageSize(10),
	)
	require.NoError(t, err)
}

func TestLaunchScope_GetDebug_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/mode")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": 2, "uuid": "debug-uuid", "name": "Debug Launch", "status": "IN_PROGRESS", "startTime": 1700000000000, "number": 3, "mode": "DEBUG"}
			],
			"page": {"number": 1, "size": 10, "totalElements": 1, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Launches().GetDebug(t.Context())
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	assert.Equal(t, "Debug Launch", result.Content[0].Name)
}

func TestLaunchScope_GetDebug_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		assert.Equal(t, "Debug Test", query.Get("filter.eq.name"))
		assert.Equal(t, "5", query.Get("page.size"))
		assertPath(t, r, "/api/v1/my_project/launch/mode")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Project("my_project").Launches().GetDebug(t.Context(),
		client.WithName("Debug Test"),
		client.WithPageSize(5),
	)
	require.NoError(t, err)
}

func TestLaunchScope_WithUserFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		option   client.ListLaunchesOption
		expected map[string]string
	}{
		{
			name:   "WithUser",
			option: client.WithUser("testuser"),
			expected: map[string]string{
				"filter.eq.user": "testuser",
			},
		},
		{
			name:   "WithUserID",
			option: client.WithUserID(42),
			expected: map[string]string{
				"filter.eq.userId": "42",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()

				for key, expectedValue := range tt.expected {
					assert.Equal(
						t,
						expectedValue,
						query.Get(key),
						"query parameter mismatch for "+key,
					)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				_, _ = w.Write([]byte(`{
					"content": [],
					"page": {"number": 1, "size": 10, "totalElements": 0, "totalPages": 0}
				}`))
			}))

			_, err := c.Project("my_project").Launches().List(t.Context(), tt.option)
			require.NoError(t, err)
		})
	}
}

func TestLaunchScope_WithMode(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		assert.Equal(t, "DEBUG", query.Get("filter.eq.mode"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Project("my_project").Launches().List(t.Context(),
		client.WithMode("DEBUG"),
	)
	require.NoError(t, err)
}

func TestLaunchScope_WithAttributeFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		option   client.ListLaunchesOption
		expected map[string]string
	}{
		{
			name:   "WithAttribute",
			option: client.WithAttribute("env", "production"),
			expected: map[string]string{
				"filter.eq.compositeAttribute": "env:production",
			},
		},
		{
			name:   "WithAttributeKey",
			option: client.WithAttributeKey("browser"),
			expected: map[string]string{
				"filter.eq.attributeKey": "browser",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()

				for key, expectedValue := range tt.expected {
					assert.Equal(
						t,
						expectedValue,
						query.Get(key),
						"query parameter mismatch for "+key,
					)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				_, _ = w.Write([]byte(`{
					"content": [],
					"page": {"number": 1, "size": 10, "totalElements": 0, "totalPages": 0}
				}`))
			}))

			_, err := c.Project("my_project").Launches().List(t.Context(), tt.option)
			require.NoError(t, err)
		})
	}
}

func TestLaunchScope_WithOtherFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		option   client.ListLaunchesOption
		expected map[string]string
	}{
		{
			name:   "WithDescription",
			option: client.WithDescription("Test description"),
			expected: map[string]string{
				"filter.eq.description": "Test description",
			},
		},
		{
			name:   "WithNumber",
			option: client.WithNumber(5),
			expected: map[string]string{
				"filter.eq.number": "5",
			},
		},
		{
			name:   "WithHasRetries",
			option: client.WithHasRetries(true),
			expected: map[string]string{
				"filter.eq.hasRetries": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()

				for key, expectedValue := range tt.expected {
					assert.Equal(
						t,
						expectedValue,
						query.Get(key),
						"query parameter mismatch for "+key,
					)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				_, _ = w.Write([]byte(`{
					"content": [],
					"page": {"number": 1, "size": 10, "totalElements": 0, "totalPages": 0}
				}`))
			}))

			_, err := c.Project("my_project").Launches().List(t.Context(), tt.option)
			require.NoError(t, err)
		})
	}
}

func TestLaunchScope_GetNames_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/names")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, "smoke", query.Get("filter.cnt.name"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["Smoke Tests", "Smoke Suite", "Smoke Regression"]`))
	}))

	names, err := c.Project("my_project").Launches().GetNames(t.Context(), "smoke")
	require.NoError(t, err)
	require.Len(t, names, 3)

	assert.Equal(t, "Smoke Tests", names[0])
	assert.Equal(t, "Smoke Suite", names[1])
	assert.Equal(t, "Smoke Regression", names[2])
}

func TestLaunchScope_GetNames_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		filter         string
		responseBody   string
		expectedLength int
		assertFilter   func(*testing.T, *http.Request)
	}{
		{
			name:           "Empty",
			filter:         "nonexistent",
			responseBody:   `[]`,
			expectedLength: 0,
			assertFilter:   nil,
		},
		{
			name:           "NoFilter",
			filter:         "",
			responseBody:   `["Launch 1", "Launch 2"]`,
			expectedLength: 2,
			assertFilter: func(t *testing.T, r *http.Request) {
				t.Helper()

				query := r.URL.Query()
				assert.Empty(t, query.Get("filter.cnt.name"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assertBearerToken(t, r, "test-token")
				assertPath(t, r, "/api/v1/my_project/launch/names")

				if tt.assertFilter != nil {
					tt.assertFilter(t, r)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				_, _ = w.Write([]byte(tt.responseBody))
			}))

			names, err := c.Project("my_project").Launches().GetNames(t.Context(), tt.filter)
			require.NoError(t, err)
			require.Len(t, names, tt.expectedLength)
		})
	}
}
