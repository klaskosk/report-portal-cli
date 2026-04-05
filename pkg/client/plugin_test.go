package client_test

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
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
	assert.True(t, *plugins[0].Enabled)
	assert.Equal(t, "jira", *plugins[1].Name)
	assert.False(t, *plugins[1].Enabled)
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

func TestPluginScope_Upload_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/plugin")
		assertMethod(t, r, http.MethodPost)

		ct := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(ct)
		require.NoError(t, err)
		assert.Equal(t, "multipart/form-data", mediaType)

		reader := multipart.NewReader(r.Body, params["boundary"])

		part, err := reader.NextPart()
		require.NoError(t, err)
		assert.Equal(t, "file", part.FormName())
		assert.Equal(t, "my-plugin.jar", part.FileName())

		data, err := io.ReadAll(part)
		require.NoError(t, err)
		assert.Equal(t, "plugin-binary-data", string(data))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		_, _ = w.Write([]byte(`{"id": 42}`))
	}))

	result, err := c.Plugins().Upload(
		t.Context(), "my-plugin.jar", strings.NewReader("plugin-binary-data"),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(42), *result.Id)
}

func TestPluginScope_Upload_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Plugins().Upload(
		t.Context(), "my-plugin.jar", strings.NewReader("data"),
	)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestPluginScope_ExecuteCommand_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/plugin/my_project/jira/common/createTicket")
		assertMethod(t, r, http.MethodPut)
		assertHeader(t, r, "Content-Type", "application/json")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "BUG-123", body["ticketId"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"status": "ok", "ticketUrl": "https://jira.example.com/BUG-123"}`))
	}))

	result, err := c.Plugins().ExecuteCommand(
		t.Context(), "my_project", "jira", "createTicket",
		map[string]any{"ticketId": "BUG-123"},
	)
	require.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
	assert.Equal(t, "https://jira.example.com/BUG-123", result["ticketUrl"])
}

func TestPluginScope_ExecuteCommand_NullResponse(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`null`))
	}))

	result, err := c.Plugins().ExecuteCommand(
		t.Context(), "my_project", "jira", "createTicket",
		map[string]any{"ticketId": "BUG-123"},
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestPluginScope_ExecuteCommand_NilBody(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.NotNil(t, body, "nil body should be normalized to empty object")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"ok": true}`))
	}))

	result, err := c.Plugins().ExecuteCommand(
		t.Context(), "my_project", "jira", "cmd", nil,
	)
	require.NoError(t, err)
	assert.Equal(t, true, result["ok"])
}

func TestPluginScope_ExecuteCommand_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Plugin 'missing' not found."}`))

	_, err := c.Plugins().ExecuteCommand(
		t.Context(), "my_project", "missing", "cmd",
		map[string]any{},
	)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestPluginScope_Import_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/plugin/my_project/jira/import")
		assertMethod(t, r, http.MethodPost)

		ct := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(ct)
		require.NoError(t, err)
		assert.Equal(t, "multipart/form-data", mediaType)

		reader := multipart.NewReader(r.Body, params["boundary"])

		part, err := reader.NextPart()
		require.NoError(t, err)
		assert.Equal(t, "file", part.FormName())
		assert.Equal(t, "results.zip", part.FileName())

		data, err := io.ReadAll(part)
		require.NoError(t, err)
		assert.Equal(t, "zip-content", string(data))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"message": "Import completed."}`))
	}))

	result, err := c.Plugins().Import(
		t.Context(), "my_project", "jira",
		"results.zip", strings.NewReader("zip-content"), nil,
	)
	require.NoError(t, err)
	assert.Equal(t, "Import completed.", result["message"])
}

func TestPluginScope_Import_NullResponse(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`null`))
	}))

	result, err := c.Plugins().Import(
		t.Context(), "my_project", "jira",
		"results.zip", strings.NewReader("zip-content"), nil,
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestPluginScope_Import_WithLaunchImportRQ(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/plugin/my_project/jira/import")
		assertMethod(t, r, http.MethodPost)

		ct := r.Header.Get("Content-Type")
		_, params, err := mime.ParseMediaType(ct)
		require.NoError(t, err)

		reader := multipart.NewReader(r.Body, params["boundary"])

		filePart, err := reader.NextPart()
		require.NoError(t, err)
		assert.Equal(t, "file", filePart.FormName())
		assert.Equal(t, "results.zip", filePart.FileName())

		_, err = io.ReadAll(filePart)
		require.NoError(t, err)

		jsonPart, err := reader.NextPart()
		require.NoError(t, err)
		assert.Equal(t, "launchImportRq", jsonPart.FormName())
		assert.Contains(t, jsonPart.Header.Get("Content-Type"), "application/json")

		var importRQ types.LaunchImportRQ
		err = json.NewDecoder(jsonPart).Decode(&importRQ)
		require.NoError(t, err)
		assert.Equal(t, "My Launch", *importRQ.Name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"message": "Import completed."}`))
	}))

	launchName := "My Launch"
	result, err := c.Plugins().Import(
		t.Context(), "my_project", "jira",
		"results.zip", strings.NewReader("zip-content"),
		&types.LaunchImportRQ{Name: &launchName},
	)
	require.NoError(t, err)
	assert.Equal(t, "Import completed.", result["message"])
}

func TestPluginScope_Import_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Plugins().Import(
		t.Context(), "my_project", "jira",
		"results.zip", strings.NewReader("data"), nil,
	)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestPluginScope_ExecutePublicCommand_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/v1/plugin/public/saml/metadata")
		assertMethod(t, r, http.MethodPut)

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "value1", body["key1"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"result": "success"}`))
	}))

	result, err := c.Plugins().ExecutePublicCommand(
		t.Context(), "saml", "metadata",
		map[string]any{"key1": "value1"},
	)
	require.NoError(t, err)
	assert.Equal(t, "success", result["result"])
}

func TestPluginScope_ExecutePublicCommand_NullResponse(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`null`))
	}))

	result, err := c.Plugins().ExecutePublicCommand(
		t.Context(), "saml", "metadata",
		map[string]any{"key1": "value1"},
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestPluginScope_ExecutePublicCommand_NilBody(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.NotNil(t, body, "nil body should be normalized to empty object")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"ok": true}`))
	}))

	result, err := c.Plugins().ExecutePublicCommand(
		t.Context(), "saml", "metadata", nil,
	)
	require.NoError(t, err)
	assert.Equal(t, true, result["ok"])
}

func TestPluginScope_ExecutePublicCommand_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Plugin 'missing' not found."}`))

	_, err := c.Plugins().ExecutePublicCommand(
		t.Context(), "missing", "cmd",
		map[string]any{},
	)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
