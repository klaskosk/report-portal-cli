package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
