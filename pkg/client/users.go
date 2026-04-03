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
