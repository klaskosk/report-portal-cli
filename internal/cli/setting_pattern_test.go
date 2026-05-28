package cli //nolint:testpackage // white-box tests for setting pattern CLI helpers

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunSettingPatternCreate_requiresNameTypeValue(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newSettingPatternCreateCmd())

	err := runSettingPatternCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunSettingPatternCreate_missingType(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newSettingPatternCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--name", "timeout", "--value", "timeout"}))

	err := runSettingPatternCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--type is required")
}

func TestRunSettingPatternCreate_missingValue(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newSettingPatternCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--name", "timeout", "--type", "STRING"}))

	err := runSettingPatternCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--value is required")
}

func TestResolvePatternCreateRequest_fromFlags(t *testing.T) {
	t.Parallel()

	cmd := newSettingPatternCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--name", "timeout",
		"--type", "STRING",
		"--value", "timeout error",
		"--enabled=false",
	}))

	request, err := resolvePatternCreateRequest(cmd, resolvePrinter(cmd))
	require.NoError(t, err)
	assert.Equal(t, "timeout", request.Name)
	assert.Equal(t, "STRING", request.Type)
	assert.Equal(t, "timeout error", request.Value)
	assert.False(t, request.Enabled)
}

func TestRunSettingPatternUpdate_requiresInput(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newSettingPatternUpdateCmd())

	err := runSettingPatternUpdate(cmd, []string{"42"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no input provided")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunSettingPatternUpdate_invalidID(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newSettingPatternUpdateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--name", "updated"}))

	err := runSettingPatternUpdate(cmd, []string{"abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid pattern ID")
}

func TestParsePatternID_valid(t *testing.T) {
	t.Parallel()

	id, err := parsePatternID("7")
	require.NoError(t, err)
	assert.EqualValues(t, 7, id)
}

func TestPatternRow_populated(t *testing.T) {
	t.Parallel()

	patternID := int64(3)
	name := "timeout"
	patternType := "STRING"
	value := "timeout"
	enabled := true

	pattern := &types.PatternTemplateResource{
		Id:      &patternID,
		Name:    &name,
		Type:    &patternType,
		Value:   &value,
		Enabled: &enabled,
	}

	row := patternRow(pattern)

	assert.Equal(t, []string{"3", "timeout", "STRING", "timeout", "true"}, row)
}

func TestPatternHeaders(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"ID", "NAME", "TYPE", "VALUE", "ENABLED"}, patternHeaders())
}
