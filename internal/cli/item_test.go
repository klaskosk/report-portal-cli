package cli //nolint:testpackage // white-box tests for unexported item helpers

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOptionalInt64Flag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		flagName  string
		value     string
		wantVal   int64
		wantFound bool
		wantErr   string
	}{
		{
			name:      "empty value returns not found",
			flagName:  "launch",
			value:     "",
			wantVal:   0,
			wantFound: false,
		},
		{
			name:      "valid positive ID",
			flagName:  "launch",
			value:     "101",
			wantVal:   101,
			wantFound: true,
		},
		{
			name:      "valid negative ID",
			flagName:  "parent",
			value:     "-1",
			wantVal:   -1,
			wantFound: true,
		},
		{
			name:      "max int32 is accepted",
			flagName:  "launch",
			value:     "2147483647",
			wantVal:   math.MaxInt32,
			wantFound: true,
		},
		{
			name:      "min int32 is accepted",
			flagName:  "launch",
			value:     "-2147483648",
			wantVal:   math.MinInt32,
			wantFound: true,
		},
		{
			name:     "non-numeric value",
			flagName: "launch",
			value:    "abc",
			wantErr:  `invalid --launch value "abc": must be a numeric ID`,
		},
		{
			name:     "value exceeds int32 max",
			flagName: "launch",
			value:    "2147483648",
			wantErr:  `invalid --launch value "2147483648": ID must fit in a signed 32-bit integer`,
		},
		{
			name:     "value below int32 min",
			flagName: "parent",
			value:    "-2147483649",
			wantErr:  `invalid --parent value "-2147483649": ID must fit in a signed 32-bit integer`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, found, err := parseOptionalInt64Flag(test.flagName, test.value)

			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantVal, got)
			assert.Equal(t, test.wantFound, found)
		})
	}
}

func TestNewItemCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newItemCmd()

	assert.Equal(t, []string{"delete", "get", "list", "update"}, commandNames(cmd))
}

func TestItemUpdateCmd_InvalidID(t *testing.T) {
	t.Parallel()

	cmd := newItemUpdateCmd()
	cmd.SetContext(t.Context())
	require.NoError(t, cmd.ParseFlags([]string{"--description", "x"}))

	err := runItemUpdate(cmd, []string{"abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid item ID")
}

func TestItemUpdateCmd_NoFlags(t *testing.T) {
	t.Parallel()

	cmd := newItemUpdateCmd()
	cmd.SetContext(t.Context())
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := runItemUpdate(cmd, []string{"100"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one of")
}

func TestBuildItemUpdateRequestFromFlags_SingleFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flags    []string
		assertFn func(t *testing.T, req types.UpdateTestItemRQ)
	}{
		{
			name:  "--description alone is valid",
			flags: []string{"--description", "my desc"},
			assertFn: func(t *testing.T, req types.UpdateTestItemRQ) {
				t.Helper()
				require.NotNil(t, req.Description)
				assert.Equal(t, "my desc", *req.Description)
				assert.Nil(t, req.Status)
				assert.Nil(t, req.Attributes)
			},
		},
		{
			name:  "--status alone is valid",
			flags: []string{"--status", "passed"},
			assertFn: func(t *testing.T, req types.UpdateTestItemRQ) {
				t.Helper()
				require.NotNil(t, req.Status)
				assert.Equal(t, "PASSED", *req.Status)
				assert.Nil(t, req.Description)
				assert.Nil(t, req.Attributes)
			},
		},
		{
			name:  "--attribute alone is valid",
			flags: []string{"--attribute", "env:prod"},
			assertFn: func(t *testing.T, req types.UpdateTestItemRQ) {
				t.Helper()
				require.NotNil(t, req.Attributes)
				assert.Len(t, *req.Attributes, 1)
				assert.Equal(t, "env", *(*req.Attributes)[0].Key)
				assert.Equal(t, "prod", (*req.Attributes)[0].Value)
				assert.Nil(t, req.Description)
				assert.Nil(t, req.Status)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cmd := newItemUpdateCmd()
			require.NoError(t, cmd.ParseFlags(testCase.flags))

			req, err := buildItemUpdateRequestFromFlags(cmd)
			require.NoError(t, err)
			testCase.assertFn(t, req)
		})
	}
}

func TestBuildItemUpdateRequest_FromFile_MatchesJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "item_update.json")
	content := `{"description":"regression test","status":"PASSED","attributes":[{"key":"env","value":"staging"}]}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newItemUpdateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	req, err := buildItemUpdateRequest(cmd)
	require.NoError(t, err)
	require.NotNil(t, req.Description)
	assert.Equal(t, "regression test", *req.Description)
	require.NotNil(t, req.Status)
	assert.Equal(t, "PASSED", *req.Status)
	require.NotNil(t, req.Attributes)
	assert.Len(t, *req.Attributes, 1)
	assert.Equal(t, "env", *(*req.Attributes)[0].Key)
	assert.Equal(t, "staging", (*req.Attributes)[0].Value)
}

func TestBuildItemUpdateRequest_FromFile_IgnoresFlags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "item_update.json")
	// File only sets description; --status flag should be ignored.
	content := `{"description":"from file only"}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newItemUpdateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path, "--status", "FAILED"}))

	req, err := buildItemUpdateRequest(cmd)
	require.NoError(t, err)
	require.NotNil(t, req.Description)
	assert.Equal(t, "from file only", *req.Description)
	assert.Nil(t, req.Status, "--status flag must be ignored when --from-file is set")
}

