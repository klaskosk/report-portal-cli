package testutil

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// CommandNames returns the immediate child command names for cmd in sorted order.
func CommandNames(cmd *cobra.Command) []string {
	names := make([]string, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		names = append(names, child.Name())
	}

	return names
}

// ExecuteCommand runs a Cobra command with the given args and returns stdout, stderr, and error.
func ExecuteCommand(root *cobra.Command, args ...string) (string, string, error) {
	var outBuf, errBuf bytes.Buffer

	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(args)

	err := root.Execute()

	return outBuf.String(), errBuf.String(), err
}

// AttachCmdToRoot wires cmd into a test root, runs InitConfig, and returns cmd.
func AttachCmdToRoot(t *testing.T, cmd *cobra.Command) *cobra.Command {
	t.Helper()

	root := NewMinimalRootCmd()
	root.AddCommand(cmd)
	RunPersistentPreRunE(t, cmd)

	return cmd
}

// RunPersistentPreRunE applies the same preamble as the rp root command: logger, output format, InitConfig.
func RunPersistentPreRunE(t *testing.T, cmd *cobra.Command) {
	t.Helper()

	root := cmd.Root()
	ctx := core.WithLogger(context.Background(), slog.New(slog.DiscardHandler))
	cmd.SetContext(ctx)

	_ = root.PersistentFlags().Set("output", core.EffectiveOutputFormat(root))

	if core.IsGroupCommand(cmd) {
		return
	}

	require.NoError(t, core.InitConfig(cmd))
}

// NewTestCmdWithProjectFlag creates a command wired to root with -p and InitConfig applied.
func NewTestCmdWithProjectFlag(t *testing.T, root *cobra.Command) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "test"}
	core.RegisterProjectFlag(cmd)
	root.AddCommand(cmd)
	RunPersistentPreRunE(t, cmd)

	return cmd
}

// DiscardLoggerContext returns a context with a discard slog logger for direct handler tests.
func DiscardLoggerContext() context.Context {
	return core.WithLogger(context.Background(), slog.New(slog.DiscardHandler))
}

// SetCmdIO redirects cmd stdout/stderr to buffers (useful for direct RunE invocations).
func SetCmdIO(cmd *cobra.Command) (*bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	return &out, &errOut
}
