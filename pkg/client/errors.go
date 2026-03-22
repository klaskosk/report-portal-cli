package client

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// APIError represents a structured error response from the Report Portal API.
//
// Callers should prefer the predicate functions (IsNotFound, IsUnauthorized, etc.) to inspect errors rather than
// asserting on this type directly. For cases not covered by a predicate, use errors.As to extract the APIError and
// call its accessor methods.
type APIError struct {
	statusCode int
	response   *types.ErrorRS
}

func (e *APIError) Error() string {
	if e.response == nil {
		return fmt.Sprintf("HTTP %d", e.statusCode)
	}

	return fmt.Sprintf(
		"HTTP %d: [%d] %s",
		e.statusCode,
		e.response.ErrorCode,
		e.response.Message,
	)
}

// newAPIError constructs an APIError from the HTTP status and parsed error body.
func newAPIError(statusCode int, resp *types.ErrorRS) *APIError {
	return &APIError{
		statusCode: statusCode,
		response:   resp,
	}
}

// StatusCode returns the HTTP status code from the response.
func (e *APIError) StatusCode() int { return e.statusCode }

// ErrorCode returns the Report Portal application error code from the response body.
// Returns 0 if no structured response body was available.
func (e *APIError) ErrorCode() int {
	if e.response == nil {
		return 0
	}

	return e.response.ErrorCode
}

// Message returns the human-readable error message from the response body.
// Returns an empty string if no structured response body was available.
func (e *APIError) Message() string {
	if e.response == nil {
		return ""
	}

	return e.response.Message
}

// IsNotFound reports whether err is an API error with HTTP 404 status.
func IsNotFound(err error) bool {
	return HasStatusCode(err, http.StatusNotFound)
}

// IsUnauthorized reports whether err is an API error with HTTP 401 status.
func IsUnauthorized(err error) bool {
	return HasStatusCode(err, http.StatusUnauthorized)
}

// IsForbidden reports whether err is an API error with HTTP 403 status.
func IsForbidden(err error) bool {
	return HasStatusCode(err, http.StatusForbidden)
}

// IsBadRequest reports whether err is an API error with HTTP 400 status.
func IsBadRequest(err error) bool {
	return HasStatusCode(err, http.StatusBadRequest)
}

// IsConflict reports whether err is an API error with HTTP 409 status.
func IsConflict(err error) bool {
	return HasStatusCode(err, http.StatusConflict)
}

// statusCoder is implemented by error types that carry an HTTP status code.
type statusCoder interface {
	StatusCode() int
}

// HasStatusCode reports whether err is an API error whose HTTP status code matches code. This is the general-purpose
// escape hatch for status codes not covered by a named predicate. It matches both APIError (structured JSON error
// bodies) and UnexpectedStatusError (non-JSON or missing error bodies).
func HasStatusCode(err error, code int) bool {
	var sc statusCoder

	return errors.As(err, &sc) && sc.StatusCode() == code
}

// HasErrorCode reports whether err is an API error whose Report Portal application error code matches code.
func HasErrorCode(err error, code int) bool {
	var apiError *APIError

	return errors.As(err, &apiError) && apiError.ErrorCode() == code
}

// UnexpectedStatusError represents an HTTP error response that did not contain a parseable JSON error body. This
// typically occurs when a proxy, gateway, or infrastructure component returns an HTML or plain-text error page.
type UnexpectedStatusError struct {
	statusCode int
}

func (e *UnexpectedStatusError) Error() string {
	return fmt.Sprintf("unexpected HTTP %d (no JSON error body)", e.statusCode)
}

// newUnexpectedStatusError constructs an UnexpectedStatusError for non-2xx responses without a JSON body.
func newUnexpectedStatusError(statusCode int) *UnexpectedStatusError {
	return &UnexpectedStatusError{
		statusCode: statusCode,
	}
}

// StatusCode returns the HTTP status code from the response.
func (e *UnexpectedStatusError) StatusCode() int { return e.statusCode }
