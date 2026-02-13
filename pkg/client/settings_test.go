package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsScope_GetServerSettings_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/settings")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"server.analytics.all": "true"}`))
	}))

	settings, err := c.Settings().GetServerSettings(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "true", settings["server.analytics.all"])
}

func TestSettingsScope_GetServerSettings_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Settings().GetServerSettings(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestSettingsScope_UpdateAnalytics_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/settings/analytics")
		assertMethod(t, r, http.MethodPut)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"message": "Analytics settings were successfully updated."}`))
	}))

	analyticsType := "all"
	result, err := c.Settings().UpdateAnalytics(t.Context(), types.AnalyticsResource{
		Type: &analyticsType,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Analytics settings were successfully updated.", *result.Message)
}

func TestSettingsScope_UpdateAnalytics_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Settings().UpdateAnalytics(t.Context(), types.AnalyticsResource{})
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}
