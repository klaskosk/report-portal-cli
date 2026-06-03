//nolint:testpackage // exercises unexported execute helpers in package core.
package core

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface assertions for generated response types.
var (
	_ TypedResponse[types.GroupPage]     = api.GetGroupsResponse{}
	_ TypedResponse[types.StartLaunchRS] = api.StartLaunch1Response{}
	_ TypedResponse[[]string]            = api.GetAllLaunchNamesResponse{}
	_ VoidResponse                       = api.DeleteGroupResponse{}
)

// fakeV2Response mimics a response struct with both a v2 content-negotiated
// field and a legacy JSON200 field.
type fakeV2Response struct {
	statusCode  int
	JSONDefault *types.ErrorRS
	V2JSON200   *string
	JSON200     *string
	Body        []byte
}

func (f fakeV2Response) StatusCode() int { return f.statusCode }

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
			v2Success:  new("v2-ok"),
			wantResult: new("v2-ok"),
		},
		{
			name:          "legacy rejection with non-nil pointer",
			statusCode:    200,
			legacySuccess: new("legacy-ok"),
			wantErr:       "API returned legacy JSON shape",
		},
		{
			name:          "typed nil pointer is not legacy",
			statusCode:    200,
			legacySuccess: (*string)(nil),
			wantErr:       "API returned success status (200) but no valid response body was parsed",
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

			got, err := HandleV2Response(
				resp,
				resp.JSONDefault,
				resp.V2JSON200,
				resp.JSON200,
				resp.Body,
			)

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

// fakeTypedResponse implements TypedResponse[T] for testing execute.
type fakeTypedResponse[T any] struct {
	statusCode  int
	body        []byte
	jsonDefault *types.ErrorRS
	success     *T
}

func (f fakeTypedResponse[T]) StatusCode() int                { return f.statusCode }
func (f fakeTypedResponse[T]) GetBody() []byte                { return f.body }
func (f fakeTypedResponse[T]) GetJSONDefault() *types.ErrorRS { return f.jsonDefault }
func (f fakeTypedResponse[T]) GetSuccess() *T                 { return f.success }

// fakeVoidResp implements VoidResponse for testing ExecuteVoid.
type fakeVoidResp struct {
	statusCode  int
	body        []byte
	jsonDefault *types.ErrorRS
}

func (f fakeVoidResp) StatusCode() int                { return f.statusCode }
func (f fakeVoidResp) GetBody() []byte                { return f.body }
func (f fakeVoidResp) GetJSONDefault() *types.ErrorRS { return f.jsonDefault }

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestExecute_Success(t *testing.T) {
	t.Parallel()

	want := "ok"
	resp := fakeTypedResponse[string]{statusCode: 200, success: &want}

	got, err := Execute(context.Background(), discardLogger(), "test op", resp, nil)
	require.NoError(t, err)
	assert.Equal(t, &want, got)
}

func TestExecute_TransportError(t *testing.T) {
	t.Parallel()

	transportErr := errors.New("connection refused")
	resp := fakeTypedResponse[string]{statusCode: 0}

	got, err := Execute(context.Background(), discardLogger(), "test op", resp, transportErr)
	require.Error(t, err)
	assert.Nil(t, got)
	require.ErrorIs(t, err, transportErr)
	assert.Contains(t, err.Error(), "test op")
}

func TestExecute_APIError(t *testing.T) {
	t.Parallel()

	errBody := &types.ErrorRS{ErrorCode: 40410, Message: "Not found"}
	resp := fakeTypedResponse[string]{statusCode: 404, jsonDefault: errBody}

	got, err := Execute(context.Background(), discardLogger(), "test op", resp, nil)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, IsNotFound(err))
	assert.Contains(t, err.Error(), "test op")
}

func TestExecute_NonJSONError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       []byte
	}{
		{"404 without JSON body", 404, nil},
		{"500 without JSON body", 500, nil},
		{"502 with HTML body", 502, []byte("<html>Bad Gateway</html>")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resp := fakeTypedResponse[string]{statusCode: testCase.statusCode, body: testCase.body}

			got, err := Execute(context.Background(), discardLogger(), "test op", resp, nil)
			require.Error(t, err)
			assert.Nil(t, got)
			assert.True(t, HasStatusCode(err, testCase.statusCode))
			assert.Contains(t, err.Error(), "test op")

			var unexpectedErr *UnexpectedStatusError
			require.ErrorAs(t, err, &unexpectedErr)
			assert.Equal(t, testCase.statusCode, unexpectedErr.StatusCode())
		})
	}
}

