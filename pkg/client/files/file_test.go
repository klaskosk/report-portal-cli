package files_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/clienttest"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectScope_DownloadFile_Success(t *testing.T) {
	t.Parallel()

	fileContent := []byte("hello world attachment")

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/data/my_project/42")
			testutil.AssertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "text/plain")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write(fileContent)
		},
	))

	result, err := testClient.Project("my_project").DownloadFile(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, fileContent, result.Data)
	assert.Equal(t, "text/plain", result.ContentType)
}

func TestProjectScope_DownloadFile_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40440, "message": "File not found."}`))

	_, err := c.Project("my_project").DownloadFile(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestProjectScope_DownloadFile_Unauthorized(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").DownloadFile(t.Context(), 42)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectScope_DownloadFile_ImageContent(t *testing.T) {
	t.Parallel()

	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "image/png")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write(pngHeader)
		},
	))

	result, err := testClient.Project("my_project").DownloadFile(t.Context(), 123)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, pngHeader, result.Data)
	assert.Equal(t, "image/png", result.ContentType)
}

func TestFileStorageScope_GetMyPhoto_Success(t *testing.T) {
	t.Parallel()

	photoBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/data/photo")
			testutil.AssertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "image/jpeg")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write(photoBytes)
		},
	))

	result, err := testClient.Files().GetMyPhoto(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, photoBytes, result.Data)
	assert.Equal(t, "image/jpeg", result.ContentType)
}

func TestFileStorageScope_GetMyPhoto_WithThumbnail(t *testing.T) {
	t.Parallel()

	photoBytes := []byte{0x89, 0x50, 0x4E, 0x47}

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertPath(t, httpReq, "/api/v1/data/photo")
			testutil.AssertMethod(t, httpReq, http.MethodGet)
			assert.Equal(t, "true", httpReq.URL.Query().Get("loadThumbnail"))

			responseWriter.Header().Set("Content-Type", "image/png")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write(photoBytes)
		},
	))

	result, err := testClient.Files().GetMyPhoto(t.Context(), client.WithPhotoThumbnail(true))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, photoBytes, result.Data)
	assert.Equal(t, "image/png", result.ContentType)
}

func TestFileStorageScope_GetMyPhoto_Error(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Files().GetMyPhoto(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestFileStorageScope_UploadPhoto_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/data/photo")
			testutil.AssertMethod(t, httpReq, http.MethodPost)
			assert.Contains(t, httpReq.Header.Get("Content-Type"), "multipart/form-data")

			httpReq.Body = http.MaxBytesReader(responseWriter, httpReq.Body, 10<<20)

			//nolint:gosec // G120: test handler caps body via MaxBytesReader and bounded ParseMultipartForm memory.
			err := httpReq.ParseMultipartForm(10 << 20)
			if !assert.NoError(t, err) {
				return
			}

			_, header, err := httpReq.FormFile("file")
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, "avatar.png", header.Filename)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			const uploadPhotoMessageJSON = `{"message": "Photo has been uploaded successfully"}`

			_, _ = responseWriter.Write([]byte(uploadPhotoMessageJSON))
		},
	))

	result, err := testClient.Files().
		UploadPhoto(t.Context(), "avatar.png", strings.NewReader("fake-image-data"))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Contains(t, *result.Message, "successfully")
}

func TestFileStorageScope_UploadPhoto_Error(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusBadRequest,
		`{"errorCode": 4001, "message": "Incorrect file format."}`))

	_, err := c.Files().UploadPhoto(t.Context(), "bad.txt", strings.NewReader("not-an-image"))
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestFileStorageScope_DeletePhoto_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/data/photo")
			testutil.AssertMethod(t, httpReq, http.MethodDelete)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(
				`{"message": "Photo has been deleted successfully"}`,
			))
		},
	))

	result, err := testClient.Files().DeletePhoto(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Contains(t, *result.Message, "successfully")
}

func TestFileStorageScope_DeletePhoto_Error(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40440, "message": "Photo not found."}`))

	_, err := c.Files().DeletePhoto(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestFileStorageScope_CleanAttachments_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/data/clean")
			testutil.AssertMethod(t, httpReq, http.MethodPost)
			assert.Contains(t, httpReq.Header.Get("Content-Type"), "multipart/form-data")

			httpReq.Body = http.MaxBytesReader(responseWriter, httpReq.Body, 10<<20)

			//nolint:gosec // G120: test handler caps body via MaxBytesReader and bounded ParseMultipartForm memory.
			err := httpReq.ParseMultipartForm(10 << 20)
			if !assert.NoError(t, err) {
				return
			}

			_, header, err := httpReq.FormFile("file")
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, "attachments.csv", header.Filename)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{"message": "3 attachments removed"}`))
		},
	))

	csvData := strings.NewReader("id1\nid2\nid3\n")

	result, err := testClient.Files().CleanAttachments(t.Context(), "attachments.csv", csvData)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Contains(t, *result.Message, "attachments")
}

func TestFileStorageScope_CleanAttachments_Error(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Files().CleanAttachments(t.Context(), "data.csv", strings.NewReader("data"))
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectScope_GetUserPhoto_Success(t *testing.T) {
	t.Parallel()

	photoBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/data/my_project/userphoto")
			testutil.AssertMethod(t, httpReq, http.MethodGet)
			assert.Equal(t, "john_doe", httpReq.URL.Query().Get("login"))

			responseWriter.Header().Set("Content-Type", "image/png")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write(photoBytes)
		},
	))

	result, err := testClient.Project("my_project").GetUserPhoto(t.Context(), "john_doe")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, photoBytes, result.Data)
	assert.Equal(t, "image/png", result.ContentType)
}

func TestProjectScope_GetUserPhoto_WithThumbnail(t *testing.T) {
	t.Parallel()

	photoBytes := []byte{0xFF, 0xD8, 0xFF}

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertPath(t, httpReq, "/api/v1/data/my_project/userphoto")
			assert.Equal(t, "jane", httpReq.URL.Query().Get("login"))
			assert.Equal(t, "true", httpReq.URL.Query().Get("loadThumbnail"))

			responseWriter.Header().Set("Content-Type", "image/jpeg")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write(photoBytes)
		},
	))

	result, err := testClient.Project("my_project").GetUserPhoto(
		t.Context(), "jane", client.WithUserPhotoThumbnail(true),
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, photoBytes, result.Data)
	assert.Equal(t, "image/jpeg", result.ContentType)
}

func TestProjectScope_GetUserPhoto_Error(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40440, "message": "User not found."}`))

	_, err := c.Project("my_project").GetUserPhoto(t.Context(), "nonexistent")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
