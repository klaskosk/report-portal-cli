package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// GroupsScope provides operations on groups at the instance level.
type GroupsScope struct {
	client *Client
}

// List returns a paginated list of groups.
func (g *GroupsScope) List(ctx context.Context, options ...ListGroupsOption) (*types.GroupPage, error) {
	logger := g.client.logger.With(slog.String("operation", "ListGroups"))

	params := &api.GetGroupsParams{}
	for _, option := range options {
		option.applyListGroups(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.GetGroupsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	return result, nil
}

// Get returns a single group by its numeric ID.
func (g *GroupsScope) Get(ctx context.Context, groupID int64) (*types.GroupInfo, error) {
	logger := g.client.logger.With(
		slog.String("operation", "GetGroup"),
		slog.Int64("groupId", groupID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.GetGroupByIdWithResponse(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}

	return result, nil
}

// Create creates a new group and returns the created group info.
func (g *GroupsScope) Create(ctx context.Context, request types.CreateGroupRequest) (*types.GroupInfo, error) {
	logger := g.client.logger.With(
		slog.String("operation", "CreateGroup"),
		slog.String("name", request.Name),
	)
	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.CreateGroupWithResponse(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON201, response.Body)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}

	return result, nil
}

// Update modifies an existing group by its numeric ID.
func (g *GroupsScope) Update(
	ctx context.Context,
	groupID int64,
	request types.UpdateGroupRequest,
) (*types.SuccessfulUpdate, error) {
	logger := g.client.logger.With(
		slog.String("operation", "UpdateGroup"),
		slog.Int64("groupId", groupID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.UpdateGroupWithResponse(ctx, groupID, request)
	if err != nil {
		return nil, fmt.Errorf("update group: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("update group: %w", err)
	}

	return result, nil
}

// Delete deletes a group by its numeric ID.
func (g *GroupsScope) Delete(ctx context.Context, groupID int64) error {
	logger := g.client.logger.With(
		slog.String("operation", "DeleteGroup"),
		slog.Int64("groupId", groupID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.DeleteGroupWithResponse(ctx, groupID)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if err := handleVoidResponse(response, response.JSONDefault, response.Body); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}

	return nil
}

// ListProjects returns a paginated list of projects assigned to a group.
func (g *GroupsScope) ListProjects(
	ctx context.Context,
	groupID int64,
	options ...ListGroupProjectsOption,
) (*types.GroupProjectsPage, error) {
	logger := g.client.logger.With(
		slog.String("operation", "ListGroupProjects"),
		slog.Int64("groupId", groupID),
	)

	params := &api.GetGroupProjectsParams{}
	for _, option := range options {
		option.applyListGroupProjects(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.GetGroupProjectsWithResponse(ctx, groupID, params)
	if err != nil {
		return nil, fmt.Errorf("list group projects: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list group projects: %w", err)
	}

	return result, nil
}

// GetProject returns a single project-group link by group and project IDs.
func (g *GroupsScope) GetProject(
	ctx context.Context,
	groupID, projectID int64,
) (*types.GroupProjectInfo, error) {
	logger := g.client.logger.With(
		slog.String("operation", "GetGroupProject"),
		slog.Int64("groupId", groupID),
		slog.Int64("projectId", projectID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.GetGroupProjectByIdWithResponse(ctx, groupID, projectID)
	if err != nil {
		return nil, fmt.Errorf("get group project: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get group project: %w", err)
	}

	return result, nil
}

// AddProject adds a project to a group with optional permission configuration.
func (g *GroupsScope) AddProject(
	ctx context.Context,
	groupID, projectID int64,
	request types.AddProjectToGroupByIdRequest,
) (*types.SuccessfulUpdate, error) {
	logger := g.client.logger.With(
		slog.String("operation", "AddProjectToGroup"),
		slog.Int64("groupId", groupID),
		slog.Int64("projectId", projectID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.AddProjectToGroupByIdWithResponse(ctx, groupID, projectID, request)
	if err != nil {
		return nil, fmt.Errorf("add project to group: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("add project to group: %w", err)
	}

	return result, nil
}

// RemoveProject removes a project from a group.
func (g *GroupsScope) RemoveProject(ctx context.Context, groupID, projectID int64) error {
	logger := g.client.logger.With(
		slog.String("operation", "RemoveProjectFromGroup"),
		slog.Int64("groupId", groupID),
		slog.Int64("projectId", projectID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.DeleteProjectFromGroupByIdWithResponse(ctx, groupID, projectID)
	if err != nil {
		return fmt.Errorf("remove project from group: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if err := handleVoidResponse(response, response.JSONDefault, response.Body); err != nil {
		return fmt.Errorf("remove project from group: %w", err)
	}

	return nil
}

// ListUsers returns a paginated list of users in a group.
func (g *GroupsScope) ListUsers(
	ctx context.Context,
	groupID int64,
	options ...ListGroupUsersOption,
) (*types.GroupUsersPage, error) {
	logger := g.client.logger.With(
		slog.String("operation", "ListGroupUsers"),
		slog.Int64("groupId", groupID),
	)

	params := &api.GetGroupUsersParams{}
	for _, option := range options {
		option.applyListGroupUsers(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.GetGroupUsersWithResponse(ctx, groupID, params)
	if err != nil {
		return nil, fmt.Errorf("list group users: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list group users: %w", err)
	}

	return result, nil
}

// GetUser returns a single user-group link by group and user IDs.
func (g *GroupsScope) GetUser(
	ctx context.Context,
	groupID, userID int64,
) (*types.GroupUserInfo, error) {
	logger := g.client.logger.With(
		slog.String("operation", "GetGroupUser"),
		slog.Int64("groupId", groupID),
		slog.Int64("userId", userID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.GetGroupUserByIdWithResponse(ctx, groupID, userID)
	if err != nil {
		return nil, fmt.Errorf("get group user: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get group user: %w", err)
	}

	return result, nil
}

// AddUser adds a user to a group.
func (g *GroupsScope) AddUser(
	ctx context.Context,
	groupID, userID int64,
) (*types.SuccessfulUpdate, error) {
	logger := g.client.logger.With(
		slog.String("operation", "AddUserToGroup"),
		slog.Int64("groupId", groupID),
		slog.Int64("userId", userID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.AddUserToGroupByIdWithResponse(ctx, groupID, userID)
	if err != nil {
		return nil, fmt.Errorf("add user to group: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("add user to group: %w", err)
	}

	return result, nil
}

// RemoveUser removes a user from a group.
func (g *GroupsScope) RemoveUser(ctx context.Context, groupID, userID int64) error {
	logger := g.client.logger.With(
		slog.String("operation", "RemoveUserFromGroup"),
		slog.Int64("groupId", groupID),
		slog.Int64("userId", userID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := g.client.raw.DeleteUserFromGroupByIdWithResponse(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("remove user from group: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if err := handleVoidResponse(response, response.JSONDefault, response.Body); err != nil {
		return fmt.Errorf("remove user from group: %w", err)
	}

	return nil
}

// ListGroupsOption configures the pagination parameters for listing groups.
type ListGroupsOption interface {
	applyListGroups(*api.GetGroupsParams)
}

type listGroupsOptionFunc func(*api.GetGroupsParams)

func (f listGroupsOptionFunc) applyListGroups(params *api.GetGroupsParams) {
	f(params)
}

// WithGroupPageSize sets the maximum number of groups per page.
func WithGroupPageSize(size int32) ListGroupsOption {
	return listGroupsOptionFunc(func(params *api.GetGroupsParams) {
		params.Limit = &size
	})
}

// WithGroupOffset sets the offset for group pagination.
func WithGroupOffset(offset int32) ListGroupsOption {
	return listGroupsOptionFunc(func(params *api.GetGroupsParams) {
		params.Offset = &offset
	})
}

// WithGroupSort sets the field to sort groups by.
func WithGroupSort(sort string) ListGroupsOption {
	return listGroupsOptionFunc(func(params *api.GetGroupsParams) {
		params.Sort = &sort
	})
}

// WithGroupOrder sets the sort direction for group listing.
func WithGroupOrder(order string) ListGroupsOption {
	return listGroupsOptionFunc(func(params *api.GetGroupsParams) {
		params.Order = &order
	})
}

// ListGroupProjectsOption configures the pagination parameters for listing group projects.
type ListGroupProjectsOption interface {
	applyListGroupProjects(*api.GetGroupProjectsParams)
}

type listGroupProjectsOptionFunc func(*api.GetGroupProjectsParams)

func (f listGroupProjectsOptionFunc) applyListGroupProjects(params *api.GetGroupProjectsParams) {
	f(params)
}

// WithGroupProjectPageSize sets the maximum number of projects per page.
func WithGroupProjectPageSize(size int32) ListGroupProjectsOption {
	return listGroupProjectsOptionFunc(func(params *api.GetGroupProjectsParams) {
		params.Limit = &size
	})
}

// WithGroupProjectOffset sets the offset for group project pagination.
func WithGroupProjectOffset(offset int32) ListGroupProjectsOption {
	return listGroupProjectsOptionFunc(func(params *api.GetGroupProjectsParams) {
		params.Offset = &offset
	})
}

// ListGroupUsersOption configures the pagination parameters for listing group users.
type ListGroupUsersOption interface {
	applyListGroupUsers(*api.GetGroupUsersParams)
}

type listGroupUsersOptionFunc func(*api.GetGroupUsersParams)

func (f listGroupUsersOptionFunc) applyListGroupUsers(params *api.GetGroupUsersParams) {
	f(params)
}

// WithGroupUserPageSize sets the maximum number of users per page.
func WithGroupUserPageSize(size int32) ListGroupUsersOption {
	return listGroupUsersOptionFunc(func(params *api.GetGroupUsersParams) {
		params.Limit = &size
	})
}

// WithGroupUserOffset sets the offset for group user pagination.
func WithGroupUserOffset(offset int32) ListGroupUsersOption {
	return listGroupUsersOptionFunc(func(params *api.GetGroupUsersParams) {
		params.Offset = &offset
	})
}
