package cli //nolint:testpackage // white-box tests for unexported log helpers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLogScope records which create method was invoked, satisfying logCreateScope.
type fakeLogScope struct {
	calledMethod string
}

func (f *fakeLogScope) Create(
	_ context.Context,
	_ types.SaveLogRQ,
) (*types.EntryCreatedAsyncRS, error) {
	f.calledMethod = "Create"

	return &types.EntryCreatedAsyncRS{}, nil
}

func (f *fakeLogScope) CreateAsync(
	_ context.Context,
	_ types.SaveLogRQ,
) (*types.EntryCreatedAsyncRS, error) {
	f.calledMethod = "CreateAsync"

	return &types.EntryCreatedAsyncRS{}, nil
}

func (f *fakeLogScope) CreateBatch(
	_ context.Context,
	_ ...client.BatchLogEntry,
) (*types.BatchSaveOperatingRS, error) {
	f.calledMethod = "CreateBatch"

	return &types.BatchSaveOperatingRS{}, nil
}

func (f *fakeLogScope) CreateBatchAsync(
	_ context.Context,
	_ ...client.BatchLogEntry,
) (*types.BatchSaveOperatingRS, error) {
	f.calledMethod = "CreateBatchAsync"

	return &types.BatchSaveOperatingRS{}, nil
}

// --- Command wiring ---

func TestNewLogCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newLogCmd()

	assert.Equal(t, []string{"create", "delete", "get", "list"}, commandNames(cmd))
}

