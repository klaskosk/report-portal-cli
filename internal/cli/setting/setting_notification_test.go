package setting //nolint:testpackage // white-box tests

import (
	"os"
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettingNotificationCreateCmd_requiresFromFile(t *testing.T) {
	t.Parallel()

	cmd := newSettingNotificationCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "from-file")
}

func TestNewSettingNotificationUpdateCmd_requiresFromFile(t *testing.T) {
	t.Parallel()

	cmd := newSettingNotificationUpdateCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "from-file")
}

func TestRunSettingNotificationCreate_requiresFromFileBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newSettingNotificationCreateCmd())

	err := runSettingNotificationCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunSettingNotificationUpdate_fromFileMissingID(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	path := t.TempDir() + "/notification.json"
	require.NoError(t, os.WriteFile(path, []byte(`{
		"ruleName": "rule-1",
		"sendCase": "FAILED",
		"enabled": true
	}`), 0o600))

	cmd := testutil.AttachCmdToRoot(t, newSettingNotificationUpdateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	err := runSettingNotificationUpdate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification rule ID must be set in --from-file payload")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunSettingNotificationUpdate_requiresFromFileBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newSettingNotificationUpdateCmd())

	err := runSettingNotificationUpdate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestParseNotificationID_valid(t *testing.T) {
	t.Parallel()

	id, err := parseNotificationID("42")
	require.NoError(t, err)
	assert.EqualValues(t, 42, id)
}

func TestParseNotificationID_invalid(t *testing.T) {
	t.Parallel()

	_, err := parseNotificationID("abc")
	require.Error(t, err)
	assert.EqualError(t, err, `invalid notification ID "abc": must be a numeric ID`)
}

func TestNotificationRow_populated(t *testing.T) {
	t.Parallel()

	id := int64(1)
	enabled := true
	recipients := []string{"a@example.com", "b@example.com"}

	notification := &types.SenderCaseDTO{
		Id:         &id,
		RuleName:   "rule-1",
		SendCase:   types.Failed,
		Recipients: &recipients,
		Enabled:    &enabled,
	}

	row := notificationRow(notification)

	assert.Equal(
		t,
		[]string{"1", "rule-1", "failed", "a@example.com, b@example.com", "true"},
		row,
	)
}

func TestNotificationHeaders(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"ID", "RULE NAME", "SEND CASE", "RECIPIENTS", "ENABLED"},
		notificationHeaders(),
	)
}

func TestFormatStringSlice_nil(t *testing.T) {
	t.Parallel()

	assert.Empty(t, formatStringSlice(nil))
}

func TestFormatOptionalBool(t *testing.T) {
	t.Parallel()

	trueVal := true
	falseVal := false

	assert.Equal(t, "true", formatOptionalBool(&trueVal))
	assert.Equal(t, "false", formatOptionalBool(&falseVal))
	assert.Empty(t, formatOptionalBool(nil))
}
