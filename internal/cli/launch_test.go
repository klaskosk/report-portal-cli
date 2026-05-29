package cli //nolint:testpackage // white-box tests for unexported launch helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAttribute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantValue string
		wantFound bool
	}{
		{"valid key:value", "env:prod", "env", "prod", true},
		{"empty value", "env:", "env", "", true},
		{
			"value with colons",
			"url:http://example.com:8080",
			"url",
			"http://example.com:8080",
			true,
		},
		{"empty key", ":value", "", "", false},
		{"missing colon", "no-colon", "", "", false},
		{"empty string", "", "", "", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			key, value, found := parseAttribute(test.input)
			assert.Equal(t, test.wantKey, key)
			assert.Equal(t, test.wantValue, value)
			assert.Equal(t, test.wantFound, found)
		})
	}
}

func TestApplyLaunchFilterOptions_MalformedAttribute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"missing colon", "missing-colon"},
		{"empty key", ":value"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			flags := launchFilterFlags{
				attributes: []string{test.input},
			}

			_, err := applyLaunchFilterOptions(flags)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid attribute format")
			assert.Contains(t, err.Error(), test.input)
		})
	}
}

func TestApplyLaunchFilterOptions_ValidAttributes(t *testing.T) {
	t.Parallel()

	flags := launchFilterFlags{
		attributes: []string{"env:prod", "team:backend"},
	}

	options, err := applyLaunchFilterOptions(flags)
	require.NoError(t, err)
	assert.Len(t, options, 2)
}

func TestNewLaunchCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newLaunchCmd()

	assert.Equal(
		t,
		[]string{
			"analyze",
			"compare",
			"delete",
			"finish",
			"get",
			"list",
			"merge",
			"report",
			"start",
			"update",
		},
		commandNames(cmd),
	)
}

func TestLaunchUpdateCmd_NonNumericID(t *testing.T) {
	t.Parallel()

	_, err := parseIDArg("abc", "launch")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid launch ID")
	assert.Contains(t, err.Error(), `"abc"`)
}

func TestBuildUpdateLaunchRequestFromFlags_NoFlags(t *testing.T) {
	t.Parallel()

	cmd := newLaunchUpdateCmd()

	_, err := buildUpdateLaunchRequestFromFlags(cmd)

	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"at least one of --description, --mode, --attribute, or --from-file must be provided",
	)
}

func TestBuildUpdateLaunchRequestFromFlags_DescriptionAlone(t *testing.T) {
	t.Parallel()

	cmd := newLaunchUpdateCmd()
	require.NoError(t, cmd.Flags().Set("description", "new desc"))

	req, err := buildUpdateLaunchRequestFromFlags(cmd)

	require.NoError(t, err)
	require.NotNil(t, req.Description)
	assert.Equal(t, "new desc", *req.Description)
	assert.Nil(t, req.Mode)
	assert.Nil(t, req.Attributes)
}

func TestBuildUpdateLaunchRequestFromFlags_ModeAlone(t *testing.T) {
	t.Parallel()

	cmd := newLaunchUpdateCmd()
	require.NoError(t, cmd.Flags().Set("mode", "DEBUG"))

	req, err := buildUpdateLaunchRequestFromFlags(cmd)

	require.NoError(t, err)
	assert.Nil(t, req.Description)
	require.NotNil(t, req.Mode)
	assert.Equal(t, types.UpdateLaunchRQModeDEBUG, *req.Mode)
	assert.Nil(t, req.Attributes)
}

