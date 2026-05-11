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

func TestLogTypesScope_List_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/projects/my_project/log-types")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"items": [
				{"id": 1, "name": "INFO", "level": 20000, "is_filterable": true, "is_system": true},
				{"id": 2, "name": "CUSTOM", "level": 50000, "is_filterable": false, "is_system": false}
			]
		}`))
		},
	))

	result, err := testClient.Project("my_project").LogTypes().List(t.Context())
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, int64(1), *result[0].Id)
	assert.Equal(t, "INFO", result[0].Name)
	assert.Equal(t, int32(20000), result[0].Level)
	assert.True(t, result[0].IsFilterable)
	assert.True(t, *result[0].IsSystem)
}

func TestLogTypesScope_List_Empty(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"items": []}`))
		},
	))

	result, err := testClient.Project("my_project").LogTypes().List(t.Context())
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.NotNil(t, result)
}

func TestLogTypesScope_List_NilItems(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))

	result, err := testClient.Project("my_project").LogTypes().List(t.Context())
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.NotNil(t, result)
}

func TestLogTypesScope_List_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").LogTypes().List(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestLogTypesScope_Create_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodPost)
			assertPath(t, httpReq, "/api/projects/my_project/log-types")

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var req types.LogTypeRequest
			if !assert.NoError(t, json.Unmarshal(body, &req)) {
				return
			}

			assert.Equal(t, "CUSTOM", req.Name)
			assert.Equal(t, int32(50000), req.Level)
			assert.True(t, req.IsFilterable)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{
			"id": 10,
			"name": "CUSTOM",
			"level": 50000,
			"is_filterable": true,
			"is_system": false
		}`))
		},
	))

	result, err := testClient.Project("my_project").LogTypes().
		Create(t.Context(), types.LogTypeRequest{
			Name:         "CUSTOM",
			Level:        50000,
			IsFilterable: true,
		})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(10), *result.Id)
	assert.Equal(t, "CUSTOM", result.Name)
	assert.Equal(t, int32(50000), result.Level)
}

func TestLogTypesScope_Create_201Response(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertMethod(t, httpReq, http.MethodPost)
			assertPath(t, httpReq, "/api/projects/my_project/log-types")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(`{
			"id": 11,
			"name": "CUSTOM201",
			"level": 55000,
			"is_filterable": true,
			"is_system": false
		}`))
		},
	))

	result, err := testClient.Project("my_project").LogTypes().Create(
		t.Context(),
		types.LogTypeRequest{
			Name:         "CUSTOM201",
			Level:        55000,
			IsFilterable: true,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(11), *result.Id)
	assert.Equal(t, "CUSTOM201", result.Name)
}

func TestLogTypesScope_Create_201BadJSON(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte("not json"))
		},
	))

	_, err := testClient.Project("my_project").LogTypes().Create(t.Context(), types.LogTypeRequest{
		Name:  "CUSTOM",
		Level: 50000,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create log type")
}

func TestLogTypesScope_Create_UnexpectedStatus(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>Bad Gateway</html>"))
	}))

	_, err := c.Project("my_project").LogTypes().Create(t.Context(), types.LogTypeRequest{
		Name:  "CUSTOM",
		Level: 50000,
	})
	require.Error(t, err)
	assert.True(t, client.HasStatusCode(err, http.StatusBadGateway))
}

func TestLogTypesScope_Create_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4003, "message": "You do not have enough permissions."}`))

	_, err := c.Project("my_project").LogTypes().Create(t.Context(), types.LogTypeRequest{
		Name:  "CUSTOM",
		Level: 50000,
	})
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestLogTypesScope_Update_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, request *http.Request) {
			assertBearerToken(t, request)
			assertMethod(t, request, http.MethodPut)
			assertPath(t, request, "/api/projects/my_project/log-types/10")

			body, err := io.ReadAll(request.Body)
			if !assert.NoError(t, err) {
				return
			}

			var req types.LogTypeRequest
			if !assert.NoError(t, json.Unmarshal(body, &req)) {
				return
			}

			assert.Equal(t, "UPDATED", req.Name)
			assert.Equal(t, int32(60000), req.Level)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			const updateLogTypeMessage = `{"message": "Log type has been updated successfully."}`

			_, _ = responseWriter.Write([]byte(updateLogTypeMessage))
		},
	))

	result, err := testClient.Project("my_project").LogTypes().Update(
		t.Context(),
		10,
		types.LogTypeRequest{
			Name:  "UPDATED",
			Level: 60000,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Log type has been updated successfully.", *result.Message)
}

func TestLogTypesScope_Update_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Log type not found."}`))

	_, err := c.Project("my_project").LogTypes().Update(t.Context(), 999, types.LogTypeRequest{
		Name:  "UPDATED",
		Level: 60000,
	})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestLogTypesScope_Delete_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertMethod(t, httpReq, http.MethodDelete)
			assertPath(t, httpReq, "/api/projects/my_project/log-types/10")

			responseWriter.WriteHeader(http.StatusNoContent)
		},
	))

	err := testClient.Project("my_project").LogTypes().Delete(t.Context(), 10)
	require.NoError(t, err)
}

func TestLogTypesScope_Delete_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Log type not found."}`))

	err := c.Project("my_project").LogTypes().Delete(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
