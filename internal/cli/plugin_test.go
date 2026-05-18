package cli //nolint:testpackage // white-box tests for plugin CLI helpers

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPluginCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newPluginCmd()

	assert.Equal(t, []string{"delete", "list", "update"}, commandNames(cmd))
}

func TestNewPluginCmd_NoProjectFlag(t *testing.T) {
	t.Parallel()

	cmd := newPluginCmd()

	assert.Nil(
		t,
		cmd.PersistentFlags().Lookup("project"),
		"plugin command must not have a --project flag",
	)
	assert.Nil(t, cmd.Flags().Lookup("project"), "plugin command must not have a --project flag")

	for _, sub := range cmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			t.Parallel()

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

	cmd := attachCmdToRoot(t, newPluginUpdateCmd())

	err := cmd.ParseFlags([]string{"1"})
	require.NoError(t, err)

	err = cmd.ValidateRequiredFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enabled")
}

func TestPluginUpdateCmd_InvalidID(t *testing.T) {
	t.Parallel()

	cmd := attachCmdToRoot(t, newPluginUpdateCmd())

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

	cmd := attachCmdToRoot(t, newPluginDeleteCmd())

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
