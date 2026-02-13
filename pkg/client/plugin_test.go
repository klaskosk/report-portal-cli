package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginScope_List_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/plugin")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
			{
				"name": "emailServer",
				"enabled": true,
				"type": 1
			},
			{
				"name": "jira",
				"enabled": false,
				"type": 2
			}
		]`))
	}))

	plugins, err := c.Plugins().List(t.Context())
	require.NoError(t, err)
	require.Len(t, plugins, 2)

	assert.Equal(t, "emailServer", *plugins[0].Name)
	assert.Equal(t, true, *plugins[0].Enabled)
	assert.Equal(t, "jira", *plugins[1].Name)
	assert.Equal(t, false, *plugins[1].Enabled)
}

func TestPluginScope_List_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Plugins().List(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestPluginScope_ListPublic_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/v1/plugin/public")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
			{
				"name": "emailServer",
				"enabled": true
			}
		]`))
	}))

	plugins, err := c.Plugins().ListPublic(t.Context())
	require.NoError(t, err)
	require.Len(t, plugins, 1)

	assert.Equal(t, "emailServer", *plugins[0].Name)
}

func TestPluginScope_UpdateState_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/plugin/42")
		assertMethod(t, r, http.MethodPut)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"message": "Plugin state was successfully updated."}`))
	}))

	enabledState := false
	result, err := c.Plugins().UpdateState(t.Context(), 42, types.UpdatePluginStateRQ{
		Enabled: &enabledState,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Plugin state was successfully updated.", *result.Message)
}

func TestPluginScope_UpdateState_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Plugin '999' not found."}`))

	enabledState := true
	_, err := c.Plugins().UpdateState(t.Context(), 999, types.UpdatePluginStateRQ{
		Enabled: &enabledState,
	})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestPluginScope_Delete_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/plugin/42")
		assertMethod(t, r, http.MethodDelete)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"message": "Plugin was successfully deleted."}`))
	}))

	result, err := c.Plugins().Delete(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Plugin was successfully deleted.", *result.Message)
}

func TestPluginScope_Delete_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Plugin '999' not found."}`))

	_, err := c.Plugins().Delete(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestPluginScope_GetPublicFile_Success(t *testing.T) {
	t.Parallel()

	expectedContent := []byte("binary-file-content-here")

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/v1/plugin/public/myPlugin/file/icon.png")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(expectedContent)
	}))

	data, err := c.Plugins().GetPublicFile(t.Context(), "myPlugin", "icon.png")
	require.NoError(t, err)
	assert.Equal(t, expectedContent, data)
}

func TestPluginScope_GetPublicFile_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "File 'missing.png' not found."}`))

	_, err := c.Plugins().GetPublicFile(t.Context(), "myPlugin", "missing.png")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
