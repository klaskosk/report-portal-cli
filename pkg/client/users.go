package client

import (
	"context"
	"fmt"
	"log/slog"

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

	response, err := u.client.raw.GetMyselfUsingGETWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("get myself: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "get myself", response.JSONDefault, response.JSON200)
}

// GetByLogin returns a user by their login name. It prefers the non-deprecated user controller endpoint
// (GetUserUsingGET) and falls back to the deprecated endpoint (GetUserUsingGET1) when the preferred endpoint is
// rejected due to authorization differences across deployments.
func (u *UsersScope) GetByLogin(ctx context.Context, login string) (*types.UserResource, error) {
	logger := u.client.logger.With(
		slog.String("operation", "GetUserByLogin"),
		slog.String("login", login),
	)
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.GetUserUsingGETWithResponse(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("get user by login: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, "get user by login", response.JSONDefault, response.JSON200)
	if err == nil {
		return result, nil
	}

	if !IsUnauthorized(err) && !IsForbidden(err) {
		return nil, err
	}

	logger.DebugContext(
		ctx,
		"falling back to deprecated user endpoint",
		"reason", err.Error(),
	)

	fallbackResponse, fallbackErr := u.client.raw.GetUserUsingGET1WithResponse(ctx, login)
	if fallbackErr != nil {
		return nil, fmt.Errorf("get user by login (fallback): %w", fallbackErr)
	}

	logger.DebugContext(ctx, "fallback API response", "status", fallbackResponse.StatusCode())

	return handleResponse(
		fallbackResponse,
		"get user by login (fallback)",
		fallbackResponse.JSONDefault,
		fallbackResponse.JSON200,
	)
}

// ListProjects returns the projects accessible to the given user.
// Uses the deprecated user controller endpoint (GetUserProjectsUsingGET1) because the
// non-deprecated variant requires admin privileges on some Report Portal deployments.
func (u *UsersScope) ListProjects(
	ctx context.Context,
	username string,
) (map[string]types.AssignedProject, error) {
	logger := u.client.logger.With(
		slog.String("operation", "ListUserProjects"),
		slog.String("username", username),
	)
	logger.InfoContext(ctx, "API request")

	response, err := u.client.raw.GetUserProjectsUsingGET1WithResponse(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("list user projects: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(
		response,
		"list user projects",
		response.JSONDefault,
		response.JSON200,
	)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return map[string]types.AssignedProject{}, nil
	}

	return *result, nil
}
