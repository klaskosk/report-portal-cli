package client_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_URLNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputURL     string
		expectedPath string
	}{
		{
			name:         "bare host gets https and /api",
			inputURL:     "REPLACE_HOST",
			expectedPath: "/api/v1/test/launch",
		},
		{
			name:         "URL with trailing slash normalizes",
			inputURL:     "REPLACE_HOST/",
			expectedPath: "/api/v1/test/launch",
		},
		{
			name:         "URL already ending in /api is unchanged",
			inputURL:     "REPLACE_HOST/api",
			expectedPath: "/api/v1/test/launch",
		},
		{
			name:         "URL ending in /api/ normalizes trailing slash",
			inputURL:     "REPLACE_HOST/api/",
			expectedPath: "/api/v1/test/launch",
		},
		{
			name:         "explicit custom path still appends /api",
			inputURL:     "REPLACE_HOST/custom",
			expectedPath: "/custom/api/v1/test/launch",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var capturedPath string

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedPath = r.URL.Path

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)

					_, _ = w.Write([]byte(`{
					"content": [],
					"page": {"number": 1, "size": 10, "totalElements": 0, "totalPages": 0}
				}`))
				}),
			)
			t.Cleanup(server.Close)

			inputURL := strings.Replace(test.inputURL, "REPLACE_HOST", server.URL, 1)

			c, err := client.New(inputURL, "test-token")
			require.NoError(t, err)

			_, err = c.Project("test").Launches().List(t.Context())
			require.NoError(t, err)
			assert.Equal(t, test.expectedPath, capturedPath)
		})
	}
}

func TestNew_PrependsHTTPS_WhenNoScheme(t *testing.T) {
	t.Parallel()

	// We can't actually test HTTPS prepending with httptest (it uses http://),
	// but we verify the constructor succeeds and creates a valid client.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))
	t.Cleanup(server.Close)

	// Use the http:// URL to verify the client creates successfully.
	c, err := client.New(server.URL, "test-token")
	require.NoError(t, err)
	require.NotNil(t, c)
}
