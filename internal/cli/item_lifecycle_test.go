package cli //nolint:testpackage // white-box tests for unexported item lifecycle helpers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeItemStartScope struct {
	calledMethod string
	parentUUID   string
}

func (f *fakeItemStartScope) StartRoot(
	_ context.Context,
	_ types.StartTestItemRQ,
) (*types.EntryCreatedAsyncRS, error) {
	f.calledMethod = "StartRoot"

	return &types.EntryCreatedAsyncRS{}, nil
}

func (f *fakeItemStartScope) StartChild(
	_ context.Context,
	parentUUID string,
	_ types.StartTestItemRQ,
) (*types.EntryCreatedAsyncRS, error) {
	f.calledMethod = "StartChild"
	f.parentUUID = parentUUID

	return &types.EntryCreatedAsyncRS{}, nil
}

func TestNewItemCmd_Subcommands_IncludesLifecycle(t *testing.T) {
	t.Parallel()

	cmd := newItemCmd()

	assert.Equal(
		t,
		[]string{"delete", "finish", "get", "list", "start", "update"},
		commandNames(cmd),
	)
}

func TestBuildItemStartRequestFromFlags_MissingRequiredFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing name",
			args: []string{"--launch-uuid", "launch-1", "--type", "SUITE"},
			want: "--name is required",
		},
		{
			name: "missing launch uuid",
			args: []string{"--name", "suite", "--type", "SUITE"},
			want: "--launch-uuid is required",
		},
		{
			name: "missing type",
			args: []string{"--name", "suite", "--launch-uuid", "launch-1"},
			want: "--type is required",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cmd := newItemStartCmd()
			require.NoError(t, cmd.ParseFlags(testCase.args))

			_, err := buildItemStartRequestFromFlags(cmd)

			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.want)
		})
	}
}

func TestResolveStartTestItemType_Validation(t *testing.T) {
	t.Parallel()

	validTypes := []string{"SUITE", "STORY", "TEST", "STEP", "BEFORE_CLASS", "suite"}

	for _, itemType := range validTypes {
		t.Run(itemType, func(t *testing.T) {
			t.Parallel()

			resolved, err := resolveStartTestItemType(itemType)

			require.NoError(t, err)
			assert.Equal(t, types.StartTestItemRQType(strings.ToUpper(itemType)), resolved)
		})
	}

	_, err := resolveStartTestItemType("INVALID")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

func TestDispatchItemStart_RoutesToStartChild(t *testing.T) {
	t.Parallel()

	fake := &fakeItemStartScope{}
	req := types.StartTestItemRQ{
		Name:       "child",
		LaunchUuid: "launch-1",
		Type:       types.TEST,
		StartTime:  time.Now(),
		Uuid:       "item-uuid",
	}

	_, err := dispatchItemStart(context.Background(), fake, "parent-uuid", req)

	require.NoError(t, err)
	assert.Equal(t, "StartChild", fake.calledMethod)
	assert.Equal(t, "parent-uuid", fake.parentUUID)
}

func TestDispatchItemStart_RoutesToStartRoot(t *testing.T) {
	t.Parallel()

	fake := &fakeItemStartScope{}
	req := types.StartTestItemRQ{
		Name:       "root",
		LaunchUuid: "launch-1",
		Type:       types.SUITE,
		StartTime:  time.Now(),
		Uuid:       "item-uuid",
	}

	_, err := dispatchItemStart(context.Background(), fake, "", req)

	require.NoError(t, err)
	assert.Equal(t, "StartRoot", fake.calledMethod)
}

func TestBuildItemStartRequest_FromFile_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "start.json")
	content := `{
		"name": "from file",
		"launchUuid": "launch-1",
		"type": "SUITE",
		"startTime": "2024-01-15T10:30:00Z",
		"uuid": "item-uuid"
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newItemStartCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--from-file", path,
		"--name", "ignored",
		"--launch-uuid", "ignored",
		"--type", "TEST",
	}))

	req, err := buildItemStartRequest(cmd)

	require.NoError(t, err)
	assert.Equal(t, "from file", req.Name)
	assert.Equal(t, "launch-1", req.LaunchUuid)
	assert.Equal(t, types.SUITE, req.Type)
}

func TestItemFinishCmd_RequiresUUIDArg(t *testing.T) {
	t.Parallel()

	cmd := newItemFinishCmd()

	err := cmd.ValidateArgs([]string{})
	require.Error(t, err)

	err = cmd.ValidateArgs([]string{"uuid-1", "uuid-2"})
	require.Error(t, err)

	require.NoError(t, cmd.ValidateArgs([]string{"uuid-1"}))
}

func TestBuildItemFinishRequestFromFlags_MissingLaunchUUID(t *testing.T) {
	t.Parallel()

	cmd := newItemFinishCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	_, err := buildItemFinishRequestFromFlags(cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--launch-uuid is required")
}

func TestBuildItemFinishRequest_FromFile_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "finish.json")
	content := `{
		"launchUuid": "launch-1",
		"endTime": "2024-01-15T10:30:00Z",
		"status": "PASSED"
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newItemFinishCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path, "--status", "FAILED"}))

	req, err := buildItemFinishRequest(cmd)

	require.NoError(t, err)
	assert.Equal(t, "launch-1", req.LaunchUuid)
	require.NotNil(t, req.Status)
	assert.Equal(t, types.FinishTestItemRQStatusPASSED, *req.Status)
}

func TestBuildItemStartRequestFromFlags_SetsStartTimeAndUUID(t *testing.T) {
	t.Parallel()

	cmd := newItemStartCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--name", "suite",
		"--launch-uuid", "launch-1",
		"--type", "SUITE",
	}))

	before := time.Now()
	req, err := buildItemStartRequestFromFlags(cmd)
	after := time.Now()

	require.NoError(t, err)
	assert.False(t, req.StartTime.Before(before))
	assert.False(t, req.StartTime.After(after))
	assert.NotEmpty(t, req.Uuid)
}
