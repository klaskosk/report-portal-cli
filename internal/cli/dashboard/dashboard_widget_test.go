package dashboard //nolint:testpackage // white-box tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDashboardCmd_includesWidgetSubcommand(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	assert.Contains(t, testutil.CommandNames(cmd), "widget")
}

func TestNewDashboardWidgetCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newDashboardWidgetCmd()

	assert.Equal(t, []string{"add", "remove"}, testutil.CommandNames(cmd))
}

func TestNewDashboardWidgetAddCmd_dashboardIDRequired(t *testing.T) {
	t.Parallel()

	cmd := newDashboardWidgetAddCmd()

	err := cmd.ValidateArgs([]string{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg")
}

func TestResolveAddWidgetRequest_noInputProvided(t *testing.T) {
	t.Parallel()

	cmd := newDashboardWidgetAddCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	printer := core.NewPrinterFromCmd(cmd)

	_, err := resolveAddWidgetRequest(cmd, printer)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no input provided")
}

func TestResolveAddWidgetRequest_fromFlags(t *testing.T) {
	t.Parallel()

	cmd := newDashboardWidgetAddCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--widget-id", "10",
		"--widget-name", "Pass Rate",
		"--widget-type", "statisticTrend",
		"--width", "6",
		"--height", "4",
		"--position-x", "0",
		"--position-y", "1",
	}))

	printer := core.NewPrinterFromCmd(cmd)

	request, err := resolveAddWidgetRequest(cmd, printer)
	require.NoError(t, err)

	assert.Equal(t, int64(10), request.AddWidget.WidgetId)
	require.NotNil(t, request.AddWidget.WidgetName)
	assert.Equal(t, "Pass Rate", *request.AddWidget.WidgetName)
	require.NotNil(t, request.AddWidget.WidgetType)
	assert.Equal(t, "statisticTrend", *request.AddWidget.WidgetType)
	require.NotNil(t, request.AddWidget.WidgetSize)
	require.NotNil(t, request.AddWidget.WidgetSize.Width)
	assert.Equal(t, int32(6), *request.AddWidget.WidgetSize.Width)
	require.NotNil(t, request.AddWidget.WidgetSize.Height)
	assert.Equal(t, int32(4), *request.AddWidget.WidgetSize.Height)
	require.NotNil(t, request.AddWidget.WidgetPosition)
	require.NotNil(t, request.AddWidget.WidgetPosition.PositionX)
	assert.Equal(t, int32(0), *request.AddWidget.WidgetPosition.PositionX)
	require.NotNil(t, request.AddWidget.WidgetPosition.PositionY)
	assert.Equal(t, int32(1), *request.AddWidget.WidgetPosition.PositionY)
}

func TestResolveAddWidgetRequest_missingWidgetName(t *testing.T) {
	t.Parallel()

	cmd := newDashboardWidgetAddCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--widget-id", "10",
		"--widget-type", "statisticTrend",
	}))

	printer := core.NewPrinterFromCmd(cmd)

	_, err := resolveAddWidgetRequest(cmd, printer)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--widget-name")
}

func TestResolveAddWidgetRequest_missingWidgetType(t *testing.T) {
	t.Parallel()

	cmd := newDashboardWidgetAddCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--widget-id", "10",
		"--widget-name", "Pass Rate",
	}))

	printer := core.NewPrinterFromCmd(cmd)

	_, err := resolveAddWidgetRequest(cmd, printer)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--widget-type")
}

func TestResolveAddWidgetRequest_fromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "widget.json")

	require.NoError(t, os.WriteFile(path, []byte(`{
		"addWidget": {
			"widgetId": 10,
			"widgetName": "From File",
			"widgetType": "launchStatistics"
		}
	}`), 0o600))

	cmd := newDashboardWidgetAddCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	printer := core.NewPrinterFromCmd(cmd)

	request, err := resolveAddWidgetRequest(cmd, printer)
	require.NoError(t, err)

	assert.Equal(t, int64(10), request.AddWidget.WidgetId)
	require.NotNil(t, request.AddWidget.WidgetName)
	assert.Equal(t, "From File", *request.AddWidget.WidgetName)
	require.NotNil(t, request.AddWidget.WidgetType)
	assert.Equal(t, "launchStatistics", *request.AddWidget.WidgetType)
}

