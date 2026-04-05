package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// FileContent holds the raw bytes and metadata of a downloaded file.
type FileContent struct {
	Data          []byte
	ContentType   string
	ContentLength int64
}

// FileStorageScope provides instance-level file storage operations such as
// user photo management and attachment cleanup.
type FileStorageScope struct {
	client *Client
}

// GetMyPhotoOption configures optional parameters for GetMyPhoto.
type GetMyPhotoOption interface {
	applyGetMyPhoto(*api.GetMyPhotoParams)
}

type getMyPhotoOptionFunc func(*api.GetMyPhotoParams)

func (f getMyPhotoOptionFunc) applyGetMyPhoto(params *api.GetMyPhotoParams) {
	f(params)
}

// WithPhotoThumbnail requests a thumbnail version of the photo.
func WithPhotoThumbnail(load bool) GetMyPhotoOption {
	return getMyPhotoOptionFunc(func(params *api.GetMyPhotoParams) {
		params.LoadThumbnail = &load
	})
}

// GetMyPhoto retrieves the current user's profile photo as raw image bytes.
func (s *FileStorageScope) GetMyPhoto(ctx context.Context, options ...GetMyPhotoOption) (*FileContent, error) {
	logger := s.client.logger.With(slog.String("operation", "GetMyPhoto"))
	logger.InfoContext(ctx, "API request")

	params := &api.GetMyPhotoParams{}
	for _, opt := range options {
		opt.applyGetMyPhoto(params)
	}

	response, err := s.client.raw.GetMyPhotoWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get my photo: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return extractFileContent(response.StatusCode(), response.JSONDefault, response.Body, response.HTTPResponse, "get my photo")
}

// UploadPhoto uploads a photo for the current user. The filename is used in the
// multipart form field and data provides the photo content.
func (s *FileStorageScope) UploadPhoto(ctx context.Context, filename string, data io.Reader) (*types.OperationCompletionRS, error) {
	logger := s.client.logger.With(slog.String("operation", "UploadPhoto"))
	logger.InfoContext(ctx, "API request")

	body, contentType, err := buildFileMultipart(filename, data)
	if err != nil {
		return nil, fmt.Errorf("upload photo: %w", err)
	}

	response, err := s.client.raw.UploadPhotoWithBodyWithResponse(ctx, contentType, body)
	if err != nil {
		return nil, fmt.Errorf("upload photo: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("upload photo: %w", err)
	}

	return result, nil
}

// DeletePhoto deletes the current user's profile photo.
func (s *FileStorageScope) DeletePhoto(ctx context.Context) (*types.OperationCompletionRS, error) {
	logger := s.client.logger.With(slog.String("operation", "DeletePhoto"))
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.DeletePhotoWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("delete photo: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("delete photo: %w", err)
	}

	return result, nil
}

// CleanAttachments removes attachments referenced in a CSV file. The filename
// is used in the multipart form field and csvData provides the CSV content.
func (s *FileStorageScope) CleanAttachments(ctx context.Context, filename string, csvData io.Reader) (*types.OperationCompletionRS, error) {
	logger := s.client.logger.With(slog.String("operation", "CleanAttachments"))
	logger.InfoContext(ctx, "API request")

	body, contentType, err := buildFileMultipart(filename, csvData)
	if err != nil {
		return nil, fmt.Errorf("clean attachments: %w", err)
	}

	response, err := s.client.raw.RemoveAttachmentsByCsvWithBodyWithResponse(ctx, contentType, body)
	if err != nil {
		return nil, fmt.Errorf("clean attachments: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("clean attachments: %w", err)
	}

	return result, nil
}

// GetUserPhotoOption configures optional parameters for GetUserPhoto.
type GetUserPhotoOption interface {
	applyGetUserPhoto(*api.GetUserPhotoParams)
}

type getUserPhotoOptionFunc func(*api.GetUserPhotoParams)

func (f getUserPhotoOptionFunc) applyGetUserPhoto(params *api.GetUserPhotoParams) {
	f(params)
}

// WithUserPhotoThumbnail requests a thumbnail version of the user's photo.
func WithUserPhotoThumbnail(load bool) GetUserPhotoOption {
	return getUserPhotoOptionFunc(func(params *api.GetUserPhotoParams) {
		params.LoadThumbnail = &load
	})
}

// GetUserPhoto retrieves a user's profile photo by login within this project.
func (p *ProjectScope) GetUserPhoto(ctx context.Context, login string, options ...GetUserPhotoOption) (*FileContent, error) {
	logger := p.operationLogger("GetUserPhoto", slog.String("login", login))
	logger.InfoContext(ctx, "API request")

	params := &api.GetUserPhotoParams{Login: login}
	for _, opt := range options {
		opt.applyGetUserPhoto(params)
	}

	response, err := p.client.raw.GetUserPhotoWithResponse(ctx, p.projectName, params)
	if err != nil {
		return nil, fmt.Errorf("get user photo: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return extractFileContent(response.StatusCode(), response.JSONDefault, response.Body, response.HTTPResponse, "get user photo")
}

// DownloadFile downloads a file attachment by its numeric data store ID
// (from a log entry's binaryContent.Id field).
func (p *ProjectScope) DownloadFile(
	ctx context.Context,
	dataID int64,
) (*FileContent, error) {
	logger := p.operationLogger("DownloadFile", slog.Int64("dataId", dataID))
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetFileWithResponse(
		ctx,
		p.projectName,
		dataID,
	)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return extractFileContent(response.StatusCode(), response.JSONDefault, response.Body, response.HTTPResponse, "download file")
}

// extractFileContent handles the common pattern for non-JSON file download
// responses: check for API error body, validate status code, then extract
// raw bytes and content metadata from the HTTP response.
func extractFileContent(
	statusCode int,
	jsonDefault *types.ErrorRS,
	body []byte,
	httpResp *http.Response,
	operation string,
) (*FileContent, error) {
	if jsonDefault != nil {
		return nil, fmt.Errorf("%s: %w", operation, newAPIError(statusCode, jsonDefault))
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("%s: %w", operation, newUnexpectedStatusError(statusCode, body))
	}

	return &FileContent{
		Data:          body,
		ContentType:   httpResp.Header.Get("Content-Type"),
		ContentLength: httpResp.ContentLength,
	}, nil
}

// buildFileMultipart constructs a multipart form body with a single "file"
// field containing the given data. It returns the encoded body, the
// Content-Type header value, and any error.
func buildFileMultipart(filename string, data io.Reader) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", fmt.Errorf("creating file part: %w", err)
	}

	if _, err := io.Copy(part, data); err != nil {
		return nil, "", fmt.Errorf("writing file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("closing multipart writer: %w", err)
	}

	return &buf, writer.FormDataContentType(), nil
}
