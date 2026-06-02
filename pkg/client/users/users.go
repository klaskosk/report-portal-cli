package users

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// UsersScope provides operations on users at the instance level.
type UsersScope struct {
	scope core.Scope
}

// GetMyself returns the currently authenticated user.
func (u *UsersScope) GetMyself(ctx context.Context) (*types.UserResource, error) {
	logger := u.scope.Logger.With(slog.String("operation", "GetMyself"))
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.GetMyselfWithResponse(ctx)

	return core.Execute(ctx, logger, "get myself", resp, err)
}

// GetByLogin returns a user by their login name via GetUserWithResponse.
func (u *UsersScope) GetByLogin(ctx context.Context, login string) (*types.UserResource, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "GetUserByLogin"),
		slog.String("login", login),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.GetUserWithResponse(ctx, login)

	return core.Execute(ctx, logger, "get user by login", resp, err)
}

// ListProjects returns the projects accessible to the given user.
// Uses GetUserProjectsWithResponse because the non-deprecated variant requires admin privileges on some deployments.
func (u *UsersScope) ListProjects(
	ctx context.Context,
	username string,
) (map[string]types.AssignedProject, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "ListUserProjects"),
		slog.String("username", username),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.GetUserProjectsWithResponse(ctx, username)

	result, err := core.Execute(ctx, logger, "list user projects", resp, err)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return map[string]types.AssignedProject{}, nil
	}

	return *result, nil
}

// List returns a paginated list of all users (admin endpoint).
func (u *UsersScope) List(
	ctx context.Context,
	options ...ListUsersOption,
) (*types.PageUserResource, error) {
	logger := u.scope.Logger.With(slog.String("operation", "ListUsers"))

	params := &api.GetUsersParams{}
	for _, option := range options {
		option.applyListUsers(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.GetUsersWithResponse(ctx, params)

	return core.Execute(ctx, logger, "list users", resp, err)
}

// Search searches for users by a term with optional pagination.
func (u *UsersScope) Search(
	ctx context.Context,
	term string,
	options ...SearchUsersOption,
) (*types.PageUserResource, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "SearchUsers"),
		slog.String("term", term),
	)

	params := &api.FindUsersParams{Term: term}
	for _, option := range options {
		option.applySearchUsers(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.FindUsersWithResponse(ctx, params)

	return core.Execute(ctx, logger, "search users", resp, err)
}

// Export returns the user list as raw bytes (typically CSV).
func (u *UsersScope) Export(ctx context.Context, options ...ListUsersOption) ([]byte, error) {
	logger := u.scope.Logger.With(slog.String("operation", "ExportUsers"))

	listParams := &api.GetUsersParams{}
	for _, option := range options {
		option.applyListUsers(listParams)
	}

	params := listUsersParamsToExportParams(listParams)

	logger.InfoContext(ctx, "API request")

	response, err := u.scope.Client.ExportWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("export users: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.JSONDefault != nil {
		return nil, fmt.Errorf(
			"export users: %w",
			core.NewAPIError(response.StatusCode(), response.JSONDefault),
		)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf(
			"export users: %w",
			core.NewUnexpectedStatusError(code, response.Body),
		)
	}

	return response.Body, nil
}

// Create creates a new user (admin only).
func (u *UsersScope) Create(
	ctx context.Context,
	request types.CreateUserRQFull,
) (*types.CreateUserRS, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "CreateUser"),
		slog.String("login", request.Login),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.CreateUserByAdminWithResponse(ctx, request)

	return core.Execute(ctx, logger, "create user", resp, err)
}

// Edit updates an existing user by login.
func (u *UsersScope) Edit(
	ctx context.Context,
	login string,
	request types.EditUserRQ,
) (*types.OperationCompletionRS, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "EditUser"),
		slog.String("login", login),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.EditUserWithResponse(ctx, login, request)

	return core.Execute(ctx, logger, "edit user", resp, err)
}

