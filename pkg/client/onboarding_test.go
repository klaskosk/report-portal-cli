package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnboardingScope_GetStatus_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/onboarding")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"page": "WELCOME", "completed": false}`))
	}))

	result, err := c.Onboarding().GetStatus(t.Context(), nil)
	require.NoError(t, err)

	m, ok := result.(map[string]any)
	require.True(t, ok, "expected map result")
	assert.Equal(t, "WELCOME", m["page"])
	assert.Equal(t, false, m["completed"])
}

func TestOnboardingScope_GetStatus_ArrayResponse(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/onboarding")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[{"page": "GENERAL"}]`))
	}))

	result, err := c.Onboarding().GetStatus(t.Context(), nil)
	require.NoError(t, err)

	arr, ok := result.([]any)
	require.True(t, ok, "expected array result")
	assert.Len(t, arr, 1)
}

func TestOnboardingScope_GetStatus_WithPage(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/onboarding")
		assertMethod(t, r, http.MethodGet)
		assert.Equal(t, "INTEGRATIONS", r.URL.Query().Get("page"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"page": "INTEGRATIONS"}`))
	}))

	pageName := "INTEGRATIONS"
	result, err := c.Onboarding().GetStatus(t.Context(), &pageName)
	require.NoError(t, err)

	m, ok := result.(map[string]any)
	require.True(t, ok, "expected map result")
	assert.Equal(t, "INTEGRATIONS", m["page"])
}

func TestOnboardingScope_GetStatus_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Onboarding().GetStatus(t.Context(), nil)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}
