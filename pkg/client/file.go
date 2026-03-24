package client

import (
	"context"
	"fmt"
	"log/slog"
)

// FileContent holds the raw bytes and metadata of a downloaded file.
type FileContent struct {
	Data          []byte
	ContentType   string
	ContentLength int64
}

// DownloadFile downloads a file attachment by its numeric data store ID
// (from a log entry's binaryContent.Id field).
func (p *ProjectScope) DownloadFile(
	ctx context.Context,
	dataID int64,
) (*FileContent, error) {
	logger := p.operationLogger("DownloadFile", slog.Int64("dataId", dataID))
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetFileUsingGETWithResponse(
		ctx,
		p.projectName,
		dataID,
	)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.JSONDefault != nil {
		return nil, fmt.Errorf(
			"download file: %w",
			newAPIError(response.StatusCode(), response.JSONDefault),
		)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("download file: %w", newUnexpectedStatusError(code))
	}

	httpResp := response.HTTPResponse
	contentType := httpResp.Header.Get("Content-Type")
	contentLength := httpResp.ContentLength

	return &FileContent{
		Data:          response.Body,
		ContentType:   contentType,
		ContentLength: contentLength,
	}, nil
}