// Delete deletes a user by numeric ID.
func (u *UsersScope) Delete(
	ctx context.Context,
	userID int64,
) (
	*types.OperationCompletionRS,
	error,
) {
	logger := u.scope.Logger.With(
		slog.String("operation", "DeleteUser"),
		slog.Int64("id", userID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.DeleteUserWithResponse(ctx, userID)

	return core.Execute(ctx, logger, "delete user", resp, err)
}

// DeleteBulk deletes multiple users by their numeric IDs.
func (u *UsersScope) DeleteBulk(ctx context.Context, ids []int64) (*types.DeleteBulkRS, error) {
	logger := u.scope.Logger.With(slog.String("operation", "DeleteBulkUsers"))
	logger.InfoContext(ctx, "API request")

	params := &api.DeleteUsersParams{Ids: ids}

	resp, err := u.scope.Client.DeleteUsersWithResponse(ctx, params)

	return core.Execute(ctx, logger, "delete users", resp, err)
}

// CreateInvitation creates an invitation bid for a new user.
func (u *UsersScope) CreateInvitation(
	ctx context.Context,
	request types.CreateUserRQ,
) (*types.CreateUserBidRS, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "CreateInvitation"),
		slog.String("email", request.Email),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.CreateUserBidWithResponse(ctx, request)

	return core.Execute(ctx, logger, "create invitation", resp, err)
}

// GetInvitation retrieves invitation bid info by UUID.
func (u *UsersScope) GetInvitation(ctx context.Context, uuid string) (*types.UserBidRS, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "GetInvitation"),
		slog.String("uuid", uuid),
	)
	logger.InfoContext(ctx, "API request")

	params := &api.GetUserBidInfoParams{Uuid: uuid}

	resp, err := u.scope.Client.GetUserBidInfoWithResponse(ctx, params)

	return core.Execute(ctx, logger, "get invitation", resp, err)
}

// Register activates an invitation and creates the user.
func (u *UsersScope) Register(
	ctx context.Context,
	uuid string,
	request types.CreateUserRQConfirm,
) (*types.CreateUserRS, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "RegisterUser"),
		slog.String("uuid", uuid),
		slog.String("login", request.Login),
	)
	logger.InfoContext(ctx, "API request")

	params := &api.CreateUserParams{Uuid: uuid}

	resp, err := u.scope.Client.CreateUserWithResponse(ctx, params, request)

	return core.Execute(ctx, logger, "register user", resp, err)
}

// ChangePassword changes the currently authenticated user's password.
func (u *UsersScope) ChangePassword(
	ctx context.Context,
	request types.ChangePasswordRQ,
) (*types.OperationCompletionRS, error) {
	logger := u.scope.Logger.With(slog.String("operation", "ChangePassword"))
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.ChangePasswordWithResponse(
		ctx,
		request,
	)

	return core.Execute(ctx, logger, "change password", resp, err)
}

// RestorePassword initiates a password restore flow by sending a reset link to the given email address.
func (u *UsersScope) RestorePassword(
	ctx context.Context,
	request types.RestorePasswordRQ,
) (*types.OperationCompletionRS, error) {
	logger := u.scope.Logger.With(slog.String("operation", "RestorePassword"))
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.RestorePasswordWithResponse(
		ctx,
		request,
	)

	return core.Execute(ctx, logger, "restore password", resp, err)
}

// IsRestorePasswordBidExist checks whether a password reset bid with the given UUID exists.
func (u *UsersScope) IsRestorePasswordBidExist(ctx context.Context, uuid string) (bool, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "IsRestorePasswordBidExist"),
		slog.String("uuid", uuid),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.IsRestorePasswordBidExistWithResponse(ctx, uuid)

	result, err := core.Execute(ctx, logger, "check restore password bid", resp, err)
	if err != nil {
		return false, err
	}

	if result == nil || result.Is == nil {
		return false, nil
	}

	return *result.Is, nil
}

// ResetPassword completes the password reset flow using the UUID from the restore email and the new password.
func (u *UsersScope) ResetPassword(
	ctx context.Context,
	request types.ResetPasswordRQ,
) (*types.OperationCompletionRS, error) {
	logger := u.scope.Logger.With(slog.String("operation", "ResetPassword"))
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.ResetPasswordWithResponse(
		ctx,
		request,
	)

	return core.Execute(ctx, logger, "reset password", resp, err)
}

// ListAPIKeys returns the API keys for the given user.
func (u *UsersScope) ListAPIKeys(ctx context.Context, userID int64) ([]types.ApiKeyRS, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "ListAPIKeys"),
		slog.Int64("userId", userID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.GetUsersApiKeysWithResponse(ctx, userID)

	result, err := core.Execute(ctx, logger, "list api keys", resp, err)
	if err != nil {
		return nil, err
	}

	if result == nil || result.Items == nil {
		return []types.ApiKeyRS{}, nil
	}

	return result.Items, nil
}

// CreateAPIKey creates a new API key for the given user. The returned ApiKeyRS
// will have the ApiKey field populated; this is the only time the raw key value
// is available.
func (u *UsersScope) CreateAPIKey(
	ctx context.Context,
	userID int64,
	request types.ApiKeyRQ,
) (*types.ApiKeyRS, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "CreateAPIKey"),
		slog.Int64("userId", userID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.CreateApiKeyWithResponse(
		ctx,
		userID,
		request,
	)

	return core.Execute(ctx, logger, "create api key", resp, err)
}

