package settings_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/clienttest"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsScope_GetServerSettings_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/settings")
			testutil.AssertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{"server.analytics.all": "true"}`))
		},
	))

	settings, err := testClient.Settings().GetServerSettings(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "true", settings["server.analytics.all"])
}

func TestSettingsScope_GetServerSettings_Unauthorized(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Settings().GetServerSettings(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestSettingsScope_UpdateAnalytics_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/settings/analytics")
			testutil.AssertMethod(t, httpReq, http.MethodPost)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(
				`{"message": "Analytics settings were successfully updated."}`,
			))
		},
	))

	analyticsType := "all"
	result, err := testClient.Settings().UpdateAnalytics(t.Context(), types.AnalyticsResource{
		Type: analyticsType,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Analytics settings were successfully updated.", *result.Message)
}

func TestSettingsScope_UpdateAnalytics_Unauthorized(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Settings().UpdateAnalytics(t.Context(), types.AnalyticsResource{})
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestSettingsScope_UpdateServerSettings_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/settings")
			testutil.AssertMethod(t, httpReq, http.MethodPut)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "server.analytics.all", payload["key"])
			assert.Equal(t, "false", payload["value"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(
				`{"message": "Server settings were successfully updated."}`,
			))
		},
	))

	result, err := testClient.Settings().UpdateServerSettings(t.Context(), types.UpdateSettingsRq{
		Key:   "server.analytics.all",
		Value: "false",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Server settings were successfully updated.", *result.Message)
}

func TestSettingsScope_UpdateServerSettings_Unauthorized(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Settings().UpdateServerSettings(t.Context(), types.UpdateSettingsRq{
		Key:   "server.analytics.all",
		Value: "false",
	})
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}
