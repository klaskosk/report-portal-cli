package client

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeResponse mimics an oapi-codegen response struct with a JSONDefault
// error body and a typed success field.
type fakeResponse struct {
	statusCode  int
	JSONDefault *types.ErrorRS
	JSON200     *string
}

func (f fakeResponse) StatusCode() int { return f.statusCode }

func TestHandleResponse_Success(t *testing.T) {
	t.Parallel()

	want := "ok"
	resp := fakeResponse{statusCode: 200, JSON200: &want}

	got, err := handleResponse(resp, resp.JSONDefault, resp.JSON200)
	require.NoError(t, err)
	assert.Equal(t, &want, got)
}

func TestHandleResponse_Error(t *testing.T) {
	t.Parallel()

	errBody := &types.ErrorRS{ErrorCode: 40410, Message: "Not found"}
	resp := fakeResponse{statusCode: 404, JSONDefault: errBody}

	got, err := handleResponse(resp, resp.JSONDefault, resp.JSON200)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, IsNotFound(err))

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.StatusCode())
	assert.Equal(t, 40410, apiErr.ErrorCode())
}

func TestHandleResponse_NilSuccess(t *testing.T) {
	t.Parallel()

	resp := fakeResponse{statusCode: 200, JSON200: nil}

	got, err := handleResponse(resp, resp.JSONDefault, resp.JSON200)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(
		t,
		err.Error(),
		"API returned success status (200) but no valid response body was parsed",
	)
}

func TestHandleResponse_NonJSONError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
	}{
		{"404 without JSON body", 404},
		{"500 without JSON body", 500},
		{"502 without JSON body", 502},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resp := fakeResponse{statusCode: testCase.statusCode, JSONDefault: nil}

			got, err := handleResponse(resp, resp.JSONDefault, resp.JSON200)
			require.Error(t, err)
			assert.Nil(t, got)
			assert.True(t, HasStatusCode(err, testCase.statusCode))

			var unexpectedErr *UnexpectedStatusError
			require.ErrorAs(t, err, &unexpectedErr)
			assert.Equal(t, testCase.statusCode, unexpectedErr.StatusCode())
			assert.Contains(t, err.Error(), "unexpected HTTP")
		})
	}
}

func TestHandleResponse_2xxWithNilJSON(t *testing.T) {
	t.Parallel()

	// 204 No Content still falls under the nil check in this client, because
	// Report Portal doesn't actually return 204s in any supported endpoints.
	resp := fakeResponse{statusCode: 204, JSON200: nil}

	got, err := handleResponse(resp, resp.JSONDefault, resp.JSON200)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(
		t,
		err.Error(),
		"API returned success status (204) but no valid response body was parsed",
	)
}
