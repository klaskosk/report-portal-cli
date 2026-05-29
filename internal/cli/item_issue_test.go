package cli //nolint:testpackage // white-box tests for unexported item issue helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewItemIssueCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newItemIssueCmd()

	assert.Equal(t, []string{"link", "unlink"}, commandNames(cmd))
}

func TestBuildLinkExternalIssueRequestFromFlags_MissingItemID(t *testing.T) {
	t.Parallel()

	cmd := newItemIssueLinkCmd()
	require.NoError(
		t,
		cmd.ParseFlags([]string{"--ticket-id", "PROJ-123", "--url", "https://example.com"}),
	)

	_, err := buildLinkExternalIssueRequestFromFlags(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--item-id is required")
}

func TestBuildLinkExternalIssueRequestFromFlags_MissingTicketID(t *testing.T) {
	t.Parallel()

	cmd := newItemIssueLinkCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--item-id", "100", "--url", "https://example.com"}))

	_, err := buildLinkExternalIssueRequestFromFlags(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--ticket-id is required")
}

func TestBuildLinkExternalIssueRequestFromFlags_MissingURL(t *testing.T) {
	t.Parallel()

	cmd := newItemIssueLinkCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--item-id", "100", "--ticket-id", "PROJ-123"}))

	_, err := buildLinkExternalIssueRequestFromFlags(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--url is required")
}

func TestBuildLinkExternalIssueRequestFromFlags_Valid(t *testing.T) {
	t.Parallel()

	cmd := newItemIssueLinkCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--item-id", "100",
		"--item-id", "101",
		"--ticket-id", "PROJ-456",
		"--url", "https://jira.example.com/browse/PROJ-456",
		"--bts-url", "https://jira.example.com",
		"--bts-project", "PROJ",
	}))

	req, err := buildLinkExternalIssueRequestFromFlags(cmd)
	require.NoError(t, err)
	assert.Equal(t, []int64{100, 101}, req.TestItemIds)
	assert.Equal(t, "PROJ-456", req.Issues.TicketId)
	assert.Equal(t, "https://jira.example.com/browse/PROJ-456", req.Issues.Url)
	assert.Equal(t, "https://jira.example.com", req.Issues.BtsUrl)
	assert.Equal(t, "PROJ", req.Issues.BtsProject)
}

func TestBuildLinkExternalIssueRequest_FromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "link.json")
	content := `{"testItemIds":[100,101],"issues":{"ticketId":"PROJ-456","url":"https://example.com/t/456"}}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newItemIssueLinkCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	req, err := buildLinkExternalIssueRequest(cmd)
	require.NoError(t, err)
	assert.Equal(t, []int64{100, 101}, req.TestItemIds)
	assert.Equal(t, "PROJ-456", req.Issues.TicketId)
}

func TestBuildLinkExternalIssueRequest_FromFile_IgnoresFlags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "link.json")
	content := `{"testItemIds":[100],"issues":{"ticketId":"FROM-FILE","url":"https://example.com/from-file"}}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newItemIssueLinkCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--from-file", path,
		"--item-id", "999",
		"--ticket-id", "IGNORED",
		"--url", "https://example.com/ignored",
	}))

	req, err := buildLinkExternalIssueRequest(cmd)
	require.NoError(t, err)
	assert.Equal(t, []int64{100}, req.TestItemIds)
	assert.Equal(t, "FROM-FILE", req.Issues.TicketId)
	assert.Equal(t, "https://example.com/from-file", req.Issues.Url)
}

func TestBuildUnlinkExternalIssueRequestFromFlags_MissingItemID(t *testing.T) {
	t.Parallel()

	cmd := newItemIssueUnlinkCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--ticket-id", "PROJ-123"}))

	_, err := buildUnlinkExternalIssueRequestFromFlags(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--item-id is required")
}

func TestBuildUnlinkExternalIssueRequestFromFlags_MissingTicketID(t *testing.T) {
	t.Parallel()

	cmd := newItemIssueUnlinkCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--item-id", "100"}))

	_, err := buildUnlinkExternalIssueRequestFromFlags(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--ticket-id is required")
}

func TestBuildUnlinkExternalIssueRequestFromFlags_Valid(t *testing.T) {
	t.Parallel()

	cmd := newItemIssueUnlinkCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--item-id", "100",
		"--item-id", "101",
		"--ticket-id", "PROJ-456",
		"--ticket-id", "PROJ-789",
	}))

	req, err := buildUnlinkExternalIssueRequestFromFlags(cmd)
	require.NoError(t, err)
	assert.Equal(t, []int64{100, 101}, req.TestItemIds)
	assert.Equal(t, []string{"PROJ-456", "PROJ-789"}, req.TicketIds)
}

func TestBuildUnlinkExternalIssueRequest_FromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "unlink.json")
	content := `{"testItemIds":[100],"ticketIds":["PROJ-456"]}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newItemIssueUnlinkCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", path}))

	req, err := buildUnlinkExternalIssueRequest(cmd)
	require.NoError(t, err)
	assert.Equal(t, []int64{100}, req.TestItemIds)
	assert.Equal(t, []string{"PROJ-456"}, req.TicketIds)
}

func TestBuildUnlinkExternalIssueRequest_FromFile_IgnoresFlags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "unlink.json")
	content := `{"testItemIds":[100],"ticketIds":["FROM-FILE"]}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cmd := newItemIssueUnlinkCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--from-file", path,
		"--item-id", "999",
		"--ticket-id", "IGNORED",
	}))

	req, err := buildUnlinkExternalIssueRequest(cmd)
	require.NoError(t, err)
	assert.Equal(t, []int64{100}, req.TestItemIds)
	assert.Equal(t, []string{"FROM-FILE"}, req.TicketIds)
}

func TestOperationResultsRows(t *testing.T) {
	t.Parallel()

	msg1 := "External issue has been linked successfully."
	msg2 := "External issue has been unlinked successfully."
	results := []types.OperationCompletionRS{
		{Message: &msg1},
		{Message: &msg2},
	}

	rows := operationResultsRows(results)
	require.Len(t, rows, 2)
	assert.Equal(t, []string{msg1}, rows[0])
	assert.Equal(t, []string{msg2}, rows[1])
}
