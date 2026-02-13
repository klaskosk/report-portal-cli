package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBugTrackingScope_GetSystemFields_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/bts/my_project/1/fields-set")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
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
	}))

	fields, err := c.Project("my_project").BugTracking().GetSystemFields(t.Context(), 1)
	require.NoError(t, err)
	require.Len(t, fields, 2)

	assert.Equal(t, "summary", fields[0].Id)
	assert.Equal(t, "Summary", *fields[0].FieldName)
}

func TestBugTrackingScope_GetSystemFields_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.Project("my_project").BugTracking().GetSystemFields(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestBugTrackingScope_GetAllowableIssueTypes_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/bts/my_project/1/issue_types")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`["Bug", "Task", "Story"]`))
	}))

	issueTypes, err := c.Project("my_project").BugTracking().GetAllowableIssueTypes(t.Context(), 1)
	require.NoError(t, err)
	require.Len(t, issueTypes, 3)

	assert.Equal(t, "Bug", issueTypes[0])
	assert.Equal(t, "Task", issueTypes[1])
	assert.Equal(t, "Story", issueTypes[2])
}

func TestBugTrackingScope_GetAllowableIssueTypes_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Integration '999' not found."}`))

	_, err := c.Project("my_project").BugTracking().GetAllowableIssueTypes(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestBugTrackingScope_GetTicket_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/bts/my_project/ticket/PROJ-123")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, "https://jira.example.com", query.Get("btsUrl"))
		assert.Equal(t, "PROJ", query.Get("btsProject"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": "PROJ-123",
			"summary": "Fix the bug",
			"status": "Open",
			"url": "https://jira.example.com/browse/PROJ-123"
		}`))
	}))

	ticket, err := c.Project("my_project").BugTracking().GetTicket(
		t.Context(), "PROJ-123", "https://jira.example.com", "PROJ",
	)
	require.NoError(t, err)
	require.NotNil(t, ticket)

	assert.Equal(t, "PROJ-123", *ticket.Id)
	assert.Equal(t, "Fix the bug", *ticket.Summary)
	assert.Equal(t, "Open", *ticket.Status)
	assert.Equal(t, "https://jira.example.com/browse/PROJ-123", *ticket.Url)
}

func TestBugTrackingScope_GetTicket_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Ticket 'PROJ-999' not found."}`))

	_, err := c.Project("my_project").BugTracking().GetTicket(
		t.Context(), "PROJ-999", "https://jira.example.com", "PROJ",
	)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
