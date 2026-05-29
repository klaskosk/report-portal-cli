package setting //nolint:testpackage // white-box tests

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettingCmd_CommandTree(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	assert.Equal(
		t,
		[]string{"issue-subtype", "notification", "pattern", "server"},
		testutil.CommandNames(cmd),
	)

	subTrees := map[string][]string{
		"server":        {"get", "update"},
		"notification":  {"create", "delete", "list", "update"},
		"pattern":       {"create", "delete", "list", "update"},
		"issue-subtype": {"create", "delete", "list", "update"},
	}

	for name, want := range subTrees {
		var sub *cobra.Command

		for _, candidate := range cmd.Commands() {
			if candidate.Name() == name {
				sub = candidate

				break
			}
		}

		require.NotNil(t, sub, "subcommand %q must exist", name)
		assert.Equal(t, want, testutil.CommandNames(sub), "subcommand %q children", name)
	}
}

func TestNewSettingCmd_ProjectFlagInheritedByProjectScopedSubcommands(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()
	projectScoped := []string{"notification", "pattern", "issue-subtype"}

	for _, name := range projectScoped {
		var sub *cobra.Command

		for _, candidate := range cmd.Commands() {
			if candidate.Name() == name {
				sub = candidate

				break
			}
		}

		require.NotNil(t, sub)

		for _, leaf := range sub.Commands() {
			t.Run(name+"/"+leaf.Name(), func(t *testing.T) {
				t.Parallel()

				flag := leaf.InheritedFlags().Lookup("project")
				require.NotNil(t, flag, "subcommand should inherit --project")
				assert.Equal(t, "p", flag.Shorthand)
			})
		}
	}
}

func TestRunSettingServerUpdate_requiresKeyAndValue(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newSettingServerUpdateCmd())

	err := runSettingServerUpdate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--key is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunSettingServerUpdate_requiresValue(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newSettingServerUpdateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--key", "analytics.enabled"}))

	err := runSettingServerUpdate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--value is required")
}

func TestResolveServerSettingUpdateRequest_fromFlags(t *testing.T) {
	t.Parallel()

	cmd := newSettingServerUpdateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--key", "foo", "--value", "bar"}))

	request, err := resolveServerSettingUpdateRequest(cmd, core.NewPrinterFromCmd(cmd))
	require.NoError(t, err)
	assert.Equal(t, "foo", request.Key)
	assert.Equal(t, "bar", request.Value)
}

func TestServerSettingRows_sortedKeys(t *testing.T) {
	t.Parallel()

	rows := serverSettingRows(map[string]string{
		"zebra": "z",
		"alpha": "a",
	})

	require.Len(t, rows, 2)
	assert.Equal(t, []string{"alpha", "a"}, rows[0])
	assert.Equal(t, []string{"zebra", "z"}, rows[1])
}

func TestServerSettingHeaders(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"KEY", "VALUE"}, serverSettingHeaders())
}
