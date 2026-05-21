package cli //nolint:testpackage // white-box tests for dashboard CLI helpers

import (
	"strings"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDashboardCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newDashboardCmd()

	assert.Equal(t, []string{"create", "delete", "get", "list", "update"}, commandNames(cmd))
}

func TestNewDashboardCmd_ProjectFlagInheritedBySubcommands(t *testing.T) {
	t.Parallel()

	cmd := newDashboardCmd()

	for _, sub := range cmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			t.Parallel()

			flag := sub.InheritedFlags().Lookup("project")
			require.NotNil(
				t,
				flag,
				"subcommand %q should inherit --project persistent flag",
				sub.Name(),
			)
			assert.Equal(t, "p", flag.Shorthand)
		})
	}
}

func TestNewDashboardListCmd_NoDefaultSort(t *testing.T) {
	t.Parallel()

	cmd := newDashboardListCmd()

	flags, err := resolveListFlags(cmd)
	require.NoError(t, err)

	assert.Empty(t, flags.sort)
}

func TestParseDashboardID_valid(t *testing.T) {
	t.Parallel()

	id, err := parseDashboardID("42")

	require.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestParseDashboardID_invalid(t *testing.T) {
	t.Parallel()

	_, err := parseDashboardID("abc")

	require.Error(t, err)
	assert.EqualError(t, err, `invalid dashboard ID "abc": must be a numeric ID`)
}

func TestDashboardRow_populated(t *testing.T) {
	t.Parallel()

	desc := "Shows overall test results"
	owner := "admin"
	resource := &types.DashboardResource{
		Id:          1,
		Name:        "Test Overview",
		Owner:       &owner,
		Description: &desc,
	}

	headers, row := dashboardRow(resource)

	assert.Equal(t, dashboardHeaders(), headers)
	assert.Equal(t, []string{"1", "Test Overview", "admin", "Shows overall test results"}, row)
}

func TestDashboardRow_nilOwner(t *testing.T) {
	t.Parallel()

	resource := &types.DashboardResource{
		Id:   2,
		Name: "Dashboard",
	}

	_, row := dashboardRow(resource)

	assert.Empty(t, row[2], "OWNER should be empty when nil")
}

func TestDashboardRow_nilDescription(t *testing.T) {
	t.Parallel()

	resource := &types.DashboardResource{
		Id:   3,
		Name: "Dashboard",
	}

	_, row := dashboardRow(resource)

	assert.Empty(t, row[3], "DESCRIPTION should be empty when nil")
}

func TestDashboardRow_descriptionTruncatedAt60(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("a", 70)
	resource := &types.DashboardResource{
		Id:          4,
		Name:        "n",
		Description: &long,
	}

	_, row := dashboardRow(resource)

	expected := strings.Repeat("a", 60) + "..."
	assert.Equal(t, expected, row[3])
}

func TestDashboardRow_descriptionExactly60(t *testing.T) {
	t.Parallel()

	exact := strings.Repeat("a", 60)
	resource := &types.DashboardResource{
		Id:          5,
		Name:        "n",
		Description: &exact,
	}

	_, row := dashboardRow(resource)

	assert.Equal(t, exact, row[3], "should not truncate at exactly 60 characters")
}

func TestNewDashboardCreateCmd_nameRequired(t *testing.T) {
	t.Parallel()

	cmd := newDashboardCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestRunDashboardCreate_emptyNameError(t *testing.T) {
	t.Parallel()

	cmd := newDashboardCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--name", ""}))

	err := runDashboardCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name must not be empty")
}

func TestNewDashboardCreateCmd_nameAndDescriptionFlags(t *testing.T) {
	t.Parallel()

	cmd := newDashboardCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--name", "My Dashboard", "--description", "desc"}))

	assert.True(t, cmd.Flags().Changed("name"))
	assert.True(t, cmd.Flags().Changed("description"))
	assert.Equal(t, "My Dashboard", mustGetStringFlag(cmd, "name"))
	assert.Equal(t, "desc", mustGetStringFlag(cmd, "description"))
}

func TestResolveDashboardUpdateName_emptyNameError(t *testing.T) {
	t.Parallel()

	cmd := newDashboardUpdateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--name", ""}))

	_, err := resolveDashboardUpdateName(cmd, nil, 42)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name must not be empty")
}

func TestRunDashboardUpdate_noFlagsError(t *testing.T) {
	t.Parallel()

	cmd := newDashboardUpdateCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := runDashboardUpdate(cmd, []string{"42"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one flag")
}

func TestRunDashboardUpdate_nameFlagRecognized(t *testing.T) {
	t.Parallel()

	cmd := newDashboardUpdateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--name", "New Name"}))

	assert.True(t, cmd.Flags().Changed("name"))
	assert.False(t, cmd.Flags().Changed("description"))
	assert.Equal(t, "New Name", mustGetStringFlag(cmd, "name"))
}

func TestRunDashboardUpdate_descriptionFlagRecognized(t *testing.T) {
	t.Parallel()

	cmd := newDashboardUpdateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--description", "New desc"}))

	assert.False(t, cmd.Flags().Changed("name"))
	assert.True(t, cmd.Flags().Changed("description"))
	assert.Equal(t, "New desc", mustGetStringFlag(cmd, "description"))
}
