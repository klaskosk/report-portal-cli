package core

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCmd_failsWithoutConnection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	root := NewMinimalRootCmd()
	cmd := &cobra.Command{Use: "test"}
	root.AddCommand(cmd)
	runTestPreamble(t, cmd)

	result, err := ResolveCmd(cmd)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no Report Portal URL configured")
}

func TestResolveProject_viaDeps_failsWithoutProject(t *testing.T) {
	t.Parallel()

	deps := &Deps{
		Printer:    NewPrinter(""),
		Connection: &Connection{URL: "http://example.com", Token: "tok"},
	}

	root := NewMinimalRootCmd()
	cmd := &cobra.Command{Use: "test"}
	RegisterProjectFlag(cmd)
	root.AddCommand(cmd)
	cmd.SetContext(WithLogger(t.Context(), NewLogger(0)))

	_, err := deps.ResolveProject(cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project specified")
}