func TestNewLogCmd_ProjectFlagInheritedBySubcommands(t *testing.T) {
	t.Parallel()

	cmd := newLogCmd()

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

// --- Delete ID parsing ---

func TestParseLogDeleteID_nonNumeric(t *testing.T) {
	t.Parallel()

	_, err := parseIDArg("abc", "log")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log ID")
	assert.Contains(t, err.Error(), `"abc"`)
	assert.Contains(t, err.Error(), "must be a numeric ID")
}

func TestParseLogDeleteID_valid(t *testing.T) {
	t.Parallel()

	id, err := parseIDArg("42", "log")

	require.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

// --- Create flag validation ---

func TestRunLogCreate_missingFromFile(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newLogCreateCmd())

	err := runLogCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func writeTempJSON(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "payload.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func newLogCreateCmdWithFlags(t *testing.T, flags []string) *cobra.Command {
	t.Helper()

	cmd := attachCmdToRoot(t, newLogCreateCmd())
	require.NoError(t, cmd.ParseFlags(flags))

	return cmd
}

func TestRunLogCreate_validFileReachesConnectionStep_default(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	path := writeTempJSON(t, `{"launchUuid":"abc","time":"2024-01-15T10:30:00Z"}`)
	cmd := newLogCreateCmdWithFlags(t, []string{"--from-file", path})

	err := runLogCreate(cmd, nil)

	// File and flag parsing succeed; fails only at the network layer.
	// Variant selection is covered by TestDispatchSingleCreate_*.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunLogCreate_validFileReachesConnectionStep_async(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	path := writeTempJSON(t, `{"launchUuid":"abc","time":"2024-01-15T10:30:00Z"}`)
	cmd := newLogCreateCmdWithFlags(t, []string{"--from-file", path, "--async"})

	err := runLogCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunLogCreate_validFileReachesConnectionStep_batchSync(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	path := writeTempJSON(t, `[{"launchUuid":"abc","time":"2024-01-15T10:30:00Z"}]`)
	cmd := newLogCreateCmdWithFlags(t, []string{"--from-file", path, "--batch"})

	err := runLogCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunLogCreate_validFileReachesConnectionStep_batchAsync(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	path := writeTempJSON(t, `[{"launchUuid":"abc","time":"2024-01-15T10:30:00Z"}]`)
	cmd := newLogCreateCmdWithFlags(t, []string{"--from-file", path, "--batch", "--async"})

	err := runLogCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Report Portal URL configured")
}

// --- Create dispatch ---

func TestDispatchSingleCreate_syncMode(t *testing.T) {
	t.Parallel()

	fake := &fakeLogScope{}
	req := types.SaveLogRQ{LaunchUuid: "abc", Time: time.Now()}

	_, err := dispatchSingleCreate(context.Background(), fake, req, false)

	require.NoError(t, err)
	assert.Equal(t, "Create", fake.calledMethod)
}

func TestDispatchSingleCreate_asyncMode(t *testing.T) {
	t.Parallel()

	fake := &fakeLogScope{}
	req := types.SaveLogRQ{LaunchUuid: "abc", Time: time.Now()}

	_, err := dispatchSingleCreate(context.Background(), fake, req, true)

	require.NoError(t, err)
	assert.Equal(t, "CreateAsync", fake.calledMethod)
}

func TestDispatchBatchCreate_syncMode(t *testing.T) {
	t.Parallel()

	fake := &fakeLogScope{}
	entries := []client.BatchLogEntry{{LaunchUUID: "abc"}}

	_, err := dispatchBatchCreate(context.Background(), fake, entries, false)

	require.NoError(t, err)
	assert.Equal(t, "CreateBatch", fake.calledMethod)
}

func TestDispatchBatchCreate_asyncMode(t *testing.T) {
	t.Parallel()

	fake := &fakeLogScope{}
	entries := []client.BatchLogEntry{{LaunchUUID: "abc"}}

	_, err := dispatchBatchCreate(context.Background(), fake, entries, true)

	require.NoError(t, err)
	assert.Equal(t, "CreateBatchAsync", fake.calledMethod)
}

// --- Single mode validation ---

func TestReadSingleLogEntry_validJSON(t *testing.T) {
	t.Parallel()

	path := writeTempJSON(
		t,
		`{"launchUuid":"abc-123","itemUuid":"def-456","level":"ERROR","message":"boom","time":"2024-01-15T10:30:00Z"}`,
	)

	req, err := readSingleLogEntry(path)

	require.NoError(t, err)
	assert.Equal(t, "abc-123", req.LaunchUuid)
	require.NotNil(t, req.ItemUuid)
	assert.Equal(t, "def-456", *req.ItemUuid)
	require.NotNil(t, req.Level)
	assert.Equal(t, "ERROR", *req.Level)
	require.NotNil(t, req.Message)
	assert.Equal(t, "boom", *req.Message)
	assert.False(t, req.Time.IsZero())
}

func TestValidateSingleLogEntry_missingLaunchUuid(t *testing.T) {
	t.Parallel()

	req := types.SaveLogRQ{
		Time: time.Now(),
	}

	err := validateSingleLogEntry(req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "launchUuid")
}

func TestValidateSingleLogEntry_zeroTime(t *testing.T) {
	t.Parallel()

	req := types.SaveLogRQ{
		LaunchUuid: "abc-123",
	}

	err := validateSingleLogEntry(req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "time")
}

func TestValidateSingleLogEntry_valid(t *testing.T) {
	t.Parallel()

	req := types.SaveLogRQ{
		LaunchUuid: "abc-123",
		Time:       time.Now(),
	}

	err := validateSingleLogEntry(req)

	require.NoError(t, err)
}

func TestReadSingleLogEntry_malformedJSON(t *testing.T) {
	t.Parallel()

	path := writeTempJSON(t, `{invalid`)

	_, err := readSingleLogEntry(path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing log entry JSON")
}

func TestReadSingleLogEntry_missingFile(t *testing.T) {
	t.Parallel()

	_, err := readSingleLogEntry(filepath.Join(t.TempDir(), "missing.json"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading log entry file")
}

// --- Batch mode validation ---

func TestReadBatchLogEntries_validJSON(t *testing.T) {
	t.Parallel()

	path := writeTempJSON(t, `[
		{"launchUuid":"abc","level":"INFO","message":"started","time":"2024-01-15T10:30:00Z"},
		{"launchUuid":"abc","itemUuid":"def","level":"ERROR","message":"failed","time":"2024-01-15T10:31:00Z"}
	]`)

	entries, err := readBatchLogEntries(path)

	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "abc", entries[0].LaunchUUID)
	assert.Equal(t, "INFO", entries[0].Level)
	assert.Equal(t, "def", entries[1].ItemUUID)
	assert.Equal(t, "ERROR", entries[1].Level)
}

func TestValidateBatchLogEntries_emptySlice(t *testing.T) {
	t.Parallel()

	err := validateBatchLogEntries([]batchLogEntryJSON{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one entry")
}

func TestValidateBatchLogEntries_missingLaunchUUID(t *testing.T) {
	t.Parallel()

	entries := []batchLogEntryJSON{
		{LaunchUUID: "", Time: time.Now()},
	}

	err := validateBatchLogEntries(entries)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "launchUuid")
}

func TestValidateBatchLogEntries_zeroTime(t *testing.T) {
	t.Parallel()

	entries := []batchLogEntryJSON{
		{LaunchUUID: "abc"},
	}

	err := validateBatchLogEntries(entries)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "time")
}

func TestReadBatchLogEntries_nonArrayJSON(t *testing.T) {
	t.Parallel()

	path := writeTempJSON(t, `{"launchUuid":"abc","time":"2024-01-15T10:30:00Z"}`)

	_, err := readBatchLogEntries(path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing batch log entries JSON")
}

func TestReadBatchLogEntries_malformedJSON(t *testing.T) {
	t.Parallel()

	path := writeTempJSON(t, `[{invalid`)

	_, err := readBatchLogEntries(path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing batch log entries JSON")
}

func TestReadBatchLogEntries_missingFile(t *testing.T) {
	t.Parallel()

	_, err := readBatchLogEntries(filepath.Join(t.TempDir(), "missing.json"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading batch log entries file")
}

// --- Batch conversion ---

func TestToBatchLogEntry_mapsAllFields(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	input := batchLogEntryJSON{
		LaunchUUID: "launch-uuid",
		ItemUUID:   "item-uuid",
		Level:      "ERROR",
		Message:    "something failed",
		Time:       timestamp,
	}

	result := toBatchLogEntry(input)

	assert.Equal(t, "launch-uuid", result.LaunchUUID)
	assert.Equal(t, "item-uuid", result.ItemUUID)
	assert.Equal(t, "ERROR", result.Level)
	assert.Equal(t, "something failed", result.Message)
	assert.Equal(t, timestamp, result.Time)
	assert.Empty(t, result.AttachmentName, "attachments should not be set via CLI")
	assert.Empty(t, result.AttachmentData, "attachments should not be set via CLI")
}

// --- Batch table formatting ---

func TestBatchLogRows_nilResponses(t *testing.T) {
	t.Parallel()

	result := &types.BatchSaveOperatingRS{Responses: nil}

	rows := batchLogRows(result)

	assert.Nil(t, rows)
}

func TestBatchLogRows_singleEntry(t *testing.T) {
	t.Parallel()

	entryID := "550e8400-e29b-41d4-a716-446655440001"
	msg := "created"
	responses := []types.BatchElementCreatedRS{
		{Id: &entryID, Message: &msg},
	}
	result := &types.BatchSaveOperatingRS{Responses: &responses}

	rows := batchLogRows(result)

	require.Len(t, rows, 1)
	assert.Equal(t, entryID, rows[0][0])
	assert.Equal(t, msg, rows[0][1])
}

func TestBatchLogRows_multipleEntries(t *testing.T) {
	t.Parallel()

	id1 := "550e8400-e29b-41d4-a716-446655440001"
	id2 := "550e8400-e29b-41d4-a716-446655440002"
	msg2 := "error"
	responses := []types.BatchElementCreatedRS{
		{Id: &id1},
		{Id: &id2, Message: &msg2},
	}
	result := &types.BatchSaveOperatingRS{Responses: &responses}

	rows := batchLogRows(result)

	require.Len(t, rows, 2)
	assert.Equal(t, id1, rows[0][0])
	assert.Empty(t, rows[0][1])
	assert.Equal(t, id2, rows[1][0])
	assert.Equal(t, msg2, rows[1][1])
}

func TestBatchLogHeaders(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"ID", "MESSAGE"}, batchLogHeaders())
}
