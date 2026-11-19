package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"unicode/utf8"

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

// NewAPIError constructs an APIError from the HTTP status and parsed error body.
func NewAPIError(statusCode int, resp *types.ErrorRS) *APIError {
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

// IsServerError reports whether err is an API error with an HTTP 5xx status code.
// Cannot delegate to HasStatusCode, which matches an exact status code; this function checks any code in [500, 600).
func IsServerError(err error) bool {
	var sc statusCoder
	if !errors.As(err, &sc) {
		return false
	}

	code := sc.StatusCode()

	return code >= 500 && code < 600
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

// UnexpectedStatusError represents an HTTP error response that did not contain
// a parseable ErrorRS body. This typically occurs when ReportPortal returns its
// Spring Security OAuth envelope ({"error":"...","error_description":"..."}) for
// authentication failures, or when a proxy, gateway, or infrastructure component
// returns an HTML or plain-text error page. The raw response body is preserved
// so callers can inspect what the server actually returned.
type UnexpectedStatusError struct {
	statusCode int
	body       []byte
}

const maxBodyPreview = 256

func (e *UnexpectedStatusError) Error() string {
	if len(e.body) == 0 {
		return fmt.Sprintf("unexpected HTTP %d (no JSON error body)", e.statusCode)
	}

	if oauthMsg, ok := oauthErrorMessage(e.body); ok {
		if len(oauthMsg) > maxBodyPreview {
			oauthMsg = truncateString(oauthMsg, maxBodyPreview) + "..."
		}

		return fmt.Sprintf("HTTP %d: %s", e.statusCode, oauthMsg)
	}

	preview := e.body
	suffix := ""

	if len(preview) > maxBodyPreview {
		preview = preview[:maxBodyPreview]
		suffix = "..."
	}

	return fmt.Sprintf(
		"unexpected HTTP %d: %s%s",
		e.statusCode,
		sanitizeBodyPreview(preview),
		suffix,
	)
}

// ErrorFor returns an APIError when errRS carries information, otherwise an
// UnexpectedStatusError carrying the raw body. Use it wherever a decoded
// ErrorRS might be empty (e.g. the OAuth envelope for 401s) to ensure the
// response body is surfaced instead of "HTTP 401: [0]".
func ErrorFor(statusCode int, errRS *types.ErrorRS, body []byte) error {
	if !IsEmptyErrorRS(errRS) {
		return NewAPIError(statusCode, errRS)
	}

	return NewUnexpectedStatusError(statusCode, body)
}

// IsEmptyErrorRS reports whether an ErrorRS carries no information.
// The generated client decodes every JSON error body into ErrorRS, so an OAuth
// payload like {"error":"invalid_token",...} unmarshals into a zero value
// without error. Empty values must be treated as "no structured error body".
// Only errorCode and message determine emptiness; stackTrace alone is not
// enough to produce a useful APIError (it would render as "HTTP 500: [0] "),
// so it falls back to the raw body preview which already contains the trace.
func IsEmptyErrorRS(errRS *types.ErrorRS) bool {
	if errRS == nil {
		return true
	}

	return errRS.ErrorCode == 0 && errRS.Message == ""
}

// oauthEnvelope represents the Spring Security OAuth error shape that
// ReportPortal returns for authentication failures. It is distinct from the
// regular ErrorRS envelope and carries "error" and "error_description" fields.
type oauthEnvelope struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// oauthErrorMessage attempts to parse body as an OAuth envelope and returns
// a formatted message if successful.
func oauthErrorMessage(body []byte) (string, bool) {
	if len(body) == 0 {
		return "", false
	}

	var env oauthEnvelope

	err := json.Unmarshal(body, &env)
	if err != nil {
		return "", false
	}

	if env.Error == "" && env.ErrorDescription == "" {
		return "", false
	}

	if env.Error != "" && env.ErrorDescription != "" {
		return fmt.Sprintf("%s: %s", env.Error, env.ErrorDescription), true
	}

	if env.ErrorDescription != "" {
		return env.ErrorDescription, true
	}

	return env.Error, true
}

// ParseErrorBody attempts to unmarshal body as an ErrorRS and returns an
// APIError. If the body is not valid ErrorRS JSON, or if the decoded ErrorRS
// is empty (e.g. an OAuth envelope), it falls back to an UnexpectedStatusError
// so the raw body preview is preserved.
func ParseErrorBody(statusCode int, body []byte) error {
	var errRS types.ErrorRS

	err := json.Unmarshal(body, &errRS)
	if err != nil {
		return NewUnexpectedStatusError(statusCode, body)
	}

	return ErrorFor(statusCode, &errRS, body)
}

// NewUnexpectedStatusError constructs an UnexpectedStatusError for non-2xx
// responses without a JSON body.
func NewUnexpectedStatusError(statusCode int, body []byte) *UnexpectedStatusError {
	return &UnexpectedStatusError{
		statusCode: statusCode,
		body:       body,
	}
}

// StatusCode returns the HTTP status code from the response.
func (e *UnexpectedStatusError) StatusCode() int { return e.statusCode }

// Body returns the raw response body from the server.
func (e *UnexpectedStatusError) Body() []byte { return e.body }

// bodyCarrier is implemented by error types that preserve a raw response body.
type bodyCarrier interface {
	Body() []byte
}

// sanitizeBodyPreview returns a string representation of body that is safe to
// include in error messages. If the bytes are valid UTF-8 they are returned
// as-is; otherwise they are quoted with Go syntax to avoid corrupt output.
func sanitizeBodyPreview(body []byte) string {
	s := string(body)
	if utf8.ValidString(s) {
		return s
	}

	return fmt.Sprintf("%q", s)
}

// truncateString truncates s to at most max bytes without splitting a UTF-8 rune.
func truncateString(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	truncated := s[:limit]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}

	return truncated
}

// BodyOf extracts the raw response body from an error that carries one (e.g. UnexpectedStatusError). The second return
// value reports whether a body was found. This is the predicate-style accessor for body inspection, so callers don't
// need to assert on concrete error types.
func BodyOf(err error) ([]byte, bool) {
	var bc bodyCarrier
	if !errors.As(err, &bc) {
		return nil, false
	}

	return bc.Body(), true
}
