package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsersScope_GetMyself_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 1,
			"userId": "superadmin",
			"email": "admin@example.com",
			"fullName": "Super Admin",
			"userRole": "ADMINISTRATOR",
			"assignedProjects": {
				"default_personal": {"entryType": "PERSONAL", "projectRole": "PROJECT_MANAGER"}
			}
		}`))
	}))

	user, err := c.Users().GetMyself(t.Context())
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, int64(1), user.Id)
	assert.Equal(t, "superadmin", user.UserId)
	assert.Equal(t, "admin@example.com", user.Email)
}

func TestUsersScope_GetMyself_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Users().GetMyself(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestUsersScope_GetByLogin_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 2,
			"userId": "testuser",
			"email": "test@example.com",
			"userRole": "USER"
		}`))
	}))

	user, err := c.Users().GetByLogin(t.Context(), "testuser")
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, "testuser", user.UserId)
}

func TestUsersScope_GetByLogin_FallsBackToDeprecatedEndpoint(t *testing.T) {
	t.Parallel()

	var (
		primaryCalls  int
		fallbackCalls int
	)

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		switch r.URL.Path {
		case "/api/v1/user/testuser":
			primaryCalls++

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errorCode": 4031, "message": "forbidden"}`))
		case "/api/users/testuser":
			fallbackCalls++

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 2,
				"userId": "testuser",
				"email": "test@example.com",
				"userRole": "USER"
			}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))

	user, err := c.Users().GetByLogin(t.Context(), "testuser")
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, "testuser", user.UserId)
	assert.Equal(t, 1, primaryCalls, "expected one call to preferred endpoint")
	assert.Equal(t, 1, fallbackCalls, "expected one call to deprecated fallback endpoint")
}

func TestUsersScope_GetByLogin_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User not found."}`))

	_, err := c.Users().GetByLogin(t.Context(), "nonexistent")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestUsersScope_ListProjects_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"default_personal": {"entryType": "PERSONAL", "projectRole": "PROJECT_MANAGER"},
			"my_project": {"entryType": "INTERNAL", "projectRole": "MEMBER"}
		}`))
	}))

	projects, err := c.Users().ListProjects(t.Context(), "superadmin")
	require.NoError(t, err)
	require.Len(t, projects, 2)
	assert.Contains(t, projects, "default_personal")
	assert.Contains(t, projects, "my_project")
}

func TestUsersScope_ListProjects_EmptyList(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{}`))
	}))

	projects, err := c.Users().ListProjects(t.Context(), "newuser")
	require.NoError(t, err)
	assert.Empty(t, projects)
}
