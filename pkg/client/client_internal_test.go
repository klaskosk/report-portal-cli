package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingTransport struct {
	err error
}

func (t *failingTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, t.err
}

func TestEmptyJSONBodyNormalizer_TransportError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("connection refused")
	normalizer := &emptyJSONBodyNormalizer{next: &failingTransport{err: wantErr}}

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://example.com/api/test",
		nil,
	)

	resp, err := normalizer.RoundTrip(req)
	require.Error(t, err)
	assert.Nil(t, resp)
	require.ErrorIs(t, err, wantErr)

	if resp != nil {
		_ = resp.Body.Close()
	}
}

func TestEmptyJSONBodyNormalizer_NonJSONResponse(t *testing.T) {
	t.Parallel()

	normalizer := &emptyJSONBodyNormalizer{
		next: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				ContentLength: 0,
				Header:        http.Header{"Content-Type": {"text/plain"}},
				Body:          http.NoBody,
			}, nil
		}),
	}

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://example.com/api/test",
		nil,
	)

	resp, err := normalizer.RoundTrip(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, int64(0), resp.ContentLength)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare host gets https and /api",
			input: "example.com",
			want:  "https://example.com/api",
		},
		{
			name:  "host with port gets https and /api",
			input: "localhost:8080",
			want:  "https://localhost:8080/api",
		},
		{
			name:  "explicit http scheme is preserved",
			input: "http://example.com",
			want:  "http://example.com/api",
		},
		{
			name:  "double-slash without scheme defaults to https",
			input: "//example.com",
			want:  "https://example.com/api",
		},
		{
			name:  "trailing slash is normalized",
			input: "https://example.com/",
			want:  "https://example.com/api",
		},
		{
			name:  "already ending in /api is unchanged",
			input: "https://example.com/api",
			want:  "https://example.com/api",
		},
		{
			name:  "/api/ normalizes trailing slash",
			input: "https://example.com/api/",
			want:  "https://example.com/api",
		},
		{
			name:  "custom path appends /api",
			input: "https://example.com/custom",
			want:  "https://example.com/custom/api",
		},
		{
			name:  "query and fragment are stripped",
			input: "https://example.com?foo=bar#frag",
			want:  "https://example.com/api",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeBaseURL(test.input)
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}
