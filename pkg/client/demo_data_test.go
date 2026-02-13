package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDemoDataScope_Generate_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/demo/my_project/generate")
		assertMethod(t, r, http.MethodPost)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"dashboardId": 10,
			"launchIds": [100, 101, 102]
		}`))
	}))

	shouldCreateDashboard := true
	result, err := c.Project("my_project").DemoData().Generate(t.Context(), types.DemoDataRq{
		CreateDashboard: &shouldCreateDashboard,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, int64(10), *result.DashboardId)
	require.NotNil(t, result.LaunchIds)
	assert.Equal(t, []int64{100, 101, 102}, *result.LaunchIds)
}

func TestDemoDataScope_Generate_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").DemoData().Generate(t.Context(), types.DemoDataRq{})
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}
