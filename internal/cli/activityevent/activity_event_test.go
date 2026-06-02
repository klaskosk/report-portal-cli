package activityevent //nolint:testpackage // white-box tests

import (
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewActivityEventCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	assert.Equal(t, []string{"list"}, testutil.CommandNames(cmd))
}

func TestNewActivityEventCmd_NoProjectFlag(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	assert.Nil(t, cmd.PersistentFlags().Lookup("project"))
	assert.Nil(t, cmd.Flags().Lookup("project"))
}

func TestNewActivityEventListCmd_DefaultSort(t *testing.T) {
	t.Parallel()

	cmd := newActivityEventListCmd()

	flags, err := core.ResolveListFlags(cmd)
	require.NoError(t, err)

	assert.Equal(t, []string{"createdAt,desc"}, flags.Sort)
}

func TestBuildActivityEventSearchCriteria_allFilters(t *testing.T) {
	t.Parallel()

	cmd := newActivityEventListCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--user", "admin",
		"--object-type", "launch",
		"--event-name", "startLaunch",
		"--project-name", "my_project",
	}))

	criteria := buildActivityEventSearchCriteria(cmd)

	require.Len(t, criteria.SearchCriterias, 4)
	assert.Equal(t, client.EventFilterByUser("admin"), criteria.SearchCriterias[0])
	assert.Equal(t, client.EventFilterByObjectType("launch"), criteria.SearchCriterias[1])
	assert.Equal(t, client.EventFilterByEventName("startLaunch"), criteria.SearchCriterias[2])
	assert.Equal(t, client.EventFilterByProjectName("my_project"), criteria.SearchCriterias[3])
}

func TestBuildActivityEventSearchCriteria_noFilters(t *testing.T) {
	t.Parallel()

	cmd := newActivityEventListCmd()
	require.NoError(t, cmd.ParseFlags(nil))

	criteria := buildActivityEventSearchCriteria(cmd)

	assert.Empty(t, criteria.SearchCriterias)
}

func TestParseActivityEventSort_defaults(t *testing.T) {
	t.Parallel()

	field, order := parseActivityEventSort(nil)

	assert.Equal(t, "createdAt", field)
	assert.Equal(t, "DESC", order)
}

func TestParseActivityEventSort_explicit(t *testing.T) {
	t.Parallel()

	field, order := parseActivityEventSort([]string{"event_name,asc"})

	assert.Equal(t, "event_name", field)
	assert.Equal(t, "ASC", order)
}

func TestActivityEventRow_populated(t *testing.T) {
	t.Parallel()

	event := &types.ActivityEventResource{
		Id:          1,
		EventName:   "startLaunch",
		ObjectType:  "launch",
		ObjectName:  "My Launch",
		ProjectName: "my_project",
		SubjectName: "admin",
		CreatedAt:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	row := activityEventRow(event)

	assert.Equal(t, []string{
		"1",
		"startLaunch",
		"launch",
		"My Launch",
		"my_project",
		"admin",
		"2024-01-15 10:30:00",
	}, row)
}

func TestActivityEventHeaders(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"ID", "EVENT", "OBJECT TYPE", "OBJECT NAME", "PROJECT", "USER", "CREATED AT"},
		activityEventHeaders(),
	)
}
