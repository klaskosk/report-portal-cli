package logtype //nolint:testpackage // white-box tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogTypeCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	assert.Equal(t, []string{"create", "delete", "list", "update"}, testutil.CommandNames(cmd))
}

func TestNewLogTypeCmd_NoGetSubcommand(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	for _, sub := range cmd.Commands() {
		assert.NotEqual(t, "get", sub.Name(), "log-type should have no get subcommand")
	}
}

func TestNewLogTypeCmd_ProjectFlagInheritedBySubcommands(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

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

// --- ID parsing ---

func TestParseLogTypeID_valid(t *testing.T) {
	t.Parallel()

	id, err := parseLogTypeID("42")
	require.NoError(t, err)
	assert.EqualValues(t, 42, id)
}

func TestParseLogTypeID_invalid(t *testing.T) {
	t.Parallel()

	_, err := parseLogTypeID("abc")
	require.Error(t, err)
	assert.EqualError(t, err, `invalid log-type ID "abc": must be a numeric ID`)
}

func TestRunLogTypeUpdate_invalidID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "f.json")

	require.NoError(t, os.WriteFile(path, []byte(`{"name":"X","level":1}`), 0o600))

	cmd := testutil.AttachCmdToRoot(t, newLogTypeUpdateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	err := runLogTypeUpdate(cmd, []string{"abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log-type ID")
}

// --- Create flag validation ---

func TestRunLogTypeCreate_noInputMode(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newLogTypeCreateCmd())

	err := runLogTypeCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no input provided")
	assert.Contains(t, err.Error(), "--name")
	assert.Contains(t, err.Error(), "--from-file")
}

func TestRunLogTypeCreate_missingName(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newLogTypeCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--level", "5000"}))

	err := runLogTypeCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
}

func TestRunLogTypeCreate_missingLevel(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newLogTypeCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--name", "TRACE"}))

	err := runLogTypeCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "level must be greater than 0")
}

func TestRunLogTypeCreate_validFlags(t *testing.T) {
	t.Parallel()

	cmd := newLogTypeCreateCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{"--name", "TRACE", "--level", "5000"}))

	printer := core.NewPrinterFromCmd(cmd)

	request, err := resolveLogTypeCreateRequest(cmd, printer)
	require.NoError(t, err)
	assert.Equal(t, "TRACE", request.Name)
	assert.Equal(t, int32(5000), request.Level)
	assert.False(t, request.IsFilterable)
}

func TestRunLogTypeCreate_filterableDefaultsFalse(t *testing.T) {
	t.Parallel()

	cmd := newLogTypeCreateCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{"--name", "INFO", "--level", "20000"}))

	printer := core.NewPrinterFromCmd(cmd)

	request, err := resolveLogTypeCreateRequest(cmd, printer)
	require.NoError(t, err)
	assert.False(t, request.IsFilterable)
}

func TestRunLogTypeCreate_filterableExplicitTrue(t *testing.T) {
	t.Parallel()

	cmd := newLogTypeCreateCmd()
	cmd.SetContext(context.Background())
	require.NoError(
		t,
		cmd.ParseFlags([]string{"--name", "WARN", "--level", "30000", "--filterable"}),
	)

	printer := core.NewPrinterFromCmd(cmd)

	request, err := resolveLogTypeCreateRequest(cmd, printer)
	require.NoError(t, err)
	assert.True(t, request.IsFilterable)
}

// --- Create from-file mode ---

func TestRunLogTypeCreate_fromFile_valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "log_type.json")

	require.NoError(t, os.WriteFile(path, []byte(`{
		"name": "CUSTOM",
		"level": 15000,
		"is_filterable": true
	}`), 0o600))

	cmd := newLogTypeCreateCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	printer := core.NewPrinterFromCmd(cmd)

	request, err := resolveLogTypeCreateRequest(cmd, printer)
	require.NoError(t, err)
	assert.Equal(t, "CUSTOM", request.Name)
	assert.Equal(t, int32(15000), request.Level)
	assert.True(t, request.IsFilterable)
}

func TestRunLogTypeCreate_fromFile_missingName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "log_type.json")

	require.NoError(t, os.WriteFile(path, []byte(`{"level": 5000}`), 0o600))

	cmd := newLogTypeCreateCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	printer := core.NewPrinterFromCmd(cmd)

	_, err := resolveLogTypeCreateRequest(cmd, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name must not be empty")
}

func TestRunLogTypeCreate_fromFile_zeroLevel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "log_type.json")

	require.NoError(t, os.WriteFile(path, []byte(`{"name": "TRACE", "level": 0}`), 0o600))

	cmd := newLogTypeCreateCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	printer := core.NewPrinterFromCmd(cmd)

	_, err := resolveLogTypeCreateRequest(cmd, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "level must be greater than 0")
}

func TestRunLogTypeCreate_fromFile_nonExistent(t *testing.T) {
	t.Parallel()

	cmd := newLogTypeCreateCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", "/nonexistent/path.json"}))

	printer := core.NewPrinterFromCmd(cmd)

	_, err := resolveLogTypeCreateRequest(cmd, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "log type definition")
}

