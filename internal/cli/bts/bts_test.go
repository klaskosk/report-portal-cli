package bts //nolint:testpackage // white-box tests

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBtsCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	assert.Equal(t, []string{"field", "issue", "ticket"}, testutil.CommandNames(cmd))
}

func TestNewBtsCmd_ProjectFlagInherited(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	for _, sub := range cmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			t.Parallel()

			flag := sub.InheritedFlags().Lookup("project")
			require.NotNil(t, flag, "subcommand %q should inherit --project", sub.Name())
			assert.Equal(t, "p", flag.Shorthand)
		})
	}
}

func TestNewBtsTicketGetCmd_requiredFlags(t *testing.T) {
	t.Parallel()

	cmd := newBtsTicketGetCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bts-url")
}

func TestNewBtsFieldListCmd_integrationRequired(t *testing.T) {
	t.Parallel()

	cmd := newBtsFieldListCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "integration")
}

func TestNewBtsIssueCreateCmd_requiredFlags(t *testing.T) {
	t.Parallel()

	cmd := newBtsIssueCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "integration")
	assert.Contains(t, err.Error(), "from-file")
}

func TestRunBtsIssueCreate_requiresFromFileBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newBtsIssueCreateCmd())
	require.NoError(t, cmd.ParseFlags([]string{"--integration", "1"}))

	err := runBtsIssueCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestBtsTicketRow_populated(t *testing.T) {
	t.Parallel()

	ticketID := "PROJ-123"
	summary := "Fix bug"
	status := "Open"
	url := "https://jira.example.com/browse/PROJ-123"
	plugin := "jira"

	ticket := &types.Ticket{
		Id:         &ticketID,
		Summary:    &summary,
		Status:     &status,
		Url:        &url,
		PluginName: &plugin,
	}

	headers, row := btsTicketRow(ticket)

	assert.Equal(t, btsTicketHeaders(), headers)
	assert.Equal(
		t,
		[]string{"PROJ-123", "Fix bug", "Open", "https://jira.example.com/browse/PROJ-123", "jira"},
		row,
	)
}

func TestBtsFieldRow_populated(t *testing.T) {
	t.Parallel()

	name := "Summary"
	fieldType := "string"
	description := "Issue summary"

	field := &types.PostFormField{
		Id:          "summary",
		FieldName:   &name,
		FieldType:   &fieldType,
		Required:    true,
		Description: &description,
	}

	row := btsFieldRow(field)

	assert.Equal(t, []string{"summary", "Summary", "string", "true", "Issue summary"}, row)
}

func TestBtsFieldHeaders(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"ID", "NAME", "TYPE", "REQUIRED", "DESCRIPTION"}, btsFieldHeaders())
}
