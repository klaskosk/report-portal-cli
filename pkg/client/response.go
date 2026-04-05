package client

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// apiResponse is implemented by all oapi-codegen response types that carry a JSONDefault error body.
type apiResponse interface {
	StatusCode() int
}

// handleResponse checks the error response body and returns the success value. It handles the "check JSONDefault, then
// return success" pattern that every scope method uses. If the status code is outside 2xx and there is no parsed JSON
// error body (e.g. a proxy returning HTML), it returns an UnexpectedStatusError with the raw body preserved. Callers
// are responsible for wrapping the returned error with operation context via fmt.Errorf.
func handleResponse[R apiResponse, T any](
	response R,
	jsonDefault *types.ErrorRS,
	success *T,
	body []byte,
) (*T, error) {
	if jsonDefault != nil {
		return nil, newAPIError(response.StatusCode(), jsonDefault)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, newUnexpectedStatusError(code, body)
	}

	if success == nil {
		return nil, fmt.Errorf(
			"API returned success status (%d) but no valid response body was parsed",
			code,
		)
	}

	return success, nil
}

// handleVoidResponse checks for API errors on endpoints that return no success body (e.g. DELETE).
func handleVoidResponse[R apiResponse](response R, jsonDefault *types.ErrorRS, body []byte) error {
	if jsonDefault != nil {
		return newAPIError(response.StatusCode(), jsonDefault)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return newUnexpectedStatusError(code, body)
	}

	return nil
}

// handleV2Response handles responses from endpoints that negotiate a versioned
// (v2) content type but may fall back to the legacy JSON shape. It returns the
// v2 success value when present, rejects the legacy shape with an error, and
// otherwise delegates to the same error handling as handleResponse.
func handleV2Response[R apiResponse, T any](
	response R,
	jsonDefault *types.ErrorRS,
	v2Success *T,
	legacySuccess any,
	body []byte,
) (*T, error) {
	if v2Success != nil {
		return v2Success, nil
	}

	if legacyPayloadPresent(legacySuccess) {
		return nil, errors.New("API returned legacy JSON shape")
	}

	if jsonDefault != nil {
		return nil, newAPIError(response.StatusCode(), jsonDefault)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, newUnexpectedStatusError(code, body)
	}

	return nil, fmt.Errorf(
		"API returned success status (%d) but no valid response body was parsed",
		code,
	)
}

// legacyPayloadPresent reports whether legacySuccess holds a non-nil value,
// accounting for typed nil pointers stored in an any interface.
func legacyPayloadPresent(legacySuccess any) bool {
	if legacySuccess == nil {
		return false
	}
	rv := reflect.ValueOf(legacySuccess)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return !rv.IsNil()
	default:
		return true
	}
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
