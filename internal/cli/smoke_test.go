package cli_test

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli"
	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_HasNounGroups(t *testing.T) {
	t.Parallel()

	root := cli.NewRootCmdForTest()
	names := testutil.CommandNames(root)

	assert.Contains(t, names, "user")
	assert.Contains(t, names, "launch")
	assert.Contains(t, names, "filter")
	assert.Greater(t, len(names), 15)
}

func TestRootCmd_GroupCommandSkipsConfigWithBrokenFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cli.NewRootCmdForTest()
	require.NoError(t, root.PersistentFlags().Set("config", "/nonexistent/broken-config.yaml"))

	_, _, err := testutil.ExecuteCommand(root, "launch")
	require.NoError(t, err, "group parent should show help without reading config")
}

func TestRootCmd_NestedGroupCommandSkipsConfigWithBrokenFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cli.NewRootCmdForTest()
	require.NoError(t, root.PersistentFlags().Set("config", "/nonexistent/broken-config.yaml"))

	_, _, err := testutil.ExecuteCommand(root, "auth", "apikey")
	require.NoError(t, err, "nested group parent should show help without reading config")
}

func TestRootCmd_LeafCommandRequiresConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cli.NewRootCmdForTest()
	require.NoError(t, root.PersistentFlags().Set("config", "/nonexistent/broken-config.yaml"))

	_, _, err := testutil.ExecuteCommand(root, "launch", "list")
	require.Error(t, err)
}
