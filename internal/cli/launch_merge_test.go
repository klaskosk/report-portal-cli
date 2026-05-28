package cli //nolint:testpackage // white-box tests for launch merge CLI helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMergeLaunchesRequest_NameRequired(t *testing.T) {
	t.Parallel()

	cmd := newLaunchMergeCmd()
	require.NoError(t, cmd.Flags().Set("launch", "41"))
	require.NoError(t, cmd.Flags().Set("launch", "42"))

	_, err := buildMergeLaunchesRequest(cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
}

func TestBuildMergeLaunchesRequest_LaunchRequired(t *testing.T) {
	t.Parallel()

	cmd := newLaunchMergeCmd()
	require.NoError(t, cmd.Flags().Set("name", "Merged Launch"))

	_, err := buildMergeLaunchesRequest(cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least two launch IDs are required for merge (got 0)")
}

func TestBuildMergeLaunchesRequest_AtLeastTwoLaunches(t *testing.T) {
	t.Parallel()

	cmd := newLaunchMergeCmd()
	require.NoError(t, cmd.Flags().Set("launch", "41"))
	require.NoError(t, cmd.Flags().Set("name", "Merged Launch"))

	_, err := buildMergeLaunchesRequest(cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least two launch IDs are required for merge (got 1)")
}

func TestBuildMergeLaunchesRequest_Valid(t *testing.T) {
	t.Parallel()

	cmd := newLaunchMergeCmd()
	require.NoError(t, cmd.Flags().Set("launch", "41"))
	require.NoError(t, cmd.Flags().Set("launch", "42"))
	require.NoError(t, cmd.Flags().Set("name", "Merged Nightly"))
	require.NoError(t, cmd.Flags().Set("merge-type", "DEEP"))
	require.NoError(t, cmd.Flags().Set("mode", "DEBUG"))
	require.NoError(t, cmd.Flags().Set("description", "merged run"))
	require.NoError(t, cmd.Flags().Set("extend-suites", "true"))
	require.NoError(t, cmd.Flags().Set("attribute", "env:staging"))

	req, err := buildMergeLaunchesRequest(cmd)

	require.NoError(t, err)
	assert.Equal(t, []int64{41, 42}, req.Launches)
	assert.Equal(t, "Merged Nightly", req.Name)
	assert.Equal(t, types.MergeLaunchesRQMergeTypeDEEP, req.MergeType)
	require.NotNil(t, req.Mode)
	assert.Equal(t, types.MergeLaunchesRQModeDEBUG, *req.Mode)
	require.NotNil(t, req.Description)
	assert.Equal(t, "merged run", *req.Description)
	assert.True(t, req.ExtendSuitesDescription)
	require.NotNil(t, req.Attributes)
	assert.Len(t, *req.Attributes, 1)
}

func TestResolveMergeLaunchesMergeType_Valid(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"BASIC", "DEEP", "basic", "deep"} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			got, err := resolveMergeLaunchesMergeType(raw)

			require.NoError(t, err)
			assert.NotEmpty(t, got)
		})
	}
}

func TestResolveMergeLaunchesMergeType_Invalid(t *testing.T) {
	t.Parallel()

	_, err := resolveMergeLaunchesMergeType("INVALID")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid merge-type")
}

func TestResolveMergeLaunchesMode_Valid(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"DEFAULT", "DEBUG", "default", "debug"} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			got, err := resolveMergeLaunchesMode(raw)

			require.NoError(t, err)
			assert.NotEmpty(t, got)
		})
	}
}

func TestResolveMergeLaunchesMode_Invalid(t *testing.T) {
	t.Parallel()

	_, err := resolveMergeLaunchesMode("INVALID")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")
}

func TestBuildMergeLaunchesRequest_FromFilePrecedence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "merge.json")
	content := `{"name":"From File","launches":[10,20],"mergeType":"BASIC"}`

	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newLaunchMergeCmd()
	require.NoError(t, cmd.Flags().Set("from-file", path))
	require.NoError(t, cmd.Flags().Set("name", "Flag Name"))
	require.NoError(t, cmd.Flags().Set("launch", "41"))
	require.NoError(t, cmd.Flags().Set("launch", "42"))

	req, err := buildMergeLaunchesRequest(cmd)

	require.NoError(t, err)
	assert.Equal(t, "From File", req.Name)
	assert.Equal(t, []int64{10, 20}, req.Launches)
	assert.Equal(t, types.MergeLaunchesRQMergeTypeBASIC, req.MergeType)
}
