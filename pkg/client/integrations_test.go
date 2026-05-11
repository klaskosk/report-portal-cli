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

func TestIntegrationsScope_ListGlobal_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/global/all")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[
			{
				"id": 1,
				"name": "email",
				"enabled": true,
				"integrationType": {"name": "email"}
			},
			{
				"id": 2,
				"name": "ldap",
				"enabled": false,
				"integrationType": {"name": "ldap"}
			}
		]`))
		},
	))

	integrations, err := testClient.Integrations().ListGlobal(t.Context())
	require.NoError(t, err)
	require.Len(t, integrations, 2)

	assert.Equal(t, int64(1), *integrations[0].Id)
	assert.Equal(t, "email", *integrations[0].Name)
	assert.True(t, *integrations[0].Enabled)
}

func TestIntegrationsScope_ListGlobal_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Integrations().ListGlobal(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestIntegrationsScope_ListGlobalByPlugin_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/global/all/email")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[
			{
				"id": 1,
				"name": "email",
				"enabled": true,
				"integrationType": {"name": "email"}
			}
		]`))
		},
	))

	integrations, err := testClient.Integrations().ListGlobalByPlugin(t.Context(), "email")
	require.NoError(t, err)
	require.Len(t, integrations, 1)

	assert.Equal(t, "email", *integrations[0].Name)
}

func TestIntegrationsScope_ListGlobalByPlugin_Empty(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[]`))
		},
	))

	integrations, err := testClient.Integrations().ListGlobalByPlugin(t.Context(), "unknown")
	require.NoError(t, err)
	assert.Empty(t, integrations)
}

func TestIntegrationsScope_GetGlobal_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/1")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 1,
			"name": "email",
			"enabled": true,
			"integrationType": {"name": "email"}
		}`))
		},
	))

	integration, err := testClient.Integrations().GetGlobal(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, integration)

	assert.Equal(t, int64(1), *integration.Id)
	assert.Equal(t, "email", *integration.Name)
}

func TestIntegrationsScope_GetGlobal_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.Integrations().GetGlobal(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestIntegrationsScope_TestGlobalConnection_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/1/connection/test")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`true`))
		},
	))

	ok, err := testClient.Integrations().TestGlobalConnection(t.Context(), 1)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestIntegrationsScope_TestGlobalConnection_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.Integrations().TestGlobalConnection(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectIntegrationsScope_List_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/project/my_project/all")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[
			{
				"id": 10,
				"name": "jira",
				"enabled": true,
				"integrationType": {"name": "jira"},
				"projectId": 1
			}
		]`))
		},
	))

	integrations, err := testClient.Project("my_project").Integrations().List(t.Context())
	require.NoError(t, err)
	require.Len(t, integrations, 1)

	assert.Equal(t, int64(10), *integrations[0].Id)
	assert.Equal(t, "jira", *integrations[0].Name)
}

func TestProjectIntegrationsScope_List_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Integrations().List(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectIntegrationsScope_ListByPlugin_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/project/my_project/all/jira")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[
			{
				"id": 10,
				"name": "jira",
				"enabled": true,
				"integrationType": {"name": "jira"}
			}
		]`))
		},
	))

	integrations, err := testClient.Project("my_project").Integrations().
		ListByPlugin(t.Context(), "jira")
	require.NoError(t, err)
	require.Len(t, integrations, 1)

	assert.Equal(t, "jira", *integrations[0].Name)
}

func TestProjectIntegrationsScope_Get_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/my_project/10")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 10,
			"name": "jira",
			"enabled": true,
			"integrationType": {"name": "jira"}
		}`))
		},
	))

	integration, err := testClient.Project("my_project").Integrations().Get(t.Context(), 10)
	require.NoError(t, err)
	require.NotNil(t, integration)

	assert.Equal(t, int64(10), *integration.Id)
}

func TestProjectIntegrationsScope_Get_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.Project("my_project").Integrations().Get(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectIntegrationsScope_TestConnection_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/my_project/10/connection/test")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`true`))
		},
	))

	ok, err := testClient.Project("my_project").Integrations().TestConnection(t.Context(), 10)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestProjectIntegrationsScope_TestConnection_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.Project("my_project").Integrations().TestConnection(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestIntegrationsScope_CreateGlobal_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/email")
			assertMethod(t, httpReq, http.MethodPost)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "my-email", payload["name"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(`{"id":5}`))
		},
	))

	name := "my-email"
	result, err := testClient.Integrations().CreateGlobal(t.Context(), "email", types.IntegrationRQ{
		Name: &name,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, int64(5), *result.Id)
}

func TestIntegrationsScope_CreateGlobal_BadRequest(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 4001, "message": "Incorrect request."}`))

	_, err := c.Integrations().CreateGlobal(t.Context(), "email", types.IntegrationRQ{})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestIntegrationsScope_UpdateGlobal_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/1")
			assertMethod(t, httpReq, http.MethodPut)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "updated-email", payload["name"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Integration successfully updated."}`))
		},
	))

	name := "updated-email"
	result, err := testClient.Integrations().UpdateGlobal(t.Context(), 1, types.IntegrationRQ{
		Name: &name,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Integration successfully updated.", *result.Message)
}

func TestIntegrationsScope_UpdateGlobal_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.Integrations().UpdateGlobal(t.Context(), 999, types.IntegrationRQ{})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestIntegrationsScope_DeleteGlobal_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/1")
			assertMethod(t, httpReq, http.MethodDelete)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Integration successfully deleted."}`))
		},
	))

	result, err := testClient.Integrations().DeleteGlobal(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Integration successfully deleted.", *result.Message)
}

