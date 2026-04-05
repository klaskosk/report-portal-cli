package client_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
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

func TestUsersScope_GetByLogin_Forbidden(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/testuser")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errorCode": 4031, "message": "forbidden"}`))
	}))

	_, err := c.Users().GetByLogin(t.Context(), "testuser")
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
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

// --- Password management tests ---

func TestUsersScope_ChangePassword_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/users/password/change")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "oldPass123", payload["oldPassword"])
		assert.Equal(t, "newPass456", payload["newPassword"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Password has been changed successfully."}`))
	}))

	newPass := "newPass456"
	result, err := c.Users().ChangePassword(t.Context(), types.ChangePasswordRQ{
		OldPassword: "oldPass123",
		NewPassword: &newPass,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, *result.Message, "successfully")
}

func TestUsersScope_ChangePassword_BadRequest(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 40010, "message": "Old password is incorrect."}`))

	newPass := "newPass456"
	_, err := c.Users().ChangePassword(t.Context(), types.ChangePasswordRQ{
		OldPassword: "wrongOldPass",
		NewPassword: &newPass,
	})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestUsersScope_RestorePassword_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/users/password/restore")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "user@example.com", payload["email"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Password restore instruction has been sent."}`))
	}))

	result, err := c.Users().RestorePassword(t.Context(), types.RestorePasswordRQ{
		Email: "user@example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, *result.Message, "restore")
}

func TestUsersScope_RestorePassword_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User with email not found."}`))

	_, err := c.Users().RestorePassword(t.Context(), types.RestorePasswordRQ{
		Email: "nonexistent@example.com",
	})
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestUsersScope_IsRestorePasswordBidExist_Exists(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/password/reset/abc-123-def")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"is": true}`))
	}))

	exists, err := c.Users().IsRestorePasswordBidExist(t.Context(), "abc-123-def")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUsersScope_IsRestorePasswordBidExist_DoesNotExist(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"is": false}`))
	}))

	exists, err := c.Users().IsRestorePasswordBidExist(t.Context(), "invalid-uuid")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestUsersScope_IsRestorePasswordBidExist_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Restore password bid not found."}`))

	_, err := c.Users().IsRestorePasswordBidExist(t.Context(), "invalid-uuid")
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestUsersScope_ResetPassword_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/api/users/password/reset")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "newSecurePass", payload["password"])
		assert.Equal(t, "abc-123-def", payload["uuid"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Password has been reset successfully."}`))
	}))

	result, err := c.Users().ResetPassword(t.Context(), types.ResetPasswordRQ{
		Password: "newSecurePass",
		Uuid:     "abc-123-def",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, *result.Message, "reset")
}

func TestUsersScope_ResetPassword_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 40010, "message": "Password reset bid has expired."}`))

	_, err := c.Users().ResetPassword(t.Context(), types.ResetPasswordRQ{
		Password: "newPass",
		Uuid:     "expired-uuid",
	})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

// --- API key management tests ---

func TestUsersScope_ListAPIKeys_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/42/api-keys")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"items": [
				{"id": 1, "name": "ci-key", "created_at": "2025-01-15T10:00:00Z", "user_id": 42},
				{"id": 2, "name": "dev-key", "created_at": "2025-02-20T14:30:00Z", "last_used_at": "2025-03-01T09:00:00Z", "user_id": 42}
			]
		}`))
	}))

	keys, err := c.Users().ListAPIKeys(t.Context(), 42)
	require.NoError(t, err)
	require.Len(t, keys, 2)

	assert.Equal(t, int64(1), keys[0].Id)
	assert.Equal(t, "ci-key", keys[0].Name)
	assert.Equal(t, int64(42), keys[0].UserId)

	assert.Equal(t, int64(2), keys[1].Id)
	assert.Equal(t, "dev-key", keys[1].Name)
	assert.NotNil(t, keys[1].LastUsedAt)
}

func TestUsersScope_ListAPIKeys_Empty(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/users/42/api-keys")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": []}`))
	}))

	keys, err := c.Users().ListAPIKeys(t.Context(), 42)
	require.NoError(t, err)
	assert.Empty(t, keys)
	assert.NotNil(t, keys)
}

func TestUsersScope_ListAPIKeys_NullItems(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))

	keys, err := c.Users().ListAPIKeys(t.Context(), 42)
	require.NoError(t, err)
	assert.Empty(t, keys)
	assert.NotNil(t, keys)
}

func TestUsersScope_ListAPIKeys_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusForbidden,
		`{"errorCode": 4031, "message": "forbidden"}`))

	_, err := c.Users().ListAPIKeys(t.Context(), 42)
	require.Error(t, err)
	assert.True(t, client.IsForbidden(err))
}

func TestUsersScope_CreateAPIKey_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, fmt.Sprintf("/api/users/%d/api-keys", 42))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "my-new-key", payload["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": 99,
			"name": "my-new-key",
			"created_at": "2025-06-01T12:00:00Z",
			"user_id": 42,
			"api_key": "rp_ultra_secret_key_value"
		}`))
	}))

	result, err := c.Users().CreateAPIKey(t.Context(), 42, types.ApiKeyRQ{Name: "my-new-key"})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, int64(99), result.Id)
	assert.Equal(t, "my-new-key", result.Name)
	assert.Equal(t, int64(42), result.UserId)
	require.NotNil(t, result.ApiKey)
	assert.Equal(t, "rp_ultra_secret_key_value", *result.ApiKey)
}

func TestUsersScope_CreateAPIKey_Error(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 40010, "message": "API key name already exists."}`))

	_, err := c.Users().CreateAPIKey(t.Context(), 42, types.ApiKeyRQ{Name: "duplicate-name"})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestUsersScope_DeleteAPIKey_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/api/users/42/api-keys/99")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "API key has been deleted."}`))
	}))

	result, err := c.Users().DeleteAPIKey(t.Context(), 42, 99)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, *result.Message, "deleted")
}

func TestUsersScope_DeleteAPIKey_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "API key not found."}`))

	_, err := c.Users().DeleteAPIKey(t.Context(), 42, 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
