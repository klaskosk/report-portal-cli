package core

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// HandleV2Response handles responses from endpoints that negotiate a versioned
// (v2) content type but may fall back to the legacy JSON shape. It returns the
// v2 success value when present, rejects the legacy shape with an error, and
// otherwise checks for structured error bodies and unexpected status codes.
func HandleV2Response[R interface{ StatusCode() int }, T any](
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
		return nil, NewAPIError(response.StatusCode(), jsonDefault)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, NewUnexpectedStatusError(code, body)
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
