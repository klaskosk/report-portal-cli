package client_test

import (
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

			c, err := client.New("https://example.com", test.token)
			if test.expectError {
				require.Error(t, err)
				assert.Nil(t, c)
				assert.ErrorContains(t, err, "bearer token")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, c)
			}
		})
	}
}
