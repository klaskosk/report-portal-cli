package client

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeResponse mimics an oapi-codegen response struct with a JSONDefault
// error body, a typed success field, and the raw response body.
type fakeResponse struct {
	statusCode  int
	JSONDefault *types.ErrorRS
	JSON200     *string
	Body        []byte
}

func (f fakeResponse) StatusCode() int { return f.statusCode }

// fakeV2Response mimics a response struct with both a v2 content-negotiated
// field and a legacy JSON200 field.
type fakeV2Response struct {
	statusCode int
	JSONDefault *types.ErrorRS
	V2JSON200   *string
	JSON200     *string
	Body        []byte
}

func (f fakeV2Response) StatusCode() int { return f.statusCode }

func TestHandleResponse_Success(t *testing.T) {
	t.Parallel()

	want := "ok"
	resp := fakeResponse{statusCode: 200, JSON200: &want}

	got, err := handleResponse(resp, resp.JSONDefault, resp.JSON200, resp.Body)
	require.NoError(t, err)
	assert.Equal(t, &want, got)
}

func TestHandleResponse_Error(t *testing.T) {
	t.Parallel()

	errBody := &types.ErrorRS{ErrorCode: 40410, Message: "Not found"}
	resp := fakeResponse{statusCode: 404, JSONDefault: errBody}

	got, err := handleResponse(resp, resp.JSONDefault, resp.JSON200, resp.Body)
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

	got, err := handleResponse(resp, resp.JSONDefault, resp.JSON200, resp.Body)
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
		body       []byte
	}{
		{"404 without JSON body", 404, nil},
		{"500 without JSON body", 500, nil},
		{"502 without JSON body", 502, []byte("<html>Bad Gateway</html>")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resp := fakeResponse{
				statusCode:  testCase.statusCode,
				JSONDefault: nil,
				Body:        testCase.body,
			}

			got, err := handleResponse(resp, resp.JSONDefault, resp.JSON200, resp.Body)
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

func TestHandleResponse_NonJSONErrorPreservesBody(t *testing.T) {
	t.Parallel()

	body := []byte("Full authentication is required to access this resource")
	resp := fakeResponse{statusCode: 401, JSONDefault: nil, Body: body}

	_, err := handleResponse(resp, resp.JSONDefault, resp.JSON200, resp.Body)
	require.Error(t, err)

	gotBody, ok := BodyOf(err)
	require.True(t, ok)
	assert.Equal(t, body, gotBody)
	assert.Contains(t, err.Error(), "Full authentication is required")
}

func TestHandleResponse_2xxWithNilJSON(t *testing.T) {
	t.Parallel()

	// 204 No Content still falls under the nil check in this client, because
	// Report Portal doesn't actually return 204s in any supported endpoints.
	resp := fakeResponse{statusCode: 204, JSON200: nil}

	got, err := handleResponse(resp, resp.JSONDefault, resp.JSON200, resp.Body)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(
		t,
		err.Error(),
		"API returned success status (204) but no valid response body was parsed",
	)
}

func TestHandleV2Response(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		statusCode    int
		jsonDefault   *types.ErrorRS
		v2Success     *string
		legacySuccess *string
		body          []byte
		wantResult    *string
		wantErr       string
	}{
		{
			name:       "v2 success",
			statusCode: 200,
			v2Success:  ptr("v2-ok"),
			wantResult: ptr("v2-ok"),
		},
		{
			name:          "legacy rejection",
			statusCode:    200,
			legacySuccess: ptr("legacy-ok"),
			wantErr:       "API returned legacy JSON shape",
		},
		{
			name:        "API error",
			statusCode:  404,
			jsonDefault: &types.ErrorRS{ErrorCode: 40410, Message: "Not found"},
			wantErr:     "HTTP 404",
		},
		{
			name:       "non-2xx without JSON",
			statusCode: 502,
			body:       []byte("<html>Bad Gateway</html>"),
			wantErr:    "unexpected HTTP 502",
		},
		{
			name:       "2xx with no body fallthrough",
			statusCode: 200,
			wantErr:    "API returned success status (200) but no valid response body was parsed",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resp := fakeV2Response{
				statusCode:  testCase.statusCode,
				JSONDefault: testCase.jsonDefault,
				V2JSON200:   testCase.v2Success,
				JSON200:     testCase.legacySuccess,
				Body:        testCase.body,
			}

			got, err := handleV2Response(resp, resp.JSONDefault, resp.V2JSON200, resp.JSON200, resp.Body)

			if testCase.wantErr != "" {
				require.Error(t, err)
				assert.Nil(t, got)
				assert.Contains(t, err.Error(), testCase.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.wantResult, got)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