func TestItemDeleteCmd_InvalidID(t *testing.T) {
	t.Parallel()

	cmd := newItemDeleteCmd()
	cmd.SetContext(t.Context())

	err := runItemDelete(cmd, []string{"abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid item ID")
}

func TestResolveItemStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rawValue string
		want     string
	}{
		{name: "lowercase passed", rawValue: "passed", want: "PASSED"},
		{name: "mixed case Failed", rawValue: "Failed", want: "FAILED"},
		{name: "already uppercase", rawValue: "SKIPPED", want: "SKIPPED"},
		{name: "with whitespace", rawValue: "  passed  ", want: "PASSED"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{}
			cmd.Flags().String("status", "", "")
			require.NoError(t, cmd.Flags().Set("status", testCase.rawValue))

			got := resolveItemStatus(cmd)
			require.NotNil(t, got)
			assert.Equal(t, testCase.want, *got)
		})
	}
}

func TestResolveItemStatus_NotSet(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.Flags().String("status", "", "")

	got := resolveItemStatus(cmd)
	assert.Nil(t, got)
}

func TestBuildItemAttributeResources_Valid(t *testing.T) {
	t.Parallel()

	resources, err := buildItemAttributeResources([]string{"env:prod", "feature:login"})

	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "env", *resources[0].Key)
	assert.Equal(t, "prod", resources[0].Value)
	assert.Equal(t, "feature", *resources[1].Key)
	assert.Equal(t, "login", resources[1].Value)
}

func TestBuildItemAttributeResources_InvalidFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"missing colon", "no-colon"},
		{"empty key", ":value"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := buildItemAttributeResources([]string{testCase.input})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid attribute format")
		})
	}
}

func TestItemUpdateCmd_FromFile_ValidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "item_update.json")
	content := `{"description":"from file","status":"PASSED"}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newItemUpdateCmd()
	cmd.SetContext(t.Context())
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	// No connection available; error is expected from resolveCmd, not from file parsing.
	// The test verifies that file parsing does not produce an error before resolveCmd.
	err := runItemUpdate(cmd, []string{"100"})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "reading item update file")
	assert.NotContains(t, err.Error(), "parsing item update JSON")
}

func TestItemUpdateCmd_FromFile_TakesPrecedenceOverFlags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "item_update.json")
	content := `{"description":"from file"}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newItemUpdateCmd()
	cmd.SetContext(t.Context())
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path, "--status", "FAILED"}))

	// Error comes from resolveCmd (no connection), not from flag conflict.
	err := runItemUpdate(cmd, []string{"100"})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "reading item update file")
	assert.NotContains(t, err.Error(), "parsing item update JSON")
}

func TestItemUpdateCmd_FromFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{invalid`), 0o600))

	cmd := newItemUpdateCmd()
	cmd.SetContext(t.Context())
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	err := runItemUpdate(cmd, []string{"100"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing item update JSON")
}
