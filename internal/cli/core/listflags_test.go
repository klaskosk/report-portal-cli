package core

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCmdWithListFlags creates a test Cobra command with standard list flags registered.
func newTestCmdWithListFlags(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "test"}
	RegisterListFlags(cmd)

	return cmd
}

func TestResolveListFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		wantPage int32
		wantSize int32
		wantSort []string
	}{
		{
			name:     "defaults",
			wantPage: 0,
			wantSize: 10,
			wantSort: []string{"startTime,desc"},
		},
		{
			name:     "custom values",
			args:     []string{"--page-size", "25", "--page", "3", "--sort", "name,asc"},
			wantPage: 2,
			wantSize: 25,
			wantSort: []string{"name,asc"},
		},
		{
			name:     "explicit page 1 converts to 0",
			args:     []string{"--page", "1"},
			wantPage: 0,
			wantSize: 10,
			wantSort: []string{"startTime,desc"},
		},
		{
			name:     "multiple sort criteria",
			args:     []string{"--sort", "startTime,desc", "--sort", "name,asc"},
			wantPage: 0,
			wantSize: 10,
			wantSort: []string{"startTime,desc", "name,asc"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cmd := newTestCmdWithListFlags(t)

			if len(test.args) > 0 {
				err := cmd.ParseFlags(test.args)
				require.NoError(t, err)
			}

			flags, err := ResolveListFlags(cmd)
			require.NoError(t, err)

			assert.Equal(t, test.wantSize, flags.PageSize)
			assert.Equal(t, test.wantPage, flags.Page)
			assert.Equal(t, test.wantSort, flags.Sort)
		})
	}
}

func TestResolveListFlags_invalidPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "page 0 is invalid",
			args: []string{"--page", "0"},
		},
		{
			name: "negative page is invalid",
			args: []string{"--page", "-1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cmd := newTestCmdWithListFlags(t)

			err := cmd.ParseFlags(test.args)
			require.NoError(t, err)

			_, err = ResolveListFlags(cmd)
			require.ErrorContains(t, err, "page must be at least 1")
		})
	}
}
