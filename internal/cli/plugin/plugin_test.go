package plugin //nolint:testpackage // white-box tests

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPluginCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	assert.Equal(
		t,
		[]string{"create", "delete", "exec", "list", "update"},
		testutil.CommandNames(cmd),
	)
}

func TestNewPluginCmd_NoProjectFlag(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	assert.Nil(
		t,
		cmd.PersistentFlags().Lookup("project"),
		"plugin command must not have a persistent --project flag",
	)
	assert.Nil(t, cmd.Flags().Lookup("project"), "plugin command must not have a --project flag")

	for _, sub := range cmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			t.Parallel()

			if sub.Name() == "exec" {
				flag := sub.Flags().Lookup("project")
				require.NotNil(t, flag, "exec subcommand must have a local --project flag")
				assert.Equal(t, "p", flag.Shorthand)

				return
			}

			assert.Nil(
				t,
				sub.Flags().Lookup("project"),
				"subcommand %q must not have a --project flag",
				sub.Name(),
			)
		})
	}
}

func TestPluginUpdateCmd_EnabledFlagRequired(t *testing.T) {
	t.Parallel()

	cmd := testutil.AttachCmdToRoot(t, newPluginUpdateCmd())

	err := cmd.ParseFlags([]string{"1"})
	require.NoError(t, err)

	err = cmd.ValidateRequiredFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enabled")
}

func TestPluginUpdateCmd_InvalidID(t *testing.T) {
	t.Parallel()

	cmd := testutil.AttachCmdToRoot(t, newPluginUpdateCmd())

	err := runPluginUpdate(cmd, []string{"abc"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid plugin ID "abc"`)
}

func TestPluginUpdateCmd_EnabledTrue(t *testing.T) {
	t.Parallel()

	cmd := newPluginUpdateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--enabled=true"}))

	enabled, err := cmd.Flags().GetBool("enabled")
	require.NoError(t, err)

	request := types.UpdatePluginStateRQ{Enabled: &enabled}

	require.NotNil(t, request.Enabled)
	assert.True(t, *request.Enabled)
}

func TestPluginUpdateCmd_EnabledFalse(t *testing.T) {
	t.Parallel()

	cmd := newPluginUpdateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--enabled=false"}))

	enabled, err := cmd.Flags().GetBool("enabled")
	require.NoError(t, err)

	request := types.UpdatePluginStateRQ{Enabled: &enabled}

	require.NotNil(t, request.Enabled)
	assert.False(t, *request.Enabled)
}

func TestPluginDeleteCmd_InvalidID(t *testing.T) {
	t.Parallel()

	cmd := testutil.AttachCmdToRoot(t, newPluginDeleteCmd())

	err := runPluginDelete(cmd, []string{"not-a-number"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid plugin ID "not-a-number"`)
}

func TestPluginRow_populated(t *testing.T) {
	t.Parallel()

	pluginID := int64(1)
	name := "JIRA"
	groupType := "bts"
	enabled := true
	pluginType := "jira"

	plugin := &types.IntegrationTypeResource{
		Type:       &pluginID,
		Name:       &name,
		GroupType:  &groupType,
		Enabled:    &enabled,
		PluginType: &pluginType,
	}

	headers, row := pluginRow(plugin)

	assert.Equal(t, pluginHeaders(), headers)
	assert.Equal(t, []string{"1", "JIRA", "bts", "true", "jira"}, row)
}

func TestPluginRow_allNilPointers(t *testing.T) {
	t.Parallel()

	plugin := &types.IntegrationTypeResource{}

	headers, row := pluginRow(plugin)

	assert.Equal(t, pluginHeaders(), headers)
	assert.Equal(t, []string{"", "", "", "", ""}, row)
}

func TestPluginHeaders(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"ID", "NAME", "GROUP", "ENABLED", "PLUGIN TYPE"}, pluginHeaders())
}

func TestNewPluginExecCmd_projectRequired(t *testing.T) {
	t.Parallel()

	cmd := testutil.AttachCmdToRoot(t, newPluginExecCmd())
	require.NoError(t, cmd.ParseFlags(nil))

	err := cmd.ValidateRequiredFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project")
}

func TestLoadPluginExecBody_emptyWhenFromFileOmitted(t *testing.T) {
	t.Parallel()

	cmd := newPluginExecCmd()
	require.NoError(t, cmd.ParseFlags([]string{"-p", "my_project"}))

	body, err := loadPluginExecBody(cmd)
	require.NoError(t, err)
	assert.Empty(t, body)
}

func TestNewPluginCreateCmd_fileRequired(t *testing.T) {
	t.Parallel()

	cmd := newPluginCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file")
}

func TestPluginExecTable_emptyResult(t *testing.T) {
	t.Parallel()

	headers, rows := pluginExecTable(map[string]any{})

	assert.Equal(t, []string{"STATUS"}, headers)
	assert.Equal(t, [][]string{{"Command executed successfully"}}, rows)
}

func TestPluginExecTable_withKeys(t *testing.T) {
	t.Parallel()

	headers, rows := pluginExecTable(map[string]any{"count": 3, "ok": true})

	assert.Equal(t, []string{"KEY", "VALUE"}, headers)
	assert.Equal(t, [][]string{{"count", "3"}, {"ok", "true"}}, rows)
}

func TestRunPluginCreate_missingFileError(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newPluginCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--file", "/nonexistent/plugin.jar"}))

	err := runPluginCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading plugin file")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}
