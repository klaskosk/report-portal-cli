package cli //nolint:testpackage // white-box tests for context CLI helpers

import (
	"context"
	"log/slog"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCmd_failsWithoutConnection(t *testing.T) {
	globalViperTestMu.Lock()
	t.Cleanup(globalViperTestMu.Unlock)

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	viper.Reset()

	cmd := newTestCmdForContext(t)

	result, err := resolveCmd(cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no Report Portal URL configured")
}

func TestResolveCmdProject_failsWithoutProject(t *testing.T) {
	t.Parallel()

	deps := &cmdDeps{
		printer:    newPrinter(""),
		connection: &Connection{URL: "http://example.com", Token: "tok"},
	}

	cmd := newTestCmdForContext(t)

	_, err := deps.resolveProject(cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project specified")
}

func newTestCmdForContext(t *testing.T) *cobra.Command {
	t.Helper()

	root := newRootCmd()
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("project", "p", "", "project name")
	root.AddCommand(cmd)

	ctx := context.WithValue(context.Background(), loggerKey{}, slog.New(slog.DiscardHandler))
	cmd.SetContext(ctx)

	return cmd
}

// attachCmdToRoot wires an existing cmd into a newRootCmd tree and sets a discarded-logger
// context on it. Use this in tests that call run functions directly and need cmd.Root() to
// resolve the --output flag.
func attachCmdToRoot(t *testing.T, cmd *cobra.Command) *cobra.Command {
	t.Helper()

	root := newRootCmd()
	root.AddCommand(cmd)

	ctx := context.WithValue(context.Background(), loggerKey{}, slog.New(slog.DiscardHandler))
	cmd.SetContext(ctx)

	return cmd
}
