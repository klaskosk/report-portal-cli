package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstanceBugTrackingScope_GetSystemFields_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/bts/1/fields-set")
			assertMethod(t, httpReq, http.MethodGet)

			assert.Equal(t, "Bug", httpReq.URL.Query().Get("issueType"))

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[
			{
				"id": "summary",
				"fieldName": "Summary",
				"fieldType": "string",
				"required": true
			},
			{
				"id": "description",
				"fieldName": "Description",
				"fieldType": "string",
				"required": false
			}
		]`))
		},
	))

	fields, err := testClient.InstanceBugTracking().GetSystemFields(t.Context(), 1, "Bug")
	require.NoError(t, err)
	require.Len(t, fields, 2)

	assert.Equal(t, "summary", fields[0].Id)
	assert.Equal(t, "Summary", *fields[0].FieldName)
	assert.Equal(t, "description", fields[1].Id)
	assert.Equal(t, "Description", *fields[1].FieldName)
}

func TestInstanceBugTrackingScope_GetSystemFields_Empty(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[]`))
	}))

	fields, err := testClient.InstanceBugTracking().GetSystemFields(t.Context(), 1, "Bug")
	require.NoError(t, err)
	assert.Empty(t, fields)
}

func TestInstanceBugTrackingScope_GetSystemFields_Null(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`null`))
		},
	))

	fields, err := testClient.InstanceBugTracking().GetSystemFields(t.Context(), 1, "Bug")
	require.NoError(t, err)
	assert.Empty(t, fields)
	assert.NotNil(t, fields)
}

func TestInstanceBugTrackingScope_GetSystemFields_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.InstanceBugTracking().GetSystemFields(t.Context(), 999, "Bug")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestInstanceBugTrackingScope_GetSystemFields_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.InstanceBugTracking().GetSystemFields(t.Context(), 1, "Bug")
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestInstanceBugTrackingScope_GetAllowableIssueTypes_Success(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			assertBearerToken(t, httpReq)
			assertPath(t, httpReq, "/api/v1/bts/1/issue_types")
			assertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`["Bug", "Task", "Story"]`))
		},
	))

	issueTypes, err := testClient.InstanceBugTracking().GetAllowableIssueTypes(t.Context(), 1)
	require.NoError(t, err)
	require.Len(t, issueTypes, 3)

	assert.Equal(t, "Bug", issueTypes[0])
	assert.Equal(t, "Task", issueTypes[1])
	assert.Equal(t, "Story", issueTypes[2])
}

func TestInstanceBugTrackingScope_GetAllowableIssueTypes_Empty(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[]`))
		},
	))

	issueTypes, err := testClient.InstanceBugTracking().GetAllowableIssueTypes(t.Context(), 1)
	require.NoError(t, err)
	assert.Empty(t, issueTypes)
}

func TestInstanceBugTrackingScope_GetAllowableIssueTypes_Null(t *testing.T) {
	t.Parallel()

	testClient := newTestClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, _ *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`null`))
		},
	))

	issueTypes, err := testClient.InstanceBugTracking().GetAllowableIssueTypes(t.Context(), 1)
	require.NoError(t, err)
	assert.Empty(t, issueTypes)
	assert.NotNil(t, issueTypes)
}

func TestInstanceBugTrackingScope_GetAllowableIssueTypes_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.InstanceBugTracking().GetAllowableIssueTypes(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestInstanceBugTrackingScope_GetAllowableIssueTypes_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.InstanceBugTracking().GetAllowableIssueTypes(t.Context(), 1)
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}
