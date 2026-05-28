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

	assert.Equal(
		t,
		[]string{"config", "create", "delete", "get", "list", "preconfigured", "update"},
		commandNames(cmd),
	)
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

func TestNewDashboardConfigCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newDashboardConfigCmd()

	assert.Equal(t, []string{"get"}, commandNames(cmd))
}

func TestNewDashboardPreconfiguredCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newDashboardPreconfiguredCmd()

	assert.Equal(t, []string{"create"}, commandNames(cmd))
}

func TestNewDashboardPreconfiguredCreateCmd_fromFileRequired(t *testing.T) {
	t.Parallel()

	cmd := newDashboardPreconfiguredCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "from-file")
}

func TestRunDashboardPreconfiguredCreate_requiresFromFileBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newDashboardPreconfiguredCreateCmd())

	err := runDashboardPreconfiguredCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestDashboardConfigRow_populated(t *testing.T) {
	t.Parallel()

	widgetName := "Launch statistics"
	widgetType := "launchStatistics"
	posX := int32(0)
	posY := int32(1)
	width := int32(6)
	height := int32(4)

	widget := &types.WidgetConfigResource{
		WidgetObject: types.WidgetObjectModel{
			WidgetId:   5,
			WidgetName: &widgetName,
			WidgetType: &widgetType,
			WidgetPosition: &types.Position{
				PositionX: &posX,
				PositionY: &posY,
			},
			WidgetSize: &types.Size{
				Width:  &width,
				Height: &height,
			},
		},
		WidgetResource: types.WidgetResource{
			Id:   5,
			Name: "Fallback name",
		},
	}

	headers := dashboardConfigHeaders()
	row := dashboardConfigRow(widget)

	assert.Equal(t, dashboardConfigHeaders(), headers)
	assert.Equal(t, []string{"5", "Launch statistics", "launchStatistics", "0,1", "6x4"}, row)
}

func TestDashboardConfigRow_usesResourceNameWhenWidgetNameMissing(t *testing.T) {
	t.Parallel()

	widget := &types.WidgetConfigResource{
		WidgetObject: types.WidgetObjectModel{
			WidgetId: 7,
		},
		WidgetResource: types.WidgetResource{
			Id:   7,
			Name: "From resource",
		},
	}

	row := dashboardConfigRow(widget)

	assert.Equal(t, "From resource", row[1])
}

func TestApplyDashboardPreconfiguredFlagOverrides(t *testing.T) {
	t.Parallel()

	cmd := newDashboardPreconfiguredCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--name", "Override", "--description", "desc"}))

	request := types.DashboardPreconfiguredRq{Name: "from-file"}
	applyDashboardPreconfiguredFlagOverrides(cmd, &request)

	assert.Equal(t, "Override", request.Name)
	require.NotNil(t, request.Description)
	assert.Equal(t, "desc", *request.Description)
}
