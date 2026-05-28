package cli //nolint:testpackage // white-box tests for demo-data CLI helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDemoDataCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newDemoDataCmd()

	assert.Equal(t, []string{"generate"}, commandNames(cmd))
}

func TestNewDemoDataCmd_ProjectFlagInherited(t *testing.T) {
	t.Parallel()

	cmd := newDemoDataCmd()

	flag := cmd.Commands()[0].InheritedFlags().Lookup("project")
	require.NotNil(t, flag, "generate subcommand should inherit --project")
	assert.Equal(t, "p", flag.Shorthand)
}

func TestNewDemoDataGenerateCmd_NoRequiredFlags(t *testing.T) {
	t.Parallel()

	cmd := newDemoDataGenerateCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()
	require.NoError(t, err)
}

func TestLoadDemoDataRequest_emptyWhenFromFileOmitted(t *testing.T) {
	t.Parallel()

	cmd := newDemoDataGenerateCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	request, err := loadDemoDataRequest(cmd)
	require.NoError(t, err)
	assert.Equal(t, types.DemoDataRq{}, request)
}

func TestLoadDemoDataRequest_validJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "demo_config.json")

	err := os.WriteFile(path, []byte(`{"createDashboard": true}`), 0o600)
	require.NoError(t, err)

	cmd := newDemoDataGenerateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	request, err := loadDemoDataRequest(cmd)
	require.NoError(t, err)
	require.NotNil(t, request.CreateDashboard)
	assert.True(t, *request.CreateDashboard)
}

func TestLoadDemoDataRequest_malformedJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	require.NoError(t, os.WriteFile(path, []byte(`{invalid`), 0o600))

	cmd := newDemoDataGenerateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	_, err := loadDemoDataRequest(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "demo data JSON")
}

func TestDemoDataGenerateTable_withResources(t *testing.T) {
	t.Parallel()

	dashboardID := int64(4)
	launchIDs := []int64{1, 2, 3}

	headers, rows := demoDataGenerateTable(&types.DemoDataRs{
		DashboardId: &dashboardID,
		LaunchIds:   &launchIDs,
	})

	assert.Equal(t, []string{"RESOURCE", "IDS"}, headers)
	assert.Equal(t, [][]string{
		{"Launches", "[1, 2, 3]"},
		{"Dashboard", "4"},
	}, rows)
}

func TestDemoDataGenerateTable_emptyResponse(t *testing.T) {
	t.Parallel()

	headers, rows := demoDataGenerateTable(&types.DemoDataRs{})

	assert.Equal(t, []string{"RESOURCE", "IDS"}, headers)
	assert.Equal(t, [][]string{{"STATUS", "Demo data generated successfully"}}, rows)
}
