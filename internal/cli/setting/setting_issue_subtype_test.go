package setting //nolint:testpackage // white-box tests

import (
	"os"
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettingIssueSubtypeUpdateCmd_requiresFromFile(t *testing.T) {
	t.Parallel()

	cmd := newSettingIssueSubtypeUpdateCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "from-file")
}

func TestRunSettingIssueSubtypeCreate_requiresAllFlags(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newSettingIssueSubtypeCreateCmd())

	err := runSettingIssueSubtypeCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no input provided")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunSettingIssueSubtypeCreate_missingLongName(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newSettingIssueSubtypeCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{
		"--short-name", "CB",
		"--color", "#ff0000",
		"--type-ref", "PRODUCT_BUG",
	}))

	err := runSettingIssueSubtypeCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--long-name is required")
}

func TestRunSettingIssueSubtypeCreate_missingShortName(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newSettingIssueSubtypeCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{
		"--long-name", "Custom Bug",
		"--color", "#ff0000",
		"--type-ref", "PRODUCT_BUG",
	}))

	err := runSettingIssueSubtypeCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--short-name is required")
}

func TestRunSettingIssueSubtypeCreate_missingColor(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newSettingIssueSubtypeCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{
		"--long-name", "Custom Bug",
		"--short-name", "CB",
		"--type-ref", "PRODUCT_BUG",
	}))

	err := runSettingIssueSubtypeCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--color is required")
}

func TestRunSettingIssueSubtypeCreate_missingTypeRef(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newSettingIssueSubtypeCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{
		"--long-name", "Custom Bug",
		"--short-name", "CB",
		"--color", "#ff0000",
	}))

	err := runSettingIssueSubtypeCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--type-ref is required")
}

func TestRunSettingIssueSubtypeCreate_fromFileMissingLongName(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	path := t.TempDir() + "/issue-subtype.json"
	require.NoError(t, os.WriteFile(path, []byte(`{
		"shortName": "CB",
		"color": "#ff0000",
		"typeRef": "PRODUCT_BUG"
	}`), 0o600))

	cmd := testutil.AttachCmdToRoot(t, newSettingIssueSubtypeCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	err := runSettingIssueSubtypeCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--long-name is required")
	assert.NotContains(t, err.Error(), "when not using --from-file")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestResolveIssueSubTypeCreateRequest_fromFlags(t *testing.T) {
	t.Parallel()

	cmd := newSettingIssueSubtypeCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--long-name", "Custom Bug",
		"--short-name", "CB",
		"--color", "#ff0000",
		"--type-ref", "PRODUCT_BUG",
	}))

	request, err := resolveIssueSubTypeCreateRequest(cmd, core.NewPrinterFromCmd(cmd))
	require.NoError(t, err)
	assert.Equal(t, "Custom Bug", request.LongName)
	assert.Equal(t, "CB", request.ShortName)
	assert.Equal(t, "#ff0000", request.Color)
	assert.Equal(t, "PRODUCT_BUG", request.TypeRef)
}

func TestRunSettingIssueSubtypeUpdate_requiresFromFileBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newSettingIssueSubtypeUpdateCmd())

	err := runSettingIssueSubtypeUpdate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestParseIssueSubTypeID_valid(t *testing.T) {
	t.Parallel()

	id, err := parseIssueSubTypeID("99")
	require.NoError(t, err)
	assert.EqualValues(t, 99, id)
}

func TestParseIssueSubTypeID_invalid(t *testing.T) {
	t.Parallel()

	_, err := parseIssueSubTypeID("bad")
	require.Error(t, err)
	assert.EqualError(t, err, `invalid issue-subtype ID "bad": must be a numeric ID`)
}

func TestFlattenIssueSubTypes_sortedGroups(t *testing.T) {
	t.Parallel()

	id1 := int64(1)
	id2 := int64(2)

	subTypes := flattenIssueSubTypes(map[string][]types.IssueSubTypeResource{
		"PRODUCT_BUG":    {{Id: &id2, LongName: new("Product Bug")}},
		"TO_INVESTIGATE": {{Id: &id1, LongName: new("To Investigate")}},
	})

	require.Len(t, subTypes, 2)
	require.NotNil(t, subTypes[0].Id)
	assert.EqualValues(t, 2, *subTypes[0].Id)
	require.NotNil(t, subTypes[1].Id)
	assert.EqualValues(t, 1, *subTypes[1].Id)
}

func TestIssueSubTypeRow_populated(t *testing.T) {
	t.Parallel()

	subTypeID := int64(5)
	locator := "cb001"
	longName := "Custom Bug"
	shortName := "CB"
	color := "#ff0000"

	subType := &types.IssueSubTypeResource{
		Id:        &subTypeID,
		Locator:   &locator,
		LongName:  &longName,
		ShortName: &shortName,
		Color:     &color,
	}

	row := issueSubTypeRow(subType)

	assert.Equal(t, []string{"5", "cb001", "Custom Bug", "CB", "#ff0000"}, row)
}

func TestIssueSubTypeHeaders(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"ID", "LOCATOR", "LONG NAME", "SHORT NAME", "COLOR"},
		issueSubTypeHeaders(),
	)
}
