package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// typedResponse is implemented by generated response structs that have a
// single 2xx success field. The codegen template generates these methods.
type typedResponse[T any] interface {
	StatusCode() int
	GetBody() []byte
	GetJSONDefault() *types.ErrorRS
	GetSuccess() *T
}

// voidResponse is implemented by generated response structs for endpoints
// that return no success body (e.g. DELETE with 204 No Content).
type voidResponse interface {
	StatusCode() int
	GetBody() []byte
	GetJSONDefault() *types.ErrorRS
}

// execute handles the full lifecycle of a typed API call: it wraps transport
// errors, logs the response status, checks for structured and unstructured
// error bodies, and validates the success payload. All returned errors are
// prefixed with opName so callers don't need an additional wrapping layer.
func execute[T any](
	ctx context.Context,
	logger *slog.Logger,
	opName string,
	resp typedResponse[T],
	err error,
) (*T, error) {
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opName, err)
	}

	logger.DebugContext(ctx, "API response", "status", resp.StatusCode())

	if jsonDefault := resp.GetJSONDefault(); jsonDefault != nil {
		return nil, fmt.Errorf("%s: %w", opName, newAPIError(resp.StatusCode(), jsonDefault))
	}

	code := resp.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("%s: %w", opName, newUnexpectedStatusError(code, resp.GetBody()))
	}

	success := resp.GetSuccess()
	if success == nil {
		return nil, fmt.Errorf(
			"%s: API returned success status (%d) but no valid response body was parsed",
			opName, code,
		)
	}

	return success, nil
}

// executeVoid handles the full lifecycle of a void API call (e.g. DELETE
// returning 204 No Content). It mirrors execute but skips success-body
// validation since no response payload is expected.
func executeVoid(
	ctx context.Context,
	logger *slog.Logger,
	opName string,
	resp voidResponse,
	err error,
) error {
	if err != nil {
		return fmt.Errorf("%s: %w", opName, err)
	}

	logger.DebugContext(ctx, "API response", "status", resp.StatusCode())

	if jsonDefault := resp.GetJSONDefault(); jsonDefault != nil {
		return fmt.Errorf("%s: %w", opName, newAPIError(resp.StatusCode(), jsonDefault))
	}

	code := resp.StatusCode()
	if code < 200 || code >= 300 {
		return fmt.Errorf("%s: %w", opName, newUnexpectedStatusError(code, resp.GetBody()))
	}

	return nil
}

// handleV2Response handles responses from endpoints that negotiate a versioned
// (v2) content type but may fall back to the legacy JSON shape. It returns the
// v2 success value when present, rejects the legacy shape with an error, and
// otherwise checks for structured error bodies and unexpected status codes.
func handleV2Response[R interface{ StatusCode() int }, T any](
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

	legacyReflect := reflect.ValueOf(legacySuccess)
	switch legacyReflect.Kind() {
	case reflect.Invalid:
		return false
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.Array, reflect.String, reflect.Struct,
		reflect.UnsafePointer:
		return true
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return !legacyReflect.IsNil()
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
