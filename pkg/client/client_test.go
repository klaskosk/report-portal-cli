package client_test

import (
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "returns error when bearer token is empty",
			token:       "",
			expectError: true,
		},
		{
			name:        "returns error when bearer token is only whitespace",
			token:       "   ",
			expectError: true,
		},
		{
			name:        "succeeds with valid token",
			token:       "valid-token",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			testClient, err := client.New("https://example.com", test.token)
			if test.expectError {
				require.Error(t, err)
				assert.Nil(t, testClient)
				assert.ErrorContains(t, err, "bearer token")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, testClient)
			}
		})
	}
}

func TestNew_OptionError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("bad option")
	c, err := client.New("https://example.com", "valid-token", client.FailingOption(wantErr))
	require.Error(t, err)
	assert.Nil(t, c)
	assert.ErrorIs(t, err, wantErr)
}

func TestNew_WithHTTPClient(t *testing.T) {
	t.Parallel()

	custom := &http.Client{Transport: http.DefaultTransport}

	c, err := client.New("https://example.com", "valid-token", client.WithHTTPClient(custom))
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestNew_WithLogger(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)

	c, err := client.New("https://example.com", "valid-token", client.WithLogger(logger))
	require.NoError(t, err)
	assert.NotNil(t, c)
}