// DeleteAPIKey deletes the API key with the given ID for the given user.
func (u *UsersScope) DeleteAPIKey(
	ctx context.Context,
	userID int64,
	keyID int64,
) (*types.OperationCompletionRS, error) {
	logger := u.scope.Logger.With(
		slog.String("operation", "DeleteAPIKey"),
		slog.Int64("userId", userID),
		slog.Int64("keyId", keyID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := u.scope.Client.DeleteApiKeyWithResponse(ctx, userID, keyID)

	return core.Execute(ctx, logger, "delete api key", resp, err)
}

func listUsersParamsToExportParams(usersParams *api.GetUsersParams) *api.ExportParams {
	return &api.ExportParams{
		FilterEqType:                usersParams.FilterEqType,
		FilterEqLastLogin:           usersParams.FilterEqLastLogin,
		FilterEqRole:                usersParams.FilterEqRole,
		FilterEqUser:                usersParams.FilterEqUser,
		FilterEqId:                  usersParams.FilterEqId,
		FilterEqExpired:             usersParams.FilterEqExpired,
		FilterEqSynchronizationDate: usersParams.FilterEqSynchronizationDate,
		FilterEqExternalId:          usersParams.FilterEqExternalId,
		FilterEqActive:              usersParams.FilterEqActive,
		FilterEqProject:             usersParams.FilterEqProject,
		FilterEqEmail:               usersParams.FilterEqEmail,
		FilterEqFullName:            usersParams.FilterEqFullName,
		FilterEqProjectId:           usersParams.FilterEqProjectId,
		FilterEqUuid:                usersParams.FilterEqUuid,
	}
}

// ListUsersOption configures parameters for listing users.
type ListUsersOption interface {
	applyListUsers(params *api.GetUsersParams)
}

type listUsersOptionFunc func(*api.GetUsersParams)

func (f listUsersOptionFunc) applyListUsers(params *api.GetUsersParams) {
	f(params)
}

// WithUsersPageNumber sets the page number (0-based).
func WithUsersPageNumber(n int32) listUsersOptionFunc {
	return listUsersOptionFunc(func(params *api.GetUsersParams) {
		params.PagePage = &n
	})
}

// WithUsersPageSize sets the number of results per page.
func WithUsersPageSize(size int32) listUsersOptionFunc {
	return listUsersOptionFunc(func(params *api.GetUsersParams) {
		params.PageSize = &size
	})
}

// WithUsersSort sets the sorting criteria (e.g. "login,asc").
func WithUsersSort(fields ...string) listUsersOptionFunc {
	return listUsersOptionFunc(func(params *api.GetUsersParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}

// WithUsersRole filters users by role (e.g. "ADMINISTRATOR", "USER").
func WithUsersRole(role string) listUsersOptionFunc {
	return listUsersOptionFunc(func(params *api.GetUsersParams) {
		params.FilterEqRole = &role
	})
}

// WithUsersType filters users by account type (e.g. "INTERNAL", "SCIM").
func WithUsersType(accountType string) listUsersOptionFunc {
	return listUsersOptionFunc(func(params *api.GetUsersParams) {
		params.FilterEqType = &accountType
	})
}

// WithUsersActive filters users by active status.
func WithUsersActive(active bool) listUsersOptionFunc {
	return listUsersOptionFunc(func(params *api.GetUsersParams) {
		params.FilterEqActive = &active
	})
}

// WithUsersEmail filters users by email.
func WithUsersEmail(email string) listUsersOptionFunc {
	return listUsersOptionFunc(func(params *api.GetUsersParams) {
		params.FilterEqEmail = &email
	})
}

// SearchUsersOption configures pagination parameters for user search.
type SearchUsersOption interface {
	applySearchUsers(params *api.FindUsersParams)
}

type searchUsersOptionFunc func(*api.FindUsersParams)

func (f searchUsersOptionFunc) applySearchUsers(params *api.FindUsersParams) {
	f(params)
}

// WithSearchUsersPageNumber sets the page number (0-based) for user search.
func WithSearchUsersPageNumber(n int32) searchUsersOptionFunc {
	return searchUsersOptionFunc(func(params *api.FindUsersParams) {
		params.PagePage = &n
	})
}

// WithSearchUsersPageSize sets the number of results per page for user search.
func WithSearchUsersPageSize(size int32) searchUsersOptionFunc {
	return searchUsersOptionFunc(func(params *api.FindUsersParams) {
		params.PageSize = &size
	})
}

// WithSearchUsersSort sets the sorting criteria for user search.
func WithSearchUsersSort(fields ...string) searchUsersOptionFunc {
	return searchUsersOptionFunc(func(params *api.FindUsersParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}

// NewUsersScope constructs a UsersScope for wiring from the facade.
func NewUsersScope(s core.Scope) *UsersScope {
	return &UsersScope{scope: s}
}
