package cli //nolint:testpackage // white-box tests for unexported config helpers

import (
	"context"
	"log/slog"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCmdWithProjectFlag(t *testing.T) *cobra.Command {
	t.Helper()

	root := newRootCmd()
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("project", "p", "", "project name")
	root.AddCommand(cmd)

	// Set up a context with a discard logger to satisfy loggerFromCmd.
	ctx := context.WithValue(context.Background(), loggerKey{}, slog.New(slog.DiscardHandler))
	cmd.SetContext(ctx)

	return cmd
}

func TestResolveProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		connection  *Connection
		wantProject string
	}{
		{
			name:        "uses flag value",
			args:        []string{"-p", "my-project"},
			connection:  &Connection{},
			wantProject: "my-project",
		},
		{
			name:        "falls back to connection default",
			connection:  &Connection{Project: "default-project"},
			wantProject: "default-project",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cmd := newTestCmdWithProjectFlag(t)

			if len(test.args) > 0 {
				err := cmd.ParseFlags(test.args)
				require.NoError(t, err)
			}

			project, err := resolveProject(cmd, test.connection)
			require.NoError(t, err)
			assert.Equal(t, test.wantProject, project)
		})
	}
}

func TestResolveProject_EmptyFlag(t *testing.T) {
	t.Parallel()

	cmd := newTestCmdWithProjectFlag(t)

	err := cmd.ParseFlags([]string{"-p", ""})
	require.NoError(t, err)

	_, err = resolveProject(cmd, &Connection{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project name cannot be empty")
}

func TestResolveProject_NoProjectSpecified(t *testing.T) {
	t.Parallel()

	cmd := newTestCmdWithProjectFlag(t)

	_, err := resolveProject(cmd, &Connection{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project specified")
}