func TestResolveAddWidgetRequest_fromFileTakesPrecedenceOverFlags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "widget.json")

	require.NoError(t, os.WriteFile(path, []byte(`{
		"addWidget": {
			"widgetId": 99,
			"widgetName": "FROM_FILE",
			"widgetType": "FROM_FILE_TYPE"
		}
	}`), 0o600))

	cmd := newDashboardWidgetAddCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--widget-id", "10",
		"--widget-name", "FROM_FLAG",
		"--widget-type", "FROM_FLAG_TYPE",
		"--from-file", path,
	}))

	printer := core.NewPrinterFromCmd(cmd)

	request, err := resolveAddWidgetRequest(cmd, printer)
	require.NoError(t, err)

	assert.Equal(t, int64(99), request.AddWidget.WidgetId)
	require.NotNil(t, request.AddWidget.WidgetName)
	assert.Equal(t, "FROM_FILE", *request.AddWidget.WidgetName)
	require.NotNil(t, request.AddWidget.WidgetType)
	assert.Equal(t, "FROM_FILE_TYPE", *request.AddWidget.WidgetType)
}

func TestNewDashboardWidgetRemoveCmd_exactlyTwoArgs(t *testing.T) {
	t.Parallel()

	cmd := newDashboardWidgetRemoveCmd()

	err := cmd.ValidateArgs([]string{"5"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg")

	err = cmd.ValidateArgs([]string{"5", "10", "11"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg")

	require.NoError(t, cmd.ValidateArgs([]string{"5", "10"}))
}

func TestRunDashboardWidgetRemove_invalidDashboardID(t *testing.T) {
	t.Parallel()

	cmd := testutil.AttachCmdToRoot(t, newDashboardWidgetRemoveCmd())

	err := runDashboardWidgetRemove(cmd, []string{"abc", "10"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dashboard ID")
}

func TestRunDashboardWidgetRemove_invalidWidgetID(t *testing.T) {
	t.Parallel()

	cmd := testutil.AttachCmdToRoot(t, newDashboardWidgetRemoveCmd())

	err := runDashboardWidgetRemove(cmd, []string{"5", "abc"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid widget ID")
}

func TestNewDashboardWidgetCmd_inheritsProjectFlag(t *testing.T) {
	t.Parallel()

	parent := NewCommand()
	widgetCmd, _, err := parent.Find([]string{"widget"})
	require.NoError(t, err)

	for _, sub := range widgetCmd.Commands() {
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

func TestNewDashboardWidgetAddCmd_invalidDashboardID(t *testing.T) {
	t.Parallel()

	cmd := testutil.AttachCmdToRoot(t, newDashboardWidgetAddCmd())
	require.NoError(t, cmd.ParseFlags([]string{
		"--widget-id", "10",
		"--widget-name", "Pass Rate",
		"--widget-type", "statisticTrend",
	}))

	err := runDashboardWidgetAdd(cmd, []string{"abc"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dashboard ID")
}

func TestResolveAddWidgetRequest_noSizeOrPositionWhenOmitted(t *testing.T) {
	t.Parallel()

	cmd := newDashboardWidgetAddCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--widget-id", "10",
		"--widget-name", "Pass Rate",
		"--widget-type", "statisticTrend",
	}))

	printer := core.NewPrinterFromCmd(cmd)

	request, err := resolveAddWidgetRequest(cmd, printer)
	require.NoError(t, err)

	assert.Nil(t, request.AddWidget.WidgetSize)
	assert.Nil(t, request.AddWidget.WidgetPosition)
}
