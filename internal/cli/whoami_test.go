package cli //nolint:testpackage // white-box tests for whoami CLI helpers

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWhoamiCmd_exists(t *testing.T) {
	t.Parallel()

	cmd := newWhoamiCmd()

	assert.Equal(t, "whoami", cmd.Name())
	require.NoError(t, cmd.Args(cmd, nil))
}

func TestWhoamiRows_withProjects(t *testing.T) {
	t.Parallel()

	fullName := "Alice Example"
	role := "USER"
	user := &types.UserResource{
		UserId:   "alice",
		Email:    "alice@example.com",
		FullName: &fullName,
		UserRole: &role,
	}

	rows := whoamiRows(user, []string{"beta", "alpha"})

	assert.Equal(t, []string{"FIELD", "VALUE"}, whoamiHeaders())
	assert.Equal(t, [][]string{
		{"Login", "alice"},
		{"Full Name", "Alice Example"},
		{"Email", "alice@example.com"},
		{"Role", "USER"},
		{"Projects", "beta, alpha"},
	}, rows)
}

func TestWhoamiRows_noProjects(t *testing.T) {
	t.Parallel()

	user := &types.UserResource{
		UserId: "bob",
		Email:  "bob@example.com",
	}

	rows := whoamiRows(user, nil)

	assert.Equal(t, "(none)", rows[len(rows)-1][1])
}

func TestSortedProjectNames(t *testing.T) {
	t.Parallel()

	projects := map[string]types.AssignedProject{
		"z_proj": {},
		"a_proj": {},
	}

	assert.Equal(t, []string{"a_proj", "z_proj"}, sortedProjectNames(projects))
}
