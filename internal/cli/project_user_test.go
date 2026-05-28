package cli //nolint:testpackage // white-box tests for project user CLI helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProjectUserCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserCmd()

	assert.Equal(t, []string{"assign", "unassign"}, commandNames(cmd))
}

func TestResolveAssignUsersRequest_requiresUser(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserAssignCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	printer := newPrinter("")

	_, err := resolveAssignUsersRequest(cmd, printer)
	require.Error(t, err)
	assert.EqualError(t, err, "--user is required when not using --from-file")
}

func TestResolveAssignUsersRequest_invalidFormat(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserAssignCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--user", "no-colon"}))

	printer := newPrinter("")

	_, err := resolveAssignUsersRequest(cmd, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --user format")
	assert.Contains(t, err.Error(), "username:role")
}

func TestResolveAssignUsersRequest_invalidRole(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserAssignCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--user", "john:INVALID"}))

	printer := newPrinter("")

	_, err := resolveAssignUsersRequest(cmd, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")
}

func TestResolveAssignUsersRequest_validRolesCaseInsensitive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"member", "MEMBER"},
		{"Member", "MEMBER"},
		{"MEMBER", "MEMBER"},
		{"project_manager", "PROJECT_MANAGER"},
		{"Project_Manager", "PROJECT_MANAGER"},
		{"customer", "CUSTOMER"},
		{"operator", "OPERATOR"},
	}

	for _, testCase := range tests {
		t.Run(testCase.input, func(t *testing.T) {
			t.Parallel()

			cmd := newProjectUserAssignCmd()
			require.NoError(t, cmd.ParseFlags([]string{"--user", "john:" + testCase.input}))

			printer := newPrinter("")

			request, err := resolveAssignUsersRequest(cmd, printer)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, request.UserNames["john"])
		})
	}
}

func TestResolveAssignUsersRequest_duplicateUser(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserAssignCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--user", "john:MEMBER",
		"--user", "john:PROJECT_MANAGER",
	}))

	printer := newPrinter("")

	_, err := resolveAssignUsersRequest(cmd, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate --user entry")
	assert.Contains(t, err.Error(), "john")
}

func TestResolveAssignUsersRequest_fromFile_nonExistent(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserAssignCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", "/nonexistent/users.json"}))

	printer := newPrinter("")

	_, err := resolveAssignUsersRequest(cmd, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project user assign")
}

func TestResolveUnassignUsersRequest_fromFile_nonExistent(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserUnassignCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--from-file", "/nonexistent/users.json"}))

	printer := newPrinter("")

	_, err := resolveUnassignUsersRequest(cmd, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project user unassign")
}

func TestResolveAssignUsersRequest_multipleUsers(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserAssignCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--user", "john:MEMBER",
		"--user", "jane:PROJECT_MANAGER",
	}))

	printer := newPrinter("")

	request, err := resolveAssignUsersRequest(cmd, printer)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"john": "MEMBER",
		"jane": "PROJECT_MANAGER",
	}, request.UserNames)
}

func TestResolveAssignUsersRequest_fromFileTakesPrecedence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "users.json")

	require.NoError(t, os.WriteFile(path, []byte(`{
		"userNames": {
			"from_file": "CUSTOMER"
		}
	}`), 0o600))

	cmd := newProjectUserAssignCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--user", "ignored:MEMBER",
		"--from-file", path,
	}))

	printer := newPrinter("")

	request, err := resolveAssignUsersRequest(cmd, printer)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"from_file": "CUSTOMER"}, request.UserNames)
}

func TestResolveUnassignUsersRequest_requiresUser(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserUnassignCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	printer := newPrinter("")

	_, err := resolveUnassignUsersRequest(cmd, printer)
	require.Error(t, err)
	assert.EqualError(t, err, "--user is required when not using --from-file")
}

func TestResolveUnassignUsersRequest_multipleUsers(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserUnassignCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--user", "john", "--user", "jane"}))

	printer := newPrinter("")

	request, err := resolveUnassignUsersRequest(cmd, printer)
	require.NoError(t, err)
	assert.Equal(t, []string{"john", "jane"}, request.UserNames)
}

func TestResolveUnassignUsersRequest_fromFileTakesPrecedence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "users.json")

	require.NoError(t, os.WriteFile(path, []byte(`{
		"userNames": ["from_file"]
	}`), 0o600))

	cmd := newProjectUserUnassignCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--user", "ignored",
		"--from-file", path,
	}))

	printer := newPrinter("")

	request, err := resolveUnassignUsersRequest(cmd, printer)
	require.NoError(t, err)
	assert.Equal(t, []string{"from_file"}, request.UserNames)
}

func TestNewProjectUserAssignCmd_requiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserAssignCmd()

	require.Error(t, cmd.Args(cmd, []string{}), "should reject zero args")
	require.NoError(t, cmd.Args(cmd, []string{"my_project"}), "should accept exactly one arg")
	require.Error(t, cmd.Args(cmd, []string{"a", "b"}), "should reject two args")
}

func TestNewProjectUserUnassignCmd_requiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	cmd := newProjectUserUnassignCmd()

	require.Error(t, cmd.Args(cmd, []string{}), "should reject zero args")
	require.NoError(t, cmd.Args(cmd, []string{"my_project"}), "should accept exactly one arg")
	require.Error(t, cmd.Args(cmd, []string{"a", "b"}), "should reject two args")
}