func TestIntegrationsScope_DeleteGlobal_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.Integrations().DeleteGlobal(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectIntegrationsScope_Create_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/my_project/email")
			assertMethod(t, httpReq, http.MethodPost)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "project-email", payload["name"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(`{"id":15}`))
		},
	))

	name := "project-email"
	result, err := testClient.Project("my_project").
		Integrations().
		Create(t.Context(), "email", types.IntegrationRQ{
			Name: &name,
		})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, int64(15), *result.Id)
}

func TestProjectIntegrationsScope_Create_BadRequest(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 4001, "message": "Incorrect request."}`))

	_, err := c.Project("my_project").
		Integrations().
		Create(t.Context(), "email", types.IntegrationRQ{})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestProjectIntegrationsScope_Update_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/my_project/10")
			assertMethod(t, httpReq, http.MethodPut)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "updated-jira", payload["name"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Integration successfully updated."}`))
		},
	))

	name := "updated-jira"
	result, err := testClient.Project("my_project").
		Integrations().
		Update(t.Context(), 10, types.IntegrationRQ{
			Name: &name,
		})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Integration successfully updated.", *result.Message)
}

func TestProjectIntegrationsScope_Update_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.Project("my_project").Integrations().Update(t.Context(), 999, types.IntegrationRQ{})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectIntegrationsScope_Delete_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/my_project/10")
			assertMethod(t, httpReq, http.MethodDelete)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Integration successfully deleted."}`))
		},
	))

	result, err := testClient.Project("my_project").Integrations().Delete(t.Context(), 10)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Integration successfully deleted.", *result.Message)
}

func TestProjectIntegrationsScope_Delete_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.Project("my_project").Integrations().Delete(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectIntegrationsScope_ExecuteCommand_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/my_project/10/testCommand")
			assertMethod(t, httpReq, http.MethodPut)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "value1", payload["key1"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"result":"ok","data":"test"}`))
		},
	))

	result, err := testClient.Project("my_project").Integrations().ExecuteCommand(
		t.Context(),
		10,
		"testCommand",
		map[string]any{"key1": "value1"},
	)
	require.NoError(t, err)
	assert.Equal(t, "ok", result["result"])
	assert.Equal(t, "test", result["data"])
}

func TestProjectIntegrationsScope_ExecuteCommand_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.Project("my_project").Integrations().ExecuteCommand(
		t.Context(),
		999,
		"testCommand",
		map[string]any{},
	)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestIntegrationsScope_DeleteAllByType_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, r *http.Request) {
			assertBearerToken(t, r)
			assertPath(t, r, "/api/v1/integration/all/jira")
			assertMethod(t, r, http.MethodDelete)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Integrations successfully deleted."}`))
		},
	))

	result, err := testClient.Integrations().DeleteAllByType(t.Context(), "jira")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Integrations successfully deleted.", *result.Message)
}

func TestIntegrationsScope_DeleteAllByType_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Integrations().DeleteAllByType(t.Context(), "jira")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectIntegrationsScope_DeleteAllByType_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/integration/my_project/all/jira")
			assertMethod(t, httpReq, http.MethodDelete)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Integrations successfully deleted."}`))
		},
	))

	result, err := testClient.Project("my_project").Integrations().
		DeleteAllByType(t.Context(), "jira")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Integrations successfully deleted.", *result.Message)
}

func TestProjectIntegrationsScope_DeleteAllByType_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Integrations().DeleteAllByType(t.Context(), "jira")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}
