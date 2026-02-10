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

	got, err := handleResponse(resp, "test op", resp.JSONDefault, resp.JSON200)
	require.NoError(t, err)
	assert.Equal(t, &want, got)
}

func TestHandleResponse_Error(t *testing.T) {
	t.Parallel()

	errBody := &types.ErrorRS{ErrorCode: 40410, Message: "Not found"}
	resp := fakeResponse{statusCode: 404, JSONDefault: errBody}

	got, err := handleResponse(resp, "test op", resp.JSONDefault, resp.JSON200)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, IsNotFound(err))

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.StatusCode())
	assert.Equal(t, "test op", apiErr.Operation())
	assert.Equal(t, 40410, apiErr.ErrorCode())
}

func TestHandleResponse_NilSuccess(t *testing.T) {
	t.Parallel()

	resp := fakeResponse{statusCode: 200, JSON200: nil}

	got, err := handleResponse(resp, "test op", resp.JSONDefault, resp.JSON200)
	require.NoError(t, err)
	assert.Nil(t, got)
}