func TestBuildUpdateLaunchRequestFromFlags_InvalidMode(t *testing.T) {
	t.Parallel()

	cmd := newLaunchUpdateCmd()
	require.NoError(t, cmd.Flags().Set("mode", "INVALID"))

	_, err := buildUpdateLaunchRequestFromFlags(cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")
}

func TestBuildUpdateLaunchRequestFromFlags_AttributeAlone(t *testing.T) {
	t.Parallel()

	cmd := newLaunchUpdateCmd()
	require.NoError(t, cmd.Flags().Set("attribute", "env:prod"))

	req, err := buildUpdateLaunchRequestFromFlags(cmd)

	require.NoError(t, err)
	assert.Nil(t, req.Description)
	assert.Nil(t, req.Mode)
	require.NotNil(t, req.Attributes)
	assert.Len(t, *req.Attributes, 1)
}

func TestResolveUpdateLaunchMode_DEBUG(t *testing.T) {
	t.Parallel()

	mode, err := resolveUpdateLaunchMode("DEBUG")

	require.NoError(t, err)
	assert.Equal(t, types.UpdateLaunchRQModeDEBUG, mode)
}

func TestResolveUpdateLaunchMode_DEFAULT(t *testing.T) {
	t.Parallel()

	mode, err := resolveUpdateLaunchMode("DEFAULT")

	require.NoError(t, err)
	assert.Equal(t, types.UpdateLaunchRQModeDEFAULT, mode)
}

func TestResolveUpdateLaunchMode_LowercaseNormalized(t *testing.T) {
	t.Parallel()

	mode, err := resolveUpdateLaunchMode("debug")

	require.NoError(t, err)
	assert.Equal(t, types.UpdateLaunchRQModeDEBUG, mode)
}

func TestResolveUpdateLaunchMode_Invalid(t *testing.T) {
	t.Parallel()

	_, err := resolveUpdateLaunchMode("INVALID")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")
	assert.Contains(t, err.Error(), "INVALID")
}

func TestBuildAttributeResources_Valid(t *testing.T) {
	t.Parallel()

	resources, err := buildAttributeResources([]string{"env:prod", "team:backend"})

	require.NoError(t, err)
	require.Len(t, resources, 2)

	require.NotNil(t, resources[0].Key)
	assert.Equal(t, "env", *resources[0].Key)
	assert.Equal(t, "prod", resources[0].Value)

	require.NotNil(t, resources[1].Key)
	assert.Equal(t, "team", *resources[1].Key)
	assert.Equal(t, "backend", resources[1].Value)
}

func TestBuildAttributeResources_InvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := buildAttributeResources([]string{"no-colon"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid attribute format")
	assert.Contains(t, err.Error(), "no-colon")
}

func TestBuildAttributeResources_EmptyKey(t *testing.T) {
	t.Parallel()

	_, err := buildAttributeResources([]string{":value"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid attribute format")
}

func TestBuildAttributeResources_Empty(t *testing.T) {
	t.Parallel()

	resources, err := buildAttributeResources([]string{})

	require.NoError(t, err)
	assert.Empty(t, resources)
}

func TestLaunchUpdateCmd_FromFile_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "update.json")
	content := `{"description": "from file", "mode": "DEFAULT"}`

	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newLaunchUpdateCmd()
	require.NoError(t, cmd.Flags().Set("from-file", path))

	req, err := buildLaunchUpdateRequest(cmd)

	require.NoError(t, err)
	require.NotNil(t, req.Description)
	assert.Equal(t, "from file", *req.Description)
	require.NotNil(t, req.Mode)
	assert.Equal(t, types.UpdateLaunchRQModeDEFAULT, *req.Mode)
}

func TestLaunchUpdateCmd_FromFilePrecedence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "update.json")
	content := `{"description": "file wins"}`

	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newLaunchUpdateCmd()
	require.NoError(t, cmd.Flags().Set("from-file", path))
	require.NoError(t, cmd.Flags().Set("description", "flag value"))

	req, err := buildLaunchUpdateRequest(cmd)

	require.NoError(t, err)
	require.NotNil(t, req.Description)
	assert.Equal(t, "file wins", *req.Description)
}

func TestLaunchDeleteCmd_ExactlyOneArg(t *testing.T) {
	t.Parallel()

	cmd := newLaunchDeleteCmd()

	require.Error(t, cmd.Args(cmd, []string{}))
	require.NoError(t, cmd.Args(cmd, []string{"42"}))
	require.Error(t, cmd.Args(cmd, []string{"42", "extra"}))
}
