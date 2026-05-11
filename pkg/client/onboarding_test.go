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

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/onboarding")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{"page": "WELCOME", "completed": false}`))
		},
	))

	result, err := testClient.Onboarding().GetStatus(t.Context(), nil)
	require.NoError(t, err)

	m, ok := result.(map[string]any)
	require.True(t, ok, "expected map result")
	assert.Equal(t, "WELCOME", m["page"])
	assert.Equal(t, false, m["completed"])
}

func TestOnboardingScope_GetStatus_ArrayResponse(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/onboarding")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[{"page": "GENERAL"}]`))
		},
	))

	result, err := testClient.Onboarding().GetStatus(t.Context(), nil)
	require.NoError(t, err)

	arr, ok := result.([]any)
	require.True(t, ok, "expected array result")
	assert.Len(t, arr, 1)
}

func TestOnboardingScope_GetStatus_WithPage(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/onboarding")
			assertMethod(t, httpReq, http.MethodGet)
			assert.Equal(t, "INTEGRATIONS", httpReq.URL.Query().Get("page"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{"page": "INTEGRATIONS"}`))
		},
	))

	pageName := "INTEGRATIONS"
	result, err := testClient.Onboarding().GetStatus(t.Context(), &pageName)
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

func TestOnboardingScope_GetStatus_NonJSONError(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "text/html")
			responseWriter.WriteHeader(http.StatusBadGateway)
			_, _ = responseWriter.Write([]byte("<html>Bad Gateway</html>"))
		},
	))

	_, err := testClient.Onboarding().GetStatus(t.Context(), nil)
	require.Error(t, err)
	assert.True(t, client.HasStatusCode(err, http.StatusBadGateway))
}

func TestOnboardingScope_GetStatus_EmptyBody(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))

	result, err := c.Onboarding().GetStatus(t.Context(), nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestOnboardingScope_GetStatus_NonJSONErrorWithJSONContentType(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			_, _ = responseWriter.Write([]byte("not valid json"))
		},
	))

	_, err := testClient.Onboarding().GetStatus(t.Context(), nil)
	require.Error(t, err)
	assert.True(t, client.HasStatusCode(err, http.StatusInternalServerError))
}
