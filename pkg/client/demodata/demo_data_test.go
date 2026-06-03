package demodata_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/clienttest"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDemoDataScope_Generate_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/demo/my_project/generate")
			testutil.AssertMethod(t, httpReq, http.MethodPost)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"dashboardId": 10,
			"launchIds": [100, 101, 102]
		}`))
		},
	))

	shouldCreateDashboard := true
	result, err := testClient.Project("my_project").DemoData().Generate(
		t.Context(),
		types.DemoDataRq{
			CreateDashboard: &shouldCreateDashboard,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, int64(10), *result.DashboardId)
	require.NotNil(t, result.LaunchIds)
	assert.Equal(t, []int64{100, 101, 102}, *result.LaunchIds)
}

func TestDemoDataScope_Generate_Unauthorized(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").DemoData().Generate(t.Context(), types.DemoDataRq{})
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}
