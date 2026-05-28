package cli //nolint:testpackage // white-box tests for unexported launch lifecycle helpers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLaunchCmd_Subcommands_IncludesLifecycle(t *testing.T) {
	t.Parallel()

	cmd := newLaunchCmd()

	assert.Equal(
		t,
		[]string{"analyze", "compare", "delete", "finish", "get", "list", "merge", "report", "start", "update"},
		commandNames(cmd),
	)
}

func TestBuildLaunchStartRequestFromFlags_MissingName(t *testing.T) {
	t.Parallel()

	cmd := newLaunchStartCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	_, err := buildLaunchStartRequestFromFlags(cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
}

func TestBuildLaunchStartRequestFromFlags_ModeValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{name: "DEBUG accepted", mode: "DEBUG", wantErr: false},
		{name: "DEFAULT accepted", mode: "DEFAULT", wantErr: false},
		{name: "lowercase debug accepted", mode: "debug", wantErr: false},
		{name: "invalid mode rejected", mode: "INVALID", wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cmd := newLaunchStartCmd()
			require.NoError(t, cmd.ParseFlags([]string{
				"--name", "run",
				"--mode", testCase.mode,
			}))

			req, err := buildLaunchStartRequestFromFlags(cmd)
			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid mode")

				return
			}

			require.NoError(t, err)
			require.NotNil(t, req.Mode)
			assert.Equal(t, types.StartLaunchRQMode(strings.ToUpper(testCase.mode)), *req.Mode)
		})
	}
}

func TestBuildLaunchStartRequest_FromFile_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "start.json")
	content := `{
		"name": "from file",
		"startTime": "2024-01-15T10:30:00Z",
		"uuid": "abc-123"
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newLaunchStartCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path, "--name", "ignored"}))

	req, err := buildLaunchStartRequest(cmd)

	require.NoError(t, err)
	assert.Equal(t, "from file", req.Name)
	assert.Equal(t, "abc-123", req.Uuid)
	assert.False(t, req.StartTime.IsZero())
}

func TestLaunchFinishCmd_RequiresUUIDArg(t *testing.T) {
	t.Parallel()

	cmd := newLaunchFinishCmd()
	cmd.SetContext(t.Context())

	err := cmd.ExecuteContext(t.Context())

	require.Error(t, err)
}

func TestBuildLaunchFinishRequest_FromFile_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "finish.json")
	content := `{
		"endTime": "2024-01-15T10:30:00Z",
		"status": "PASSED",
		"description": "done"
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newLaunchFinishCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path, "--status", "FAILED"}))

	req, err := buildLaunchFinishRequest(cmd)

	require.NoError(t, err)
	require.NotNil(t, req.Status)
	assert.Equal(t, types.FinishExecutionRQStatusPASSED, *req.Status)
	require.NotNil(t, req.Description)
	assert.Equal(t, "done", *req.Description)
}

func TestBuildLaunchStartRequestFromFlags_SetsStartTimeAndUUID(t *testing.T) {
	t.Parallel()

	cmd := newLaunchStartCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--name", "run"}))

	before := time.Now()
	req, err := buildLaunchStartRequestFromFlags(cmd)
	after := time.Now()

	require.NoError(t, err)
	assert.False(t, req.StartTime.Before(before))
	assert.False(t, req.StartTime.After(after))
	assert.NotEmpty(t, req.Uuid)
}

func TestLaunchStartCmd_FromFileOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "start.json")
	content := `{
		"name": "file only",
		"startTime": "2024-01-15T10:30:00Z",
		"uuid": "launch-uuid"
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newLaunchStartCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	req, err := buildLaunchStartRequest(cmd)

	require.NoError(t, err)
	assert.Equal(t, "file only", req.Name)
}

func TestLaunchFinishCmd_ArgsValidation(t *testing.T) {
	t.Parallel()

	cmd := newLaunchFinishCmd()

	err := cmd.ValidateArgs([]string{})
	require.Error(t, err)

	err = cmd.ValidateArgs([]string{"uuid-1", "uuid-2"})
	require.Error(t, err)

	require.NoError(t, cmd.ValidateArgs([]string{"uuid-1"}))
}

// Ensure cobra wiring for finish uses ExactArgs.
func TestNewLaunchFinishCmd_ExactArgs(t *testing.T) {
	t.Parallel()

	cmd := newLaunchFinishCmd()

	assert.NotNil(t, cmd.Args)
}
