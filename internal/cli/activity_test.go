package cli //nolint:testpackage // white-box tests for activity CLI helpers

import (
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewActivityCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newActivityCmd()

	assert.Equal(t, []string{"get", "list"}, commandNames(cmd))
}

func TestNewActivityCmd_ProjectFlagInheritedBySubcommands(t *testing.T) {
	t.Parallel()

	cmd := newActivityCmd()

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

func TestNewActivityGetCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	cmd := newActivityGetCmd()

	require.Error(t, cmd.Args(cmd, []string{}), "should reject zero args")
	require.NoError(t, cmd.Args(cmd, []string{"123"}), "should accept exactly one arg")
	require.Error(t, cmd.Args(cmd, []string{"123", "456"}), "should reject two args")
}

func TestNewActivityListCmd_RequiresItemFlag(t *testing.T) {
	t.Parallel()

	cmd := newActivityListCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := cmd.ValidateRequiredFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "item")
}

func TestNewActivityListCmd_PaginationFlags(t *testing.T) {
	t.Parallel()

	cmd := newActivityListCmd()

	require.NotNil(t, cmd.Flags().Lookup("page"))
	require.NotNil(t, cmd.Flags().Lookup("page-size"))
	require.NotNil(t, cmd.Flags().Lookup("sort"))
}

func TestNewActivityListCmd_DefaultSort(t *testing.T) {
	t.Parallel()

	cmd := newActivityListCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--item", "456"}))

	flags, err := resolveListFlags(cmd)
	require.NoError(t, err)

	assert.Equal(t, []string{"lastModified,desc"}, flags.sort)
}

func TestActivityRow_populated(t *testing.T) {
	t.Parallel()

	activity := &types.ActivityResource{
		Id:           123,
		User:         "admin",
		ObjectType:   "testItem",
		ActionType:   "updateItem",
		LastModified: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	row := activityRow(activity)

	assert.Equal(t, []string{
		"123",
		"admin",
		"testItem",
		"updateItem",
		"2024-01-15 10:30:00",
	}, row)
}

func TestActivityListRow_populated(t *testing.T) {
	t.Parallel()

	event := &types.ActivityEventResource{
		Id:          789,
		SubjectName: "admin",
		ObjectType:  "testItem",
		EventName:   "updateItem",
		CreatedAt:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	row := activityListRow(event)

	assert.Equal(t, []string{
		"789",
		"admin",
		"testItem",
		"updateItem",
		"2024-01-15 10:30:00",
	}, row)
}

func TestActivityHeaders(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"ID", "USER", "OBJECT TYPE", "ACTION TYPE", "CREATED AT"},
		activityHeaders(),
	)
}

func TestParseActivityID_valid(t *testing.T) {
	t.Parallel()

	id, err := parseActivityID("42")
	require.NoError(t, err)
	assert.EqualValues(t, 42, id)
}

func TestParseActivityID_invalid(t *testing.T) {
	t.Parallel()

	_, err := parseActivityID("abc")
	require.Error(t, err)
	assert.EqualError(t, err, `invalid activity ID "abc": must be a numeric ID`)
}
