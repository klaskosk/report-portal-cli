package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// UsersScope provides operations on users at the instance level.
type UsersScope struct {
	client *Client
}

// GetMyself returns the currently authenticated user.
func (u *UsersScope) GetMyself(ctx context.Context) (*types.UserResource, error) {
	logger := u.client.logger.With(slog.String("operation", "GetMyself"))
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.GetMyselfWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("get myself: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get myself: %w", err)
	}

	return result, nil
}

// GetByLogin returns a user by their login name via GetUserWithResponse.
func (u *UsersScope) GetByLogin(ctx context.Context, login string) (*types.UserResource, error) {
	logger := u.client.logger.With(
		slog.String("operation", "GetUserByLogin"),
		slog.String("login", login),
	)
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.GetUserWithResponse(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("get user by login: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get user by login: %w", err)
	}

	return result, nil
}

// ListProjects returns the projects accessible to the given user.
// Uses GetUserProjectsWithResponse because the non-deprecated variant requires admin privileges on some deployments.
func (u *UsersScope) ListProjects(
	ctx context.Context,
	username string,
) (map[string]types.AssignedProject, error) {
	logger := u.client.logger.With(
		slog.String("operation", "ListUserProjects"),
		slog.String("username", username),
	)
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.GetUserProjectsWithResponse(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("list user projects: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(
		response,
		response.JSONDefault,
		response.JSON200,
		response.Body,
	)
	if err != nil {
		return nil, fmt.Errorf("list user projects: %w", err)
	}

	if result == nil {
		return map[string]types.AssignedProject{}, nil
	}

	return *result, nil
}

// ChangePassword changes the currently authenticated user's password.
func (u *UsersScope) ChangePassword(ctx context.Context, request types.ChangePasswordRQ) (*types.OperationCompletionRS, error) {
	logger := u.client.logger.With(slog.String("operation", "ChangePassword"))
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.ChangePasswordWithResponse(ctx, api.ChangePasswordJSONRequestBody(request))
	if err != nil {
		return nil, fmt.Errorf("change password: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("change password: %w", err)
	}

	return result, nil
}

// RestorePassword initiates a password restore flow by sending a reset link to the given email address.
func (u *UsersScope) RestorePassword(ctx context.Context, request types.RestorePasswordRQ) (*types.OperationCompletionRS, error) {
	logger := u.client.logger.With(slog.String("operation", "RestorePassword"))
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.RestorePasswordWithResponse(ctx, api.RestorePasswordJSONRequestBody(request))
	if err != nil {
		return nil, fmt.Errorf("restore password: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("restore password: %w", err)
	}

	return result, nil
}

// IsRestorePasswordBidExist checks whether a password reset bid with the given UUID exists.
func (u *UsersScope) IsRestorePasswordBidExist(ctx context.Context, uuid string) (bool, error) {
	logger := u.client.logger.With(
		slog.String("operation", "IsRestorePasswordBidExist"),
		slog.String("uuid", uuid),
	)
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.IsRestorePasswordBidExistWithResponse(ctx, uuid)
	if err != nil {
		return false, fmt.Errorf("check restore password bid: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return false, fmt.Errorf("check restore password bid: %w", err)
	}

	if result == nil || result.Is == nil {
		return false, nil
	}

	return *result.Is, nil
}

// ResetPassword completes the password reset flow using the UUID from the restore email and the new password.
func (u *UsersScope) ResetPassword(ctx context.Context, request types.ResetPasswordRQ) (*types.OperationCompletionRS, error) {
	logger := u.client.logger.With(slog.String("operation", "ResetPassword"))
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.ResetPasswordWithResponse(ctx, api.ResetPasswordJSONRequestBody(request))
	if err != nil {
		return nil, fmt.Errorf("reset password: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("reset password: %w", err)
	}

	return result, nil
}

// ListAPIKeys returns the API keys for the given user.
func (u *UsersScope) ListAPIKeys(ctx context.Context, userID int64) ([]types.ApiKeyRS, error) {
	logger := u.client.logger.With(
		slog.String("operation", "ListAPIKeys"),
		slog.Int64("userId", userID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.GetUsersApiKeysWithResponse(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}

	if result == nil || result.Items == nil {
		return []types.ApiKeyRS{}, nil
	}

	return result.Items, nil
}

// CreateAPIKey creates a new API key for the given user. The returned ApiKeyRS
// will have the ApiKey field populated; this is the only time the raw key value
// is available.
func (u *UsersScope) CreateAPIKey(ctx context.Context, userID int64, request types.ApiKeyRQ) (*types.ApiKeyRS, error) {
	logger := u.client.logger.With(
		slog.String("operation", "CreateAPIKey"),
		slog.Int64("userId", userID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.CreateApiKeyWithResponse(ctx, userID, api.CreateApiKeyJSONRequestBody(request))
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON201, response.Body)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	return result, nil
}

// DeleteAPIKey deletes the API key with the given ID for the given user.
func (u *UsersScope) DeleteAPIKey(ctx context.Context, userID int64, keyID int64) (*types.OperationCompletionRS, error) {
	logger := u.client.logger.With(
		slog.String("operation", "DeleteAPIKey"),
		slog.Int64("userId", userID),
		slog.Int64("keyId", keyID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.DeleteApiKeyWithResponse(ctx, userID, keyID)
	if err != nil {
		return nil, fmt.Errorf("delete api key: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("delete api key: %w", err)
	}

	return result, nil
}
