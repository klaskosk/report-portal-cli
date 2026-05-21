package cli //nolint:testpackage // white-box tests for integration CLI helpers

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIntegrationCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newIntegrationCmd()

	assert.Equal(t, []string{"create", "delete", "get", "list", "update"}, commandNames(cmd))
}

func TestNewIntegrationCmd_ProjectFlagInheritedBySubcommands(t *testing.T) {
	t.Parallel()

	cmd := newIntegrationCmd()

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

func TestResolveIntegrationOps_noProjectFlag_globalScope(t *testing.T) {
	t.Parallel()

	cmd := newIntegrationCmd()
	deps := &cmdDeps{printer: resolvePrinter(cmd)}

	ops := resolveIntegrationOps(cmd, deps)

	assert.IsType(t, &globalIntegrationOps{}, ops)
}

func TestResolveIntegrationOps_withProjectFlag_projectScope(t *testing.T) {
	t.Parallel()

	cmd := newIntegrationCmd()
	require.NoError(t, cmd.ParseFlags([]string{"-p", "my_project"}))

	deps := &cmdDeps{printer: resolvePrinter(cmd)}

	ops := resolveIntegrationOps(cmd, deps)

	projectOps, ok := ops.(*projectIntegrationOps)
	require.True(t, ok, "expected *projectIntegrationOps")
	assert.Equal(t, "my_project", projectOps.project)
}

func TestResolveIntegrationOps_emptyProjectFlag_createsProjectScope(t *testing.T) {
	// resolveIntegrationOps no longer validates the empty-project case; that is
	// handled by validateIntegrationProject before resolveCmd is called.
	t.Parallel()

	cmd := newIntegrationCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--project", ""}))

	deps := &cmdDeps{printer: resolvePrinter(cmd)}

	ops := resolveIntegrationOps(cmd, deps)

	projectOps, ok := ops.(*projectIntegrationOps)
	require.True(t, ok, "expected *projectIntegrationOps")
	assert.Empty(t, projectOps.project)
}

func TestValidateIntegrationProject_noProjectFlag_noError(t *testing.T) {
	t.Parallel()

	cmd := newIntegrationCmd()

	err := validateIntegrationProject(cmd, resolvePrinter(cmd))

	require.NoError(t, err)
}

func TestValidateIntegrationProject_emptyProjectFlag_errors(t *testing.T) {
	t.Parallel()

	cmd := newIntegrationCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--project", ""}))

	err := validateIntegrationProject(cmd, resolvePrinter(cmd))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "project name cannot be empty")
}

// integrationSubCmdForTest wires the named integration subcommand into the full
// integration → root tree, simulates cobra's mergeFlags so that the persistent
// --project flag is visible on the subcommand's Flags(), and sets a discarded-
// logger context. Use this when testing run functions that need the --project flag.
func integrationSubCmdForTest(t *testing.T, subName string) *cobra.Command {
	t.Helper()

	integration := newIntegrationCmd()
	root := newRootCmd()
	root.AddCommand(integration)

	var sub *cobra.Command

	for _, s := range integration.Commands() {
		if s.Name() == subName {
			sub = s

			break
		}
	}

	require.NotNil(t, sub, "integration subcommand %q not found", subName)

	// Simulate cobra's internal mergeFlags() so the persistent --project flag from
	// the parent is accessible via sub.Flags().
	sub.Flags().AddFlagSet(integration.PersistentFlags())

	ctx := context.WithValue(context.Background(), loggerKey{}, slog.New(slog.DiscardHandler))
	sub.SetContext(ctx)

	return sub
}

func TestParseIntegrationID_valid(t *testing.T) {
	t.Parallel()

	integrationID, err := parseIntegrationID("42")
	require.NoError(t, err)
	assert.EqualValues(t, 42, integrationID)
}

func TestParseIntegrationID_invalid(t *testing.T) {
	t.Parallel()

	_, err := parseIntegrationID("abc")
	require.Error(t, err)
	assert.EqualError(t, err, `invalid integration ID "abc": must be a numeric ID`)
}

