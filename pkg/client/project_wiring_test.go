package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/clienttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectScope_Launches_GetByID(t *testing.T) {
	t.Parallel()

	const launchJSON = `{
		"id": 42,
		"uuid": "launch-uuid",
		"name": "Smoke",
		"status": "PASSED",
		"number": 1
	}`

	c := newTestClient(t, clienttest.JSONHandler(http.StatusOK, launchJSON))

	launch, err := c.Project("my_project").Launches().GetByID(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, launch)
	assert.Equal(t, int64(42), launch.Id)
	assert.Equal(t, "Smoke", launch.Name)
}

func TestProjectScope_Items_NotNil(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, clienttest.JSONHandler(http.StatusOK, `{"content": [], "page": {}}`))
	assert.NotNil(t, c.Project("my_project").Items())
}

func TestClient_Users_NotNil(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, clienttest.JSONHandler(http.StatusOK, `{"content": [], "page": {}}`))
	assert.NotNil(t, c.Users())
}