func TestRunLogTypeCreate_fromFile_malformedJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	require.NoError(t, os.WriteFile(path, []byte(`{invalid`), 0o600))

	cmd := newLogTypeCreateCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	printer := core.NewPrinterFromCmd(cmd)

	_, err := resolveLogTypeCreateRequest(cmd, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "log type JSON")
}

func TestRunLogTypeCreate_fromFileTakesPrecedenceOverFlags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "log_type.json")

	require.NoError(t, os.WriteFile(path, []byte(`{
		"name": "FROM_FILE",
		"level": 99000,
		"is_filterable": false
	}`), 0o600))

	cmd := newLogTypeCreateCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{
		"--name", "FROM_FLAG",
		"--level", "1000",
		"--from-file", path,
	}))

	printer := core.NewPrinterFromCmd(cmd)

	request, err := resolveLogTypeCreateRequest(cmd, printer)
	require.NoError(t, err)
	assert.Equal(
		t,
		"FROM_FILE",
		request.Name,
		"--from-file should take precedence over --name flag",
	)
	assert.Equal(
		t,
		int32(99000),
		request.Level,
		"--from-file should take precedence over --level flag",
	)
}

// --- Update validation ---

func TestRunLogTypeUpdate_fromFile_valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "update.json")

	require.NoError(t, os.WriteFile(path, []byte(`{
		"name": "UPDATED",
		"level": 25000,
		"is_filterable": true
	}`), 0o600))

	printer := core.NewPrinterFromCmd(newLogTypeUpdateCmd())

	request, err := loadAndValidateLogTypeRequest(path, printer)
	require.NoError(t, err)
	assert.Equal(t, "UPDATED", request.Name)
	assert.Equal(t, int32(25000), request.Level)
}

func TestRunLogTypeUpdate_fromFile_missingName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "update.json")

	require.NoError(t, os.WriteFile(path, []byte(`{"level": 5000}`), 0o600))

	printer := core.NewPrinterFromCmd(newLogTypeUpdateCmd())

	_, err := loadAndValidateLogTypeRequest(path, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name must not be empty")
}

func TestRunLogTypeUpdate_fromFile_zeroLevel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "update.json")

	require.NoError(t, os.WriteFile(path, []byte(`{"name": "X", "level": 0}`), 0o600))

	printer := core.NewPrinterFromCmd(newLogTypeUpdateCmd())

	_, err := loadAndValidateLogTypeRequest(path, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "level must be greater than 0")
}

func TestRunLogTypeUpdate_requiresFromFile(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newLogTypeUpdateCmd())

	err := runLogTypeUpdate(cmd, []string{"42"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
}

// --- Table formatting ---

func TestLogTypeRow_allFieldsPopulated(t *testing.T) {
	t.Parallel()

	id := int64(5)
	isSystem := true

	logType := &types.LogTypeResponse{
		Id:           &id,
		Name:         "TRACE",
		Level:        5000,
		IsFilterable: true,
		IsSystem:     &isSystem,
	}

	headers, row := logTypeRow(logType)

	assert.Equal(t, logTypeHeaders(), headers)
	assert.Equal(t, []string{"5", "TRACE", "5000", "true", "true"}, row)
}

func TestLogTypeRow_nilId(t *testing.T) {
	t.Parallel()

	logType := &types.LogTypeResponse{
		Name:  "TRACE",
		Level: 5000,
	}

	_, row := logTypeRow(logType)

	assert.Empty(t, row[0], "ID should be empty when nil")
}

func TestLogTypeRow_nilIsSystem(t *testing.T) {
	t.Parallel()

	id := int64(1)

	logType := &types.LogTypeResponse{
		Id:    &id,
		Name:  "ERROR",
		Level: 40000,
	}

	_, row := logTypeRow(logType)

	assert.Empty(t, row[4], "SYSTEM should be empty when IsSystem is nil")
}

func TestLogTypeRow_notFilterable(t *testing.T) {
	t.Parallel()

	id := int64(3)
	isSystem := false

	logType := &types.LogTypeResponse{
		Id:           &id,
		Name:         "DEBUG",
		Level:        10000,
		IsFilterable: false,
		IsSystem:     &isSystem,
	}

	_, row := logTypeRow(logType)

	assert.Equal(t, "false", row[3], "FILTERABLE should be false")
	assert.Equal(t, "false", row[4], "SYSTEM should be false")
}

// --- Validate log type request ---

func TestValidateLogTypeRequest_valid(t *testing.T) {
	t.Parallel()

	err := validateLogTypeRequest(types.LogTypeRequest{Name: "TRACE", Level: 5000})
	require.NoError(t, err)
}

func TestValidateLogTypeRequest_emptyName(t *testing.T) {
	t.Parallel()

	err := validateLogTypeRequest(types.LogTypeRequest{Name: "", Level: 5000})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name must not be empty")
}

func TestValidateLogTypeRequest_zeroLevel(t *testing.T) {
	t.Parallel()

	err := validateLogTypeRequest(types.LogTypeRequest{Name: "X", Level: 0})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "level must be greater than 0")
}

func TestValidateLogTypeRequest_negativeLevel(t *testing.T) {
	t.Parallel()

	err := validateLogTypeRequest(types.LogTypeRequest{Name: "X", Level: -1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "level must be greater than 0")
}
