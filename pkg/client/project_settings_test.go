package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectSettingsScope_Get_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/settings")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"project": 1,
			"subTypes": {
				"TO_INVESTIGATE": [
					{
						"id": 1,
						"locator": "ti001",
						"typeRef": "TO_INVESTIGATE",
						"longName": "To Investigate",
						"shortName": "TI",
						"color": "#ffb743"
					}
				],
				"PRODUCT_BUG": [
					{
						"id": 2,
						"locator": "pb001",
						"typeRef": "PRODUCT_BUG",
						"longName": "Product Bug",
						"shortName": "PB",
						"color": "#ec3900"
					}
				]
			}
		}`))
	}))

	settings, err := c.Project("my_project").Settings().Get(t.Context())
	require.NoError(t, err)
	require.NotNil(t, settings)

	assert.Equal(t, int64(1), settings.Project)
	assert.Contains(t, settings.SubTypes, "TO_INVESTIGATE")
	assert.Contains(t, settings.SubTypes, "PRODUCT_BUG")

	tiSubTypes := settings.SubTypes["TO_INVESTIGATE"]
	require.Len(t, tiSubTypes, 1)
	require.NotNil(t, tiSubTypes[0].Id)
	assert.Equal(t, int64(1), *tiSubTypes[0].Id)
	require.NotNil(t, tiSubTypes[0].Locator)
	assert.Equal(t, "ti001", *tiSubTypes[0].Locator)
	require.NotNil(t, tiSubTypes[0].LongName)
	assert.Equal(t, "To Investigate", *tiSubTypes[0].LongName)
	require.NotNil(t, tiSubTypes[0].ShortName)
	assert.Equal(t, "TI", *tiSubTypes[0].ShortName)
}

func TestProjectSettingsScope_Get_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    int
		body      string
		assertErr func(error) bool
	}{
		{
			name:      "Unauthorized",
			status:    http.StatusUnauthorized,
			body:      `{"errorCode": 4001, "message": "Full authentication is required."}`,
			assertErr: client.IsUnauthorized,
		},
		{
			name:      "NotFound",
			status:    http.StatusNotFound,
			body:      `{"errorCode": 40410, "message": "Project 'unknown' not found."}`,
			assertErr: client.IsNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(t, jsonResponse(tt.status, tt.body))

			_, err := c.Project("my_project").Settings().Get(t.Context())
			require.Error(t, err)
			assert.True(t, tt.assertErr(err))
		})
	}
}

func TestProjectSettingsScope_GetNotifications_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/settings/notification")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
			{
				"id": 10,
				"ruleName": "Launch finished",
				"sendCase": "ALWAYS",
				"enabled": true,
				"recipients": ["admin@example.com"]
			}
		]`))
	}))

	notifications, err := c.Project("my_project").Settings().GetNotifications(t.Context())
	require.NoError(t, err)
	require.Len(t, notifications, 1)

	assert.Equal(t, int64(10), *notifications[0].Id)
	assert.Equal(t, "Launch finished", *notifications[0].RuleName)
}

func TestProjectSettingsScope_GetNotifications_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Settings().GetNotifications(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}
