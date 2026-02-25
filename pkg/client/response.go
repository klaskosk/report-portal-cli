package client

import (
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// apiResponse is implemented by all oapi-codegen response types that carry
// a JSONDefault error body.
type apiResponse interface {
	StatusCode() int
}

// handleResponse checks the error response body and returns the success value. It handles the "check JSONDefault, then
// return success" pattern that every scope method uses. If the status code is outside 2xx and there is no parsed JSON
// error body (e.g. a proxy returning HTML), it returns an UnexpectedStatusError.
func handleResponse[R apiResponse, T any](
	response R,
	operation string,
	jsonDefault *types.ErrorRS,
	success *T,
) (*T, error) {
	if jsonDefault != nil {
		return nil, newAPIError(operation, response.StatusCode(), jsonDefault)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, newUnexpectedStatusError(operation, code)
	}

	if success == nil {
		return nil, fmt.Errorf("%s: API returned success status (%d) but no valid response body was parsed", operation, code)
	}

	return success, nil
}

// operationLogger returns a logger pre-populated with the project name and the
// given operation, plus any additional attributes.
func (p *ProjectScope) operationLogger(operation string, attrs ...slog.Attr) *slog.Logger {
	logger := p.logger().With(slog.String("operation", operation))
	for _, attr := range attrs {
		logger = logger.With(attr)
	}

	return logger
}