func TestExecute_NonJSONErrorPreservesBody(t *testing.T) {
	t.Parallel()

	body := []byte("Full authentication is required to access this resource")
	resp := fakeTypedResponse[string]{statusCode: 401, body: body}

	_, err := Execute(context.Background(), discardLogger(), "test op", resp, nil)
	require.Error(t, err)

	gotBody, ok := BodyOf(err)
	require.True(t, ok)
	assert.Equal(t, body, gotBody)
}

func TestExecute_NilSuccess(t *testing.T) {
	t.Parallel()

	resp := fakeTypedResponse[string]{statusCode: 200}

	got, err := Execute(context.Background(), discardLogger(), "test op", resp, nil)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(
		t,
		err.Error(),
		"API returned success status (200) but no valid response body was parsed",
	)
	assert.Contains(t, err.Error(), "test op")
}

func TestExecute_ErrorWrapsOpName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		resp fakeTypedResponse[string]
		err  error
	}{
		{
			name: "transport error",
			resp: fakeTypedResponse[string]{},
			err:  errors.New("dial timeout"),
		},
		{
			name: "API error",
			resp: fakeTypedResponse[string]{
				statusCode:  403,
				jsonDefault: &types.ErrorRS{ErrorCode: 4030, Message: "Forbidden"},
			},
		},
		{
			name: "non-JSON error",
			resp: fakeTypedResponse[string]{statusCode: 502},
		},
		{
			name: "nil success",
			resp: fakeTypedResponse[string]{statusCode: 200},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := Execute(
				context.Background(),
				discardLogger(),
				"get launch",
				testCase.resp,
				testCase.err,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "get launch:")
		})
	}
}

func TestExecute_ErrorPredicates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		resp      fakeTypedResponse[string]
		predicate func(error) bool
	}{
		{
			name: "IsNotFound through wrapping",
			resp: fakeTypedResponse[string]{
				statusCode:  404,
				jsonDefault: &types.ErrorRS{ErrorCode: 40410, Message: "Not found"},
			},
			predicate: IsNotFound,
		},
		{
			name: "IsUnauthorized through wrapping",
			resp: fakeTypedResponse[string]{
				statusCode:  401,
				jsonDefault: &types.ErrorRS{ErrorCode: 4010, Message: "Unauthorized"},
			},
			predicate: IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := Execute(context.Background(), discardLogger(), "test op", testCase.resp, nil)
			require.Error(t, err)
			assert.True(t, testCase.predicate(err))
		})
	}
}

func TestExecuteVoid_Success(t *testing.T) {
	t.Parallel()

	resp := fakeVoidResp{statusCode: 204}

	err := ExecuteVoid(context.Background(), discardLogger(), "test op", resp, nil)
	assert.NoError(t, err)
}

func TestExecuteVoid_TransportError(t *testing.T) {
	t.Parallel()

	transportErr := errors.New("connection refused")
	resp := fakeVoidResp{statusCode: 0}

	err := ExecuteVoid(context.Background(), discardLogger(), "test op", resp, transportErr)
	require.Error(t, err)
	require.ErrorIs(t, err, transportErr)
	assert.Contains(t, err.Error(), "test op")
}

func TestExecuteVoid_APIError(t *testing.T) {
	t.Parallel()

	errBody := &types.ErrorRS{ErrorCode: 40410, Message: "Not found"}
	resp := fakeVoidResp{statusCode: 404, jsonDefault: errBody}

	err := ExecuteVoid(context.Background(), discardLogger(), "test op", resp, nil)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.Contains(t, err.Error(), "test op")
}

func TestExecuteVoid_NonJSONError(t *testing.T) {
	t.Parallel()

	resp := fakeVoidResp{statusCode: 502, body: []byte("<html>Bad Gateway</html>")}

	err := ExecuteVoid(context.Background(), discardLogger(), "test op", resp, nil)
	require.Error(t, err)
	assert.True(t, HasStatusCode(err, 502))
	assert.Contains(t, err.Error(), "test op")
}

func TestLegacyPayloadPresent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  bool
	}{
		{"nil any", nil, false},
		{"typed nil pointer", (*string)(nil), false},
		{"non-nil pointer", new("value"), true},
		{"string value (default kind)", "hello", true},
		{"int value (default kind)", 42, true},
		{"nil slice", ([]string)(nil), false},
		{"non-nil slice", []string{"a"}, true},
		{"nil map", (map[string]string)(nil), false},
		{"non-nil map", map[string]string{"k": "v"}, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.want, legacyPayloadPresent(test.input))
		})
	}
}
