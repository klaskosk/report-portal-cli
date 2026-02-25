package client

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
)

// FileContent holds the raw bytes and metadata of a downloaded file.
type FileContent struct {
	Data          []byte
	ContentType   string
	ContentLength int64
}

// DownloadFile downloads a file attachment by its data store ID.
// The dataID is a string representation of a numeric ID from a log entry's binaryContent.id field.
func (p *ProjectScope) DownloadFile(
	ctx context.Context,
	dataID string,
) (*FileContent, error) {
	logger := p.operationLogger("DownloadFile", slog.String("dataId", dataID))
	logger.InfoContext(ctx, "API request")

	numericID, err := strconv.ParseInt(dataID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid data ID %q: must be numeric: %w", dataID, err)
	}

	response, err := p.client.raw.GetFileUsingGETWithResponse(
		ctx,
		p.projectName,
		numericID,
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
