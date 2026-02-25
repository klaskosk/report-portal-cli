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

func TestAPIError_Error(t *testing.T) {
	t.Parallel()

	err := client.NewTestAPIError(http.StatusNotFound, &types.ErrorRS{
		ErrorCode: 40410,
		Message:   "Launch '999' not found.",
	})

	assert.Equal(t, "HTTP 404: [40410] Launch '999' not found.", err.Error())
}

func TestAPIError_NilResponse(t *testing.T) {
	t.Parallel()

	err := client.NewTestAPIError(http.StatusNotFound, nil)

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)

	assert.Equal(t, "HTTP 404", apiErr.Error())
	assert.Equal(t, 0, apiErr.ErrorCode())
	assert.Empty(t, apiErr.Message())
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode())
}

func TestAPIError_Accessors(t *testing.T) {
	t.Parallel()

	err := client.NewTestAPIError(http.StatusForbidden, &types.ErrorRS{
		ErrorCode: 4003,
		Message:   "Access denied.",
	})

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)

	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode())
	assert.Equal(t, 4003, apiErr.ErrorCode())
	assert.Equal(t, "Access denied.", apiErr.Message())
}

func TestErrorPredicates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		errorCode  int
		predicate  func(error) bool
	}{
		{"IsNotFound", http.StatusNotFound, 40410, client.IsNotFound},
		{"IsUnauthorized", http.StatusUnauthorized, 4001, client.IsUnauthorized},
		{"IsForbidden", http.StatusForbidden, 4003, client.IsForbidden},
		{"IsBadRequest", http.StatusBadRequest, 40001, client.IsBadRequest},
		{"IsConflict", http.StatusConflict, 4091, client.IsConflict},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			apiErr := client.NewTestAPIError(
				testCase.statusCode,
				&types.ErrorRS{ErrorCode: testCase.errorCode, Message: "test"},
			)

			assert.True(t, testCase.predicate(apiErr))
			assert.False(t, testCase.predicate(errors.New("plain error")))
			assert.False(t, testCase.predicate(nil))
		})
	}
}

func TestHasStatusCode(t *testing.T) {
	t.Parallel()

	apiErr := client.NewTestAPIError(
		http.StatusInternalServerError,
		&types.ErrorRS{ErrorCode: 5000, Message: "internal"},
	)

	assert.True(t, client.HasStatusCode(apiErr, http.StatusInternalServerError))
	assert.False(t, client.HasStatusCode(apiErr, http.StatusNotFound))
}

func TestHasErrorCode(t *testing.T) {
	t.Parallel()

	apiErr := client.NewTestAPIError(
		http.StatusNotFound,
		&types.ErrorRS{ErrorCode: 40410, Message: "not found"},
	)

	assert.True(t, client.HasErrorCode(apiErr, 40410))
	assert.False(t, client.HasErrorCode(apiErr, 99999))
}

func TestPredicates_WrappedError(t *testing.T) {
	t.Parallel()

	apiErr := client.NewTestAPIError(
		http.StatusNotFound,
		&types.ErrorRS{ErrorCode: 40410, Message: "not found"},
	)
	wrapped := fmt.Errorf("get launch by id: %w", apiErr)

	assert.True(t, client.IsNotFound(wrapped))
	assert.True(t, client.HasStatusCode(wrapped, http.StatusNotFound))
	assert.True(t, client.HasErrorCode(wrapped, 40410))
}

func TestUnexpectedStatusError_Error(t *testing.T) {
	t.Parallel()

	err := client.NewTestUnexpectedStatusError(http.StatusBadGateway)

	assert.Equal(t, "unexpected HTTP 502 (no JSON error body)", err.Error())
}

func TestUnexpectedStatusError_Accessors(t *testing.T) {
	t.Parallel()

	err := client.NewTestUnexpectedStatusError(http.StatusNotFound)

	var unexpectedErr *client.UnexpectedStatusError
	require.ErrorAs(t, err, &unexpectedErr)

	assert.Equal(t, http.StatusNotFound, unexpectedErr.StatusCode())
}

func TestHasStatusCode_MatchesUnexpectedStatusError(t *testing.T) {
	t.Parallel()

	err := client.NewTestUnexpectedStatusError(http.StatusBadGateway)

	assert.True(t, client.HasStatusCode(err, http.StatusBadGateway))
	assert.False(t, client.HasStatusCode(err, http.StatusNotFound))
}

func TestPredicates_MatchUnexpectedStatusError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		predicate  func(error) bool
	}{
		{"IsNotFound", http.StatusNotFound, client.IsNotFound},
		{"IsUnauthorized", http.StatusUnauthorized, client.IsUnauthorized},
		{"IsForbidden", http.StatusForbidden, client.IsForbidden},
		{"IsBadRequest", http.StatusBadRequest, client.IsBadRequest},
		{"IsConflict", http.StatusConflict, client.IsConflict},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := client.NewTestUnexpectedStatusError(testCase.statusCode)
			assert.True(t, testCase.predicate(err))
		})
	}
}

func TestPredicates_WrappedUnexpectedStatusError(t *testing.T) {
	t.Parallel()

	err := client.NewTestUnexpectedStatusError(http.StatusNotFound)
	wrapped := fmt.Errorf("get launch by id: %w", err)

	assert.True(t, client.IsNotFound(wrapped))
	assert.True(t, client.HasStatusCode(wrapped, http.StatusNotFound))
}
