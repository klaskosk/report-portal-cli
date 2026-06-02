package user

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserExportCmd_outputFileFlag(t *testing.T) {
	t.Parallel()

	cmd := newUserExportCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--output-file", "users.csv"}))

	outputFile, err := cmd.Flags().GetString("output-file")
	require.NoError(t, err)

	assert.Equal(t, "users.csv", outputFile)
}

func TestResolveUserExportOptions_WithFilters(t *testing.T) {
	t.Parallel()

	cmd := newUserExportCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--role", "ADMINISTRATOR",
		"--account-type", "INTERNAL",
		"--email", "admin@example.com",
		"--active",
	}))

	options := resolveUserExportOptions(cmd)

	require.Len(t, options, 4)

	for _, option := range options {
		assert.Implements(t, (*client.ListUsersOption)(nil), option)
	}
}

func TestResolveUserExportOptions_NoFilters(t *testing.T) {
	t.Parallel()

	cmd := newUserExportCmd()
	require.NoError(t, cmd.ParseFlags(nil))

	options := resolveUserExportOptions(cmd)

	assert.Empty(t, options)
}
