package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
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

	launch, err := c.Project("my_project").Launches().GetByID(t.Context(), 42)
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
		id        int64
		assertErr func(error) bool
	}{
		{
			name:      "NotFound",
			status:    http.StatusNotFound,
			body:      `{"errorCode": 40410, "message": "Launch '999' not found."}`,
			id:        999,
			assertErr: client.IsNotFound,
		},
		{
			name:      "Unauthorized",
			status:    http.StatusUnauthorized,
			body:      `{"errorCode": 4001, "message": "Full authentication is required."}`,
			id:        42,
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

func TestLaunchScope_Start_Success(t *testing.T) {
	t.Parallel()

	startTime := time.UnixMilli(1700000000000)
	mode := types.StartLaunchRQModeDEFAULT

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch")
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "Integration Launch", payload["name"])
		assert.Equal(t, float64(startTime.UnixMilli()), payload["startTime"])
		assert.Equal(t, string(mode), payload["mode"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"launch-uuid-1","number":7}`))
	}))

	result, err := c.Project("my_project").Launches().Start(t.Context(), types.StartLaunchRQ{
		Name:      "Integration Launch",
		StartTime: types.EpochMillis(startTime),
		Mode:      &mode,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	require.NotNil(t, result.Number)

	assert.Equal(t, "launch-uuid-1", *result.Id)
	assert.Equal(t, int64(7), *result.Number)
}

func TestLaunchScope_Start_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "BadRequest",
			statusCode: http.StatusBadRequest,
			body:       `{"errorCode": 4001, "message": "Invalid launch payload."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Launches().Start(
				t.Context(),
				types.StartLaunchRQ{
					Name:      "Integration Launch",
					StartTime: types.EpochMillis(time.Now()),
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLaunchScope_Finish_Success(t *testing.T) {
	t.Parallel()

	endTime := time.UnixMilli(1700000007000)
	status := types.FinishExecutionRQStatusPASSED

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/launch-uuid-1/finish")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, float64(endTime.UnixMilli()), payload["endTime"])
		assert.Equal(t, string(status), payload["status"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"launch-uuid-1","link":"http://rp/launch/1","number":7}`))
	}))

	result, err := c.Project("my_project").Launches().Finish(
		t.Context(),
		"launch-uuid-1",
		types.FinishExecutionRQ{
			EndTime: types.EpochMillis(endTime),
			Status:  &status,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	require.NotNil(t, result.Link)
	require.NotNil(t, result.Number)

	assert.Equal(t, "launch-uuid-1", *result.Id)
	assert.Equal(t, "http://rp/launch/1", *result.Link)
	assert.Equal(t, int64(7), *result.Number)
}

func TestLaunchScope_Finish_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40410, "message": "Launch not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			status := types.FinishExecutionRQStatusPASSED
			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Launches().Finish(
				t.Context(),
				"missing-launch",
				types.FinishExecutionRQ{
					EndTime: types.EpochMillis(time.Now()),
					Status:  &status,
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
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

func TestLaunchScope_Update_Success(t *testing.T) {
	t.Parallel()

	description := "Updated description"

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/42/update")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "Updated description", payload["description"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Launch successfully updated."}`))
	}))

	result, err := c.Project("my_project").Launches().Update(
		t.Context(),
		42,
		types.UpdateLaunchRQ{
			Description: &description,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Launch successfully updated.", *result.Message)
}

func TestLaunchScope_Update_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40410, "message": "Launch '999' not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Launches().Update(
				t.Context(),
				999,
				types.UpdateLaunchRQ{},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLaunchScope_Delete_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/42")
		assertMethod(t, r, http.MethodDelete)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Launch successfully deleted."}`))
	}))

	result, err := c.Project("my_project").Launches().Delete(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Launch successfully deleted.", *result.Message)
}

func TestLaunchScope_Delete_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40410, "message": "Launch '999' not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Launches().Delete(t.Context(), 999)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLaunchScope_DeleteBulk_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch")
		assertMethod(t, r, http.MethodDelete)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		ids := payload["ids"].([]any)
		assert.Len(t, ids, 2)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"successfullyDeleted":[1,2],"notFound":[],"errors":[]}`))
	}))

	result, err := c.Project("my_project").Launches().DeleteBulk(t.Context(), []int64{1, 2})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.SuccessfullyDeleted)
	assert.Equal(t, []int64{1, 2}, *result.SuccessfullyDeleted)
}

func TestLaunchScope_DeleteBulk_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Launches().DeleteBulk(t.Context(), []int64{1, 2})
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestLaunchScope_ForceFinish_Success(t *testing.T) {
	t.Parallel()

	endTime := time.UnixMilli(1700000007000)
	status := types.FinishExecutionRQStatusFAILED

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/42/stop")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, float64(endTime.UnixMilli()), payload["endTime"])
		assert.Equal(t, string(status), payload["status"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Launch force finished."}`))
	}))

	result, err := c.Project("my_project").Launches().ForceFinish(
		t.Context(),
		42,
		types.FinishExecutionRQ{
			EndTime: types.EpochMillis(endTime),
			Status:  &status,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Launch force finished.", *result.Message)
}

func TestLaunchScope_ForceFinish_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40410, "message": "Launch '999' not found."}`,
			check:      client.IsNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Launches().ForceFinish(
				t.Context(),
				999,
				types.FinishExecutionRQ{
					EndTime: types.EpochMillis(time.Now()),
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLaunchScope_Merge_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/merge")
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "Merged Launch", payload["name"])
		assert.Equal(t, "BASIC", payload["mergeType"], "should default to BASIC merge type")

		launches := payload["launches"].([]any)
		assert.Len(t, launches, 2)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 100,
			"uuid": "merged-uuid",
			"name": "Merged Launch",
			"status": "PASSED",
			"startTime": 1700000000000,
			"number": 10
		}`))
	}))

	result, err := c.Project("my_project").Launches().Merge(
		t.Context(),
		types.MergeLaunchesRQ{
			Name:     "Merged Launch",
			Launches: []int64{1, 2},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Merged Launch", result.Name)
	assert.Equal(t, int64(100), result.Id)
}

func TestLaunchScope_Merge_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "BadRequest",
			statusCode: http.StatusBadRequest,
			body:       `{"errorCode": 4001, "message": "Invalid merge configuration."}`,
			check:      client.IsBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Launches().Merge(
				t.Context(),
				types.MergeLaunchesRQ{
					Name:     "Bad Merge",
					Launches: []int64{1, 2},
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLaunchScope_Analyze_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/analyze")
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, float64(42), payload["launchId"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Analysis started."}`))
	}))

	launchID := int64(42)
	result, err := c.Project("my_project").Launches().Analyze(
		t.Context(),
		types.AnalyzeLaunchRQ{
			LaunchId: &launchID,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Analysis started.", *result.Message)
}

func TestLaunchScope_Analyze_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40410, "message": "Launch '999' not found."}`,
			check:      client.IsNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			launchID := int64(999)
			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Launches().Analyze(
				t.Context(),
				types.AnalyzeLaunchRQ{
					LaunchId: &launchID,
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLaunchScope_BulkUpdate_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/info")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		ids := payload["ids"].([]any)
		assert.Len(t, ids, 2)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Launches successfully updated."}`))
	}))

	comment := "Updated via bulk"
	description := types.Description{Comment: &comment}
	result, err := c.Project("my_project").Launches().BulkUpdate(
		t.Context(),
		types.BulkInfoUpdateRQ{
			Ids:         &[]int64{1, 2},
			Description: &description,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Launches successfully updated.", *result.Message)
}

func TestLaunchScope_BulkUpdate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "BadRequest",
			statusCode: http.StatusBadRequest,
			body:       `{"errorCode": 4001, "message": "Invalid bulk update payload."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Launches().BulkUpdate(
				t.Context(),
				types.BulkInfoUpdateRQ{
					Ids: &[]int64{1, 2},
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestLaunchScope_BulkForceFinish_Success(t *testing.T) {
	t.Parallel()

	endTime := time.UnixMilli(1700000007000)
	status := types.FinishExecutionRQStatusFAILED

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/launch/stop")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		entities := payload["entities"].(map[string]any)
		assert.Len(t, entities, 1)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"message":"Launch force finished."}]`))
	}))

	result, err := c.Project("my_project").Launches().BulkForceFinish(
		t.Context(),
		types.BulkRQLongFinishExecutionRQ{
			Entities: map[string]types.FinishExecutionRQ{
				"42": {
					EndTime: types.EpochMillis(endTime),
					Status:  &status,
				},
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].Message)
	assert.Equal(t, "Launch force finished.", *result[0].Message)
}

func TestLaunchScope_BulkForceFinish_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "BadRequest",
			statusCode: http.StatusBadRequest,
			body:       `{"errorCode": 4001, "message": "Invalid bulk force finish payload."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := newTestClient(t, jsonResponse(testCase.statusCode, testCase.body))
			_, err := testClient.Project("my_project").Launches().BulkForceFinish(
				t.Context(),
				types.BulkRQLongFinishExecutionRQ{
					Entities: map[string]types.FinishExecutionRQ{},
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}
