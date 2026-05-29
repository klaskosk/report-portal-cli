package cli //nolint:testpackage // white-box tests for launch analysis CLI helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLaunchCmd_AnalysisSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newLaunchCmd()

	assert.Contains(t, commandNames(cmd), "analyze")
	assert.Contains(t, commandNames(cmd), "compare")
	assert.Contains(t, commandNames(cmd), "merge")
}

func TestNewLaunchAnalyzeCmd_ExactlyOneArg(t *testing.T) {
	t.Parallel()

	cmd := newLaunchAnalyzeCmd()

	require.Error(t, cmd.Args(cmd, []string{}))
	require.NoError(t, cmd.Args(cmd, []string{"42"}))
	require.Error(t, cmd.Args(cmd, []string{"42", "extra"}))
}

func TestBuildAnalyzeLaunchRequest_AnalyzerModeRequired(t *testing.T) {
	t.Parallel()

	cmd := newLaunchAnalyzeCmd()

	_, err := buildAnalyzeLaunchRequest(cmd, 42)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--analyzer-mode is required")
}

func TestBuildAnalyzeLaunchRequest_ValidAnalyzerMode(t *testing.T) {
	t.Parallel()

	cmd := newLaunchAnalyzeCmd()
	require.NoError(t, cmd.Flags().Set("analyzer-mode", "ALL"))

	req, err := buildAnalyzeLaunchRequest(cmd, 42)

	require.NoError(t, err)
	assert.Equal(t, int64(42), req.LaunchId)
	assert.Equal(t, types.AnalyzeLaunchRQAnalyzerMode("ALL"), req.AnalyzerMode)
	require.Len(t, req.AnalyzeItemsMode, 1)
	assert.Equal(t, types.TOINVESTIGATE, req.AnalyzeItemsMode[0])
	assert.Equal(t, types.AutoAnalyzer, req.AnalyzerTypeName)
}

func TestBuildAnalyzeLaunchRequest_InvalidAnalyzerMode(t *testing.T) {
	t.Parallel()

	cmd := newLaunchAnalyzeCmd()
	require.NoError(t, cmd.Flags().Set("analyzer-mode", "INVALID"))

	_, err := buildAnalyzeLaunchRequest(cmd, 42)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid analyzer-mode")
}

func TestResolveAnalyzeLaunchAnalyzerMode_AcceptsAllModes(t *testing.T) {
	t.Parallel()

	modes := []string{
		"ALL",
		"LAUNCH_NAME",
		"CURRENT_LAUNCH",
		"PREVIOUS_LAUNCH",
		"CURRENT_AND_THE_SAME_NAME",
	}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			got, err := resolveAnalyzeLaunchAnalyzerMode(mode)

			require.NoError(t, err)
			assert.Equal(t, types.AnalyzeLaunchRQAnalyzerMode(mode), got)
		})
	}
}

func TestResolveAnalyzeLaunchItemsMode_Default(t *testing.T) {
	t.Parallel()

	cmd := newLaunchAnalyzeCmd()

	modes, err := resolveAnalyzeLaunchItemsMode(cmd)

	require.NoError(t, err)
	require.Len(t, modes, 1)
	assert.Equal(t, types.TOINVESTIGATE, modes[0])
}

func TestResolveCompareLaunchIDs_RequiresAtLeastTwo(t *testing.T) {
	t.Parallel()

	cmd := newLaunchCompareCmd()

	_, err := resolveCompareLaunchIDs(cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least two launch IDs are required for compare (got 0)")

	require.NoError(t, cmd.Flags().Set("id", "41"))

	_, err = resolveCompareLaunchIDs(cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least two launch IDs are required for compare (got 1)")
}

func TestResolveCompareLaunchIDs_Valid(t *testing.T) {
	t.Parallel()

	cmd := newLaunchCompareCmd()
	require.NoError(t, cmd.Flags().Set("id", "41"))
	require.NoError(t, cmd.Flags().Set("id", "42"))

	ids, err := resolveCompareLaunchIDs(cmd)

	require.NoError(t, err)
	assert.Equal(t, []int64{41, 42}, ids)
}

func TestCompareLaunchTable(t *testing.T) {
	t.Parallel()

	launch41 := int64(41)
	launch42 := int64(42)

	result := map[string][]types.ChartStatisticsContent{
		"statistics$executions": {
			{
				Id:     &launch41,
				Values: &map[string]string{"total": "100", "passed": "95"},
			},
			{
				Id:     &launch42,
				Values: &map[string]string{"total": "120", "passed": "110"},
			},
		},
	}

	headers, rows := compareLaunchTable([]int64{41, 42}, result)

	assert.Equal(t, []string{"METRIC", "LAUNCH 41", "LAUNCH 42"}, headers)
	require.Len(t, rows, 2)
	assert.Equal(t, "statistics$executions$passed", rows[0][0])
	assert.Equal(t, "95", rows[0][1])
	assert.Equal(t, "110", rows[0][2])
	assert.Equal(t, "statistics$executions$total", rows[1][0])
	assert.Equal(t, "100", rows[1][1])
	assert.Equal(t, "120", rows[1][2])
}

func TestBuildAnalyzeLaunchRequest_FromFilePrecedence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "analyze.json")
	content := `{
		"launchId": 99,
		"analyzerMode": "ALL",
		"analyzeItemsMode": ["TO_INVESTIGATE"],
		"analyzerTypeName": "autoAnalyzer"
	}`

	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newLaunchAnalyzeCmd()
	require.NoError(t, cmd.Flags().Set("from-file", path))
	require.NoError(t, cmd.Flags().Set("analyzer-mode", "PREVIOUS_LAUNCH"))

	req, err := buildAnalyzeLaunchRequest(cmd, 42)

	require.NoError(t, err)
	assert.Equal(t, int64(99), req.LaunchId)
	assert.Equal(t, types.AnalyzeLaunchRQAnalyzerMode("ALL"), req.AnalyzerMode)
}
