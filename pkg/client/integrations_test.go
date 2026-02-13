package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationsScope_ListGlobal_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/integration/global/all")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
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
	}))

	integrations, err := c.Integrations().ListGlobal(t.Context())
	require.NoError(t, err)
	require.Len(t, integrations, 2)

	assert.Equal(t, int64(1), *integrations[0].Id)
	assert.Equal(t, "email", *integrations[0].Name)
	assert.Equal(t, true, *integrations[0].Enabled)
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

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/integration/global/all/email")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
			{
				"id": 1,
				"name": "email",
				"enabled": true,
				"integrationType": {"name": "email"}
			}
		]`))
	}))

	integrations, err := c.Integrations().ListGlobalByPlugin(t.Context(), "email")
	require.NoError(t, err)
	require.Len(t, integrations, 1)

	assert.Equal(t, "email", *integrations[0].Name)
}

func TestIntegrationsScope_ListGlobalByPlugin_Empty(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[]`))
	}))

	integrations, err := c.Integrations().ListGlobalByPlugin(t.Context(), "unknown")
	require.NoError(t, err)
	assert.Empty(t, integrations)
}

func TestIntegrationsScope_GetGlobal_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/integration/1")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 1,
			"name": "email",
			"enabled": true,
			"integrationType": {"name": "email"}
		}`))
	}))

	integration, err := c.Integrations().GetGlobal(t.Context(), 1)
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

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/integration/1/connection/test")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`true`))
	}))

	ok, err := c.Integrations().TestGlobalConnection(t.Context(), 1)
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

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/integration/project/my_project/all")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
			{
				"id": 10,
				"name": "jira",
				"enabled": true,
				"integrationType": {"name": "jira"},
				"projectId": 1
			}
		]`))
	}))

	integrations, err := c.Project("my_project").Integrations().List(t.Context())
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

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/integration/project/my_project/all/jira")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
			{
				"id": 10,
				"name": "jira",
				"enabled": true,
				"integrationType": {"name": "jira"}
			}
		]`))
	}))

	integrations, err := c.Project("my_project").Integrations().ListByPlugin(t.Context(), "jira")
	require.NoError(t, err)
	require.Len(t, integrations, 1)

	assert.Equal(t, "jira", *integrations[0].Name)
}

func TestProjectIntegrationsScope_Get_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/integration/my_project/10")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 10,
			"name": "jira",
			"enabled": true,
			"integrationType": {"name": "jira"}
		}`))
	}))

	integration, err := c.Project("my_project").Integrations().Get(t.Context(), 10)
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

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/integration/my_project/10/connection/test")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`true`))
	}))

	ok, err := c.Project("my_project").Integrations().TestConnection(t.Context(), 10)
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
