package core

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// TypedResponse is implemented by generated response structs that have a
// single 2xx success field. The codegen template generates these methods.
type TypedResponse[T any] interface {
	StatusCode() int
	GetBody() []byte
	GetJSONDefault() *types.ErrorRS
	GetSuccess() *T
}

// VoidResponse is implemented by generated response structs for endpoints
// that return no success body (e.g. DELETE with 204 No Content).
type VoidResponse interface {
	StatusCode() int
	GetBody() []byte
	GetJSONDefault() *types.ErrorRS
}

// Execute handles the full lifecycle of a typed API call: it wraps transport
// errors, logs the response status, checks for structured and unstructured
// error bodies, and validates the success payload. All returned errors are
// prefixed with opName so callers don't need an additional wrapping layer.
func Execute[T any](
	ctx context.Context,
	logger *slog.Logger,
	opName string,
	resp TypedResponse[T],
	err error,
) (*T, error) {
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opName, err)
	}

	logger.DebugContext(ctx, "API response", "status", resp.StatusCode())

	if jsonDefault := resp.GetJSONDefault(); jsonDefault != nil {
		return nil, fmt.Errorf("%s: %w", opName, NewAPIError(resp.StatusCode(), jsonDefault))
	}

	code := resp.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("%s: %w", opName, NewUnexpectedStatusError(code, resp.GetBody()))
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

// ExecuteVoid handles the full lifecycle of a void API call (e.g. DELETE
// returning 204 No Content). It mirrors Execute but skips success-body
// validation since no response payload is expected.
func ExecuteVoid(
	ctx context.Context,
	logger *slog.Logger,
	opName string,
	resp VoidResponse,
	err error,
) error {
	if err != nil {
		return fmt.Errorf("%s: %w", opName, err)
	}

	logger.DebugContext(ctx, "API response", "status", resp.StatusCode())

	if jsonDefault := resp.GetJSONDefault(); jsonDefault != nil {
		return fmt.Errorf("%s: %w", opName, NewAPIError(resp.StatusCode(), jsonDefault))
	}

	code := resp.StatusCode()
	if code < 200 || code >= 300 {
		return fmt.Errorf("%s: %w", opName, NewUnexpectedStatusError(code, resp.GetBody()))
	}

	return nil
}