func TestRunIntegrationList_rejectsEmptyProjectBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := integrationSubCmdForTest(t, "list")
	require.NoError(t, cmd.ParseFlags([]string{"--project", ""}))

	err := runIntegrationList(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "project name cannot be empty")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunIntegrationGet_rejectsEmptyProjectBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := integrationSubCmdForTest(t, "get")
	require.NoError(t, cmd.ParseFlags([]string{"--project", ""}))

	err := runIntegrationGet(cmd, []string{"42"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "project name cannot be empty")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunIntegrationCreate_rejectsEmptyProjectBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	dir := t.TempDir()
	jsonFile := filepath.Join(dir, "integration.json")
	require.NoError(t, os.WriteFile(jsonFile, []byte("{}"), 0o600))

	cmd := integrationSubCmdForTest(t, "create")
	require.NoError(
		t,
		cmd.ParseFlags([]string{"--plugin", "jira", "--from-file", jsonFile, "--project", ""}),
	)

	err := runIntegrationCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "project name cannot be empty")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunIntegrationUpdate_rejectsEmptyProjectBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	dir := t.TempDir()
	jsonFile := filepath.Join(dir, "integration.json")
	require.NoError(t, os.WriteFile(jsonFile, []byte("{}"), 0o600))

	cmd := integrationSubCmdForTest(t, "update")
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", jsonFile, "--project", ""}))

	err := runIntegrationUpdate(cmd, []string{"42"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "project name cannot be empty")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunIntegrationDelete_rejectsEmptyProjectBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := integrationSubCmdForTest(t, "delete")
	require.NoError(t, cmd.ParseFlags([]string{"--project", ""}))

	err := runIntegrationDelete(cmd, []string{"42"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "project name cannot be empty")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunIntegrationCreate_requiresPluginBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newIntegrationCreateCmd())

	err := runIntegrationCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--plugin is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunIntegrationCreate_rejectsEmptyPlugin(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newIntegrationCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--plugin", ""}))

	err := runIntegrationCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--plugin cannot be empty")
}

func TestRunIntegrationCreate_requiresFromFileBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newIntegrationCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--plugin", "email"}))

	err := runIntegrationCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunIntegrationUpdate_requiresFromFileBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newIntegrationUpdateCmd())

	err := runIntegrationUpdate(cmd, []string{"42"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestReadIntegrationRequestFromFile_happyPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "integration.json")

	jsonContent := `{
  "name": "Email Notifications",
  "enabled": true,
  "integrationParameters": {
    "host": "smtp.example.com",
    "port": 587
  }
}`

	require.NoError(t, os.WriteFile(path, []byte(jsonContent), 0o600))

	request, err := readIntegrationRequestFromFile(path)
	require.NoError(t, err)

	require.NotNil(t, request.Name)
	assert.Equal(t, "Email Notifications", *request.Name)
	require.NotNil(t, request.Enabled)
	assert.True(t, *request.Enabled)
	require.NotNil(t, request.IntegrationParameters)
	assert.Equal(t, "smtp.example.com", (*request.IntegrationParameters)["host"])
}

func TestReadIntegrationRequestFromFile_invalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{`), 0o600))

	_, err := readIntegrationRequestFromFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing integration JSON")
}

func TestReadIntegrationRequestFromFile_missingFile(t *testing.T) {
	t.Parallel()

	_, err := readIntegrationRequestFromFile(filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading integration definition")
}

//nolint:paralleltest // mutates os.Stdin
func TestReadIntegrationRequestFromFile_stdin(
	t *testing.T,
) {
	jsonContent := `{"name":"stdin-integration","enabled":true}`

	pipeRead, pipeWrite, err := os.Pipe()
	require.NoError(t, err)

	_, err = pipeWrite.WriteString(jsonContent)
	require.NoError(t, err)
	require.NoError(t, pipeWrite.Close())

	origStdin := os.Stdin
	os.Stdin = pipeRead

	defer func() {
		os.Stdin = origStdin

		require.NoError(t, pipeRead.Close())
	}()

	request, err := readIntegrationRequestFromFile("-")
	require.NoError(t, err)
	require.NotNil(t, request.Name)
	assert.Equal(t, "stdin-integration", *request.Name)
}

func TestIntegrationRow_allFieldsPopulated(t *testing.T) {
	t.Parallel()

	integrationID := int64(1)
	name := "JIRA Integration"
	enabled := true
	creator := "superadmin"
	pluginName := "jira"
	integration := &types.IntegrationResource{
		Id:      &integrationID,
		Name:    &name,
		Enabled: &enabled,
		Creator: &creator,
		IntegrationType: &types.IntegrationTypeResource{
			Name: &pluginName,
		},
	}

	headers, row := integrationRow(integration)

	assert.Equal(t, integrationHeaders(), headers)
	assert.Equal(t, []string{"1", "JIRA Integration", "jira", "true", "superadmin"}, row)
}

func TestIntegrationRow_allNilFields(t *testing.T) {
	t.Parallel()

	integration := &types.IntegrationResource{}

	headers, row := integrationRow(integration)

	assert.Equal(t, integrationHeaders(), headers)
	assert.Equal(t, []string{"", "", "", "", ""}, row)
}

func TestIntegrationRow_nilIntegrationType(t *testing.T) {
	t.Parallel()

	integrationID := int64(5)
	name := "My Integration"
	integration := &types.IntegrationResource{
		Id:              &integrationID,
		Name:            &name,
		IntegrationType: nil,
	}

	_, row := integrationRow(integration)

	assert.Empty(t, row[2], "plugin column should be empty when IntegrationType is nil")
}

func TestIntegrationRow_nilIntegrationTypeName(t *testing.T) {
	t.Parallel()

	integrationID := int64(5)
	integration := &types.IntegrationResource{
		Id: &integrationID,
		IntegrationType: &types.IntegrationTypeResource{
			Name: nil,
		},
	}

	_, row := integrationRow(integration)

	assert.Empty(t, row[2], "plugin column should be empty when IntegrationType.Name is nil")
}

func TestIntegrationListCmd_hasPluginFlag(t *testing.T) {
	t.Parallel()

	cmd := newIntegrationListCmd()

	flag := cmd.Flags().Lookup("plugin")
	require.NotNil(t, flag, "list subcommand should have --plugin flag")
	assert.Empty(t, flag.DefValue)
}

func TestIntegrationCreateCmd_hasPluginAndFromFileFlags(t *testing.T) {
	t.Parallel()

	cmd := newIntegrationCreateCmd()

	pluginFlag := cmd.Flags().Lookup("plugin")
	require.NotNil(t, pluginFlag, "create subcommand should have --plugin flag")

	fromFileFlag := cmd.Flags().Lookup("from-file")
	require.NotNil(t, fromFileFlag, "create subcommand should have --from-file flag")
}
