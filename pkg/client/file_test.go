package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectScope_DownloadFile_Success(t *testing.T) {
	t.Parallel()

	fileContent := []byte("hello world attachment")

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/data/my_project/42")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(fileContent)
	}))

	result, err := c.Project("my_project").DownloadFile(t.Context(), "42")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, fileContent, result.Data)
	assert.Equal(t, "text/plain", result.ContentType)
}

func TestProjectScope_DownloadFile_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40440, "message": "File not found."}`))

	_, err := c.Project("my_project").DownloadFile(t.Context(), "999")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectScope_DownloadFile_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").DownloadFile(t.Context(), "42")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectScope_DownloadFile_InvalidDataID(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	_, err := c.Project("my_project").DownloadFile(t.Context(), "not-a-number")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid data ID")
}

func TestProjectScope_DownloadFile_ImageContent(t *testing.T) {
	t.Parallel()

	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(pngHeader)
	}))

	result, err := c.Project("my_project").DownloadFile(t.Context(), "123")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, pngHeader, result.Data)
	assert.Equal(t, "image/png", result.ContentType)
}
