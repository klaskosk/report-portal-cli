package cli //nolint:testpackage // white-box tests for unexported launch helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAttribute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantValue string
		wantFound bool
	}{
		{"valid key:value", "env:prod", "env", "prod", true},
		{"empty value", "env:", "env", "", true},
		{
			"value with colons",
			"url:http://example.com:8080",
			"url",
			"http://example.com:8080",
			true,
		},
		{"empty key", ":value", "", "", false},
		{"missing colon", "no-colon", "", "", false},
		{"empty string", "", "", "", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			key, value, found := parseAttribute(test.input)
			assert.Equal(t, test.wantKey, key)
			assert.Equal(t, test.wantValue, value)
			assert.Equal(t, test.wantFound, found)
		})
	}
}

func TestApplyLaunchFilterOptions_MalformedAttribute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"missing colon", "missing-colon"},
		{"empty key", ":value"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			flags := launchFilterFlags{
				attributes: []string{test.input},
			}

			_, err := applyLaunchFilterOptions(flags)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid attribute format")
			assert.Contains(t, err.Error(), test.input)
		})
	}
}

func TestApplyLaunchFilterOptions_ValidAttributes(t *testing.T) {
	t.Parallel()

	flags := launchFilterFlags{
		attributes: []string{"env:prod", "team:backend"},
	}

	options, err := applyLaunchFilterOptions(flags)
	require.NoError(t, err)
	assert.Len(t, options, 2)
}
