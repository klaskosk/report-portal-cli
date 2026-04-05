package client_test

import (
	"net/http"
	"strings"
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

	result, err := c.Project("my_project").DownloadFile(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, fileContent, result.Data)
	assert.Equal(t, "text/plain", result.ContentType)
}

func TestProjectScope_DownloadFile_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40440, "message": "File not found."}`))

	_, err := c.Project("my_project").DownloadFile(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectScope_DownloadFile_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").DownloadFile(t.Context(), 42)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectScope_DownloadFile_ImageContent(t *testing.T) {
	t.Parallel()

	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(pngHeader)
	}))

	result, err := c.Project("my_project").DownloadFile(t.Context(), 123)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, pngHeader, result.Data)
	assert.Equal(t, "image/png", result.ContentType)
}

func TestFileStorageScope_GetMyPhoto_Success(t *testing.T) {
	t.Parallel()

	photoBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/data/photo")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(photoBytes)
	}))

	result, err := c.Files().GetMyPhoto(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, photoBytes, result.Data)
	assert.Equal(t, "image/jpeg", result.ContentType)
}

func TestFileStorageScope_GetMyPhoto_WithThumbnail(t *testing.T) {
	t.Parallel()

	photoBytes := []byte{0x89, 0x50, 0x4E, 0x47}

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/v1/data/photo")
		assertMethod(t, r, http.MethodGet)
		assert.Equal(t, "true", r.URL.Query().Get("loadThumbnail"))

		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(photoBytes)
	}))

	result, err := c.Files().GetMyPhoto(t.Context(), client.WithPhotoThumbnail(true))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, photoBytes, result.Data)
	assert.Equal(t, "image/png", result.ContentType)
}

func TestFileStorageScope_GetMyPhoto_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Files().GetMyPhoto(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestFileStorageScope_UploadPhoto_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/data/photo")
		assertMethod(t, r, http.MethodPost)
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		_, header, err := r.FormFile("file")
		require.NoError(t, err)
		assert.Equal(t, "avatar.png", header.Filename)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"message": "Photo has been uploaded successfully"}`))
	}))

	result, err := c.Files().UploadPhoto(t.Context(), "avatar.png", strings.NewReader("fake-image-data"))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Contains(t, *result.Message, "successfully")
}

func TestFileStorageScope_UploadPhoto_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 4001, "message": "Incorrect file format."}`))

	_, err := c.Files().UploadPhoto(t.Context(), "bad.txt", strings.NewReader("not-an-image"))
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestFileStorageScope_DeletePhoto_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/data/photo")
		assertMethod(t, r, http.MethodDelete)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"message": "Photo has been deleted successfully"}`))
	}))

	result, err := c.Files().DeletePhoto(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Contains(t, *result.Message, "successfully")
}

func TestFileStorageScope_DeletePhoto_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40440, "message": "Photo not found."}`))

	_, err := c.Files().DeletePhoto(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestFileStorageScope_CleanAttachments_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/data/clean")
		assertMethod(t, r, http.MethodPost)
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		_, header, err := r.FormFile("file")
		require.NoError(t, err)
		assert.Equal(t, "attachments.csv", header.Filename)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"message": "3 attachments removed"}`))
	}))

	csvData := strings.NewReader("id1\nid2\nid3\n")

	result, err := c.Files().CleanAttachments(t.Context(), "attachments.csv", csvData)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Contains(t, *result.Message, "attachments")
}

func TestFileStorageScope_CleanAttachments_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Files().CleanAttachments(t.Context(), "data.csv", strings.NewReader("data"))
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectScope_GetUserPhoto_Success(t *testing.T) {
	t.Parallel()

	photoBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/data/my_project/userphoto")
		assertMethod(t, r, http.MethodGet)
		assert.Equal(t, "john_doe", r.URL.Query().Get("login"))

		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(photoBytes)
	}))

	result, err := c.Project("my_project").GetUserPhoto(t.Context(), "john_doe")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, photoBytes, result.Data)
	assert.Equal(t, "image/png", result.ContentType)
}

func TestProjectScope_GetUserPhoto_WithThumbnail(t *testing.T) {
	t.Parallel()

	photoBytes := []byte{0xFF, 0xD8, 0xFF}

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/v1/data/my_project/userphoto")
		assert.Equal(t, "jane", r.URL.Query().Get("login"))
		assert.Equal(t, "true", r.URL.Query().Get("loadThumbnail"))

		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(photoBytes)
	}))

	result, err := c.Project("my_project").GetUserPhoto(
		t.Context(), "jane", client.WithUserPhotoThumbnail(true),
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, photoBytes, result.Data)
	assert.Equal(t, "image/jpeg", result.ContentType)
}

func TestProjectScope_GetUserPhoto_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40440, "message": "User not found."}`))

	_, err := c.Project("my_project").GetUserPhoto(t.Context(), "nonexistent")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
