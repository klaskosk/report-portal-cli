package core

import (
	"context"
	"log/slog"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func runTestPreamble(t *testing.T, cmd *cobra.Command) {
	t.Helper()

	root := cmd.Root()
	cmd.SetContext(WithLogger(context.Background(), slog.New(slog.DiscardHandler)))

	_ = root.PersistentFlags().Set("output", EffectiveOutputFormat(root))

	if IsGroupCommand(cmd) {
		return
	}

	require.NoError(t, InitConfig(cmd))
}

func newTestCmdWithProjectFlag(t *testing.T) *cobra.Command {
	t.Helper()

	root := NewMinimalRootCmd()
	cmd := &cobra.Command{Use: "test"}
	RegisterProjectFlag(cmd)
	root.AddCommand(cmd)
	runTestPreamble(t, cmd)

	return cmd
}
