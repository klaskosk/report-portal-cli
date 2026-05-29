package group //nolint:testpackage // white-box tests

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroupCmd_AssociationSubcommands(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	assert.Equal(
		t,
		[]string{"create", "delete", "get", "list", "project", "update", "user"},
		testutil.CommandNames(cmd),
	)
}

func TestNewGroupUserCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newGroupUserCmd()

	assert.Equal(t, []string{"add", "get", "list", "remove"}, testutil.CommandNames(cmd))
}

func TestNewGroupProjectCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newGroupProjectCmd()

	assert.Equal(t, []string{"add", "get", "list", "remove"}, testutil.CommandNames(cmd))
}

func TestGroupAssociationListCmd_ExactArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "user list", cmd: newGroupUserListCmd()},
		{name: "project list", cmd: newGroupProjectListCmd()},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, testCase.cmd.Args(testCase.cmd, []string{"42"}))
			require.Error(t, testCase.cmd.Args(testCase.cmd, nil))
			require.Error(t, testCase.cmd.Args(testCase.cmd, []string{"42", "7"}))
		})
	}
}

func TestGroupAssociationGetAddRemoveCmd_ExactArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "user get", cmd: newGroupUserGetCmd()},
		{name: "user add", cmd: newGroupUserAddCmd()},
		{name: "user remove", cmd: newGroupUserRemoveCmd()},
		{name: "project get", cmd: newGroupProjectGetCmd()},
		{name: "project add", cmd: newGroupProjectAddCmd()},
		{name: "project remove", cmd: newGroupProjectRemoveCmd()},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, testCase.cmd.Args(testCase.cmd, []string{"42", "7"}))
			require.Error(t, testCase.cmd.Args(testCase.cmd, []string{"42"}))
			require.Error(t, testCase.cmd.Args(testCase.cmd, []string{"42", "7", "8"}))
		})
	}
}

func TestGroupProjectAddCmd_permissionFlagOptional(t *testing.T) {
	t.Parallel()

	cmd := newGroupProjectAddCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	permission := core.OptionalStringFlag(cmd, "permission")
	assert.Nil(t, permission)

	require.NoError(t, cmd.ParseFlags([]string{"--permission", "MEMBER"}))

	permission = core.OptionalStringFlag(cmd, "permission")
	require.NotNil(t, permission)
	assert.Equal(t, "MEMBER", *permission)
}

func TestParseGroupID_invalidAssociation(t *testing.T) {
	t.Parallel()

	_, err := parseGroupID("not-a-number")
	require.Error(t, err)
	assert.EqualError(t, err, `invalid group ID "not-a-number": must be a numeric ID`)
}

func TestParseUserID_valid(t *testing.T) {
	t.Parallel()

	id, err := parseUserID("7")
	require.NoError(t, err)
	assert.EqualValues(t, 7, id)
}

func TestParseUserID_invalid(t *testing.T) {
	t.Parallel()

	_, err := parseUserID("not-a-number")
	require.Error(t, err)
	assert.EqualError(t, err, `invalid user ID "not-a-number": must be a numeric ID`)
}

func TestParseProjectID_valid(t *testing.T) {
	t.Parallel()

	id, err := parseProjectID("3")
	require.NoError(t, err)
	assert.EqualValues(t, 3, id)
}

func TestParseProjectID_invalid(t *testing.T) {
	t.Parallel()

	_, err := parseProjectID("not-a-number")
	require.Error(t, err)
	assert.EqualError(t, err, `invalid project ID "not-a-number": must be a numeric ID`)
}

func TestParseGroupUserIDs_valid(t *testing.T) {
	t.Parallel()

	groupID, userID, err := parseGroupUserIDs([]string{"42", "7"})
	require.NoError(t, err)
	assert.EqualValues(t, 42, groupID)
	assert.EqualValues(t, 7, userID)
}

func TestParseGroupUserIDs_invalidUserID(t *testing.T) {
	t.Parallel()

	_, _, err := parseGroupUserIDs([]string{"42", "bad"})
	require.Error(t, err)
	assert.EqualError(t, err, `invalid user ID "bad": must be a numeric ID`)
}

func TestParseGroupProjectIDs_valid(t *testing.T) {
	t.Parallel()

	groupID, projectID, err := parseGroupProjectIDs([]string{"42", "3"})
	require.NoError(t, err)
	assert.EqualValues(t, 42, groupID)
	assert.EqualValues(t, 3, projectID)
}

func TestParseGroupProjectIDs_invalidProjectID(t *testing.T) {
	t.Parallel()

	_, _, err := parseGroupProjectIDs([]string{"42", "bad"})
	require.Error(t, err)
	assert.EqualError(t, err, `invalid project ID "bad": must be a numeric ID`)
}

func TestResolveGroupAssociationPagination_defaults(t *testing.T) {
	t.Parallel()

	cmd := newGroupUserListCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	pageSize, offset, err := resolveGroupAssociationPagination(cmd)
	require.NoError(t, err)
	assert.Equal(t, int32(10), pageSize)
	assert.Equal(t, int32(0), offset)
}

func TestResolveGroupAssociationPagination_customValues(t *testing.T) {
	t.Parallel()

	cmd := newGroupProjectListCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--page-size", "25", "--offset", "50"}))

	pageSize, offset, err := resolveGroupAssociationPagination(cmd)
	require.NoError(t, err)
	assert.Equal(t, int32(25), pageSize)
	assert.Equal(t, int32(50), offset)
}

func TestResolveGroupAssociationPagination_invalidPageSize(t *testing.T) {
	t.Parallel()

	cmd := newGroupUserListCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--page-size", "0"}))

	_, _, err := resolveGroupAssociationPagination(cmd)
	require.Error(t, err)
	assert.EqualError(t, err, "page-size must be greater than 0")
}

func TestResolveGroupAssociationPagination_invalidOffset(t *testing.T) {
	t.Parallel()

	cmd := newGroupUserListCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--offset", "-1"}))

	_, _, err := resolveGroupAssociationPagination(cmd)
	require.Error(t, err)
	assert.EqualError(t, err, "offset must be greater than or equal to 0")
}
