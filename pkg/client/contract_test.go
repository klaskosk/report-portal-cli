package client_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Contract: each named predicate matches exactly one HTTP status code
// and rejects all others from the documented set.
func TestContract_PredicateCrossIsolation(t *testing.T) {
	t.Parallel()

	type predicateEntry struct {
		name      string
		fn        func(error) bool
		matchCode int
	}

	predicates := []predicateEntry{
		{"IsNotFound", client.IsNotFound, http.StatusNotFound},
		{"IsUnauthorized", client.IsUnauthorized, http.StatusUnauthorized},
		{"IsForbidden", client.IsForbidden, http.StatusForbidden},
		{"IsBadRequest", client.IsBadRequest, http.StatusBadRequest},
		{"IsConflict", client.IsConflict, http.StatusConflict},
	}

	for _, p := range predicates {
		for _, other := range predicates {
			expected := p.matchCode == other.matchCode

			t.Run(fmt.Sprintf("%s_vs_%d", p.name, other.matchCode), func(t *testing.T) {
				t.Parallel()

				err := client.NewTestAPIError(other.matchCode, &types.ErrorRS{
					ErrorCode: other.matchCode * 100,
					Message:   "test",
				})

				assert.Equal(t, expected, p.fn(err),
					"%s(status=%d) should be %v", p.name, other.matchCode, expected)
			})
		}
	}
}

// Contract: HasStatusCode works as a unified predicate across both error types
// for the full range of common HTTP error codes.
func TestContract_HasStatusCodeExtended(t *testing.T) {
	t.Parallel()

	codes := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusGone,
		http.StatusUnprocessableEntity,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}

	for _, code := range codes {
		t.Run(fmt.Sprintf("APIError/%d", code), func(t *testing.T) {
			t.Parallel()

			err := client.NewTestAPIError(code, &types.ErrorRS{
				ErrorCode: code, Message: "test",
			})
			assert.True(t, client.HasStatusCode(err, code))

			for _, other := range codes {
				if other != code {
					assert.False(t, client.HasStatusCode(err, other),
						"HasStatusCode(%d) should not match status %d", other, code)
				}
			}
		})

		t.Run(fmt.Sprintf("UnexpectedStatusError/%d", code), func(t *testing.T) {
			t.Parallel()

			err := client.NewTestUnexpectedStatusError(code)
			assert.True(t, client.HasStatusCode(err, code))
		})
	}
}

// Contract: HasErrorCode is specific to APIError and does not match
// UnexpectedStatusError (which has no application error code).
func TestContract_HasErrorCodeSpecificity(t *testing.T) {
	t.Parallel()

	t.Run("matches APIError", func(t *testing.T) {
		t.Parallel()

		err := client.NewTestAPIError(http.StatusNotFound, &types.ErrorRS{
			ErrorCode: 40410, Message: "not found",
		})
		assert.True(t, client.HasErrorCode(err, 40410))
		assert.False(t, client.HasErrorCode(err, 99999))
	})

	t.Run("rejects UnexpectedStatusError", func(t *testing.T) {
		t.Parallel()

		err := client.NewTestUnexpectedStatusError(http.StatusNotFound)
		assert.False(t, client.HasErrorCode(err, 404))
	})

	t.Run("rejects nil", func(t *testing.T) {
		t.Parallel()

		assert.False(t, client.HasErrorCode(nil, 40410))
	})

	t.Run("rejects plain error", func(t *testing.T) {
		t.Parallel()

		assert.False(t, client.HasErrorCode(errors.New("plain"), 40410))
	})
}

// Contract: predicates work correctly through multiple levels of wrapping.
func TestContract_DeepWrappingPreservesPredicates(t *testing.T) {
	t.Parallel()

	innerErr := client.NewTestAPIError(http.StatusNotFound, &types.ErrorRS{
		ErrorCode: 40410, Message: "not found",
	})

	wrapped := fmt.Errorf("level 3: %w",
		fmt.Errorf("level 2: %w",
			fmt.Errorf("level 1: %w", innerErr)))

	assert.True(t, client.IsNotFound(wrapped))
	assert.True(t, client.HasStatusCode(wrapped, http.StatusNotFound))
	assert.True(t, client.HasErrorCode(wrapped, 40410))
	assert.False(t, client.IsUnauthorized(wrapped))
}

// Contract: APIError.Error() format is stable and contains cause-only information
// (no operation context — that belongs to the caller via fmt.Errorf wrapping).
func TestContract_APIErrorMessageFormat(t *testing.T) {
	t.Parallel()

	t.Run("with response body", func(t *testing.T) {
		t.Parallel()

		err := client.NewTestAPIError(http.StatusNotFound, &types.ErrorRS{
			ErrorCode: 40410,
			Message:   "Launch 'abc' not found.",
		})

		var apiErr *client.APIError
		require.ErrorAs(t, err, &apiErr)

		msg := apiErr.Error()
		assert.Contains(t, msg, "HTTP 404", "HTTP status")
		assert.Contains(t, msg, "40410", "error code")
		assert.Contains(t, msg, "Launch 'abc' not found.", "error message")
	})

	t.Run("without response body", func(t *testing.T) {
		t.Parallel()

		err := client.NewTestAPIError(http.StatusNotFound, nil)

		var apiErr *client.APIError
		require.ErrorAs(t, err, &apiErr)

		msg := apiErr.Error()
		assert.Contains(t, msg, "HTTP 404", "HTTP status")
		assert.NotContains(t, msg, "[0]", "should not contain zero error code bracket")
	})
}

// Contract: UnexpectedStatusError.Error() format is stable.
func TestContract_UnexpectedStatusErrorMessageFormat(t *testing.T) {
	t.Parallel()

	err := client.NewTestUnexpectedStatusError(http.StatusBadGateway)

	var unexpectedErr *client.UnexpectedStatusError
	require.ErrorAs(t, err, &unexpectedErr)

	msg := unexpectedErr.Error()
	assert.Contains(t, msg, "unexpected HTTP 502", "HTTP status with 'unexpected' prefix")
	assert.Contains(t, msg, "no JSON error body", "clarification about missing body")
}
