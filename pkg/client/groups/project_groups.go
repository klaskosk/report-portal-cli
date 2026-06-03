package groups

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// ProjectGroupsScope provides operations on group-project associations within a specific project.
type ProjectGroupsScope struct {
	scope core.Scope
}

// List returns a paginated list of groups associated with the project.
func (s *ProjectGroupsScope) List(
	ctx context.Context,
	options ...ListProjectGroupsOption,
) (*types.ProjectGroupsPage, error) {
	logger := s.scope.OperationLogger("ListProjectGroups")

	params := &api.GetGroupsOfProjectParams{}
	for _, option := range options {
		option.applyListProjectGroups(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetGroupsOfProjectWithResponse(
		ctx,
		s.scope.ProjectName,
		params,
	)

	return core.Execute(ctx, logger, "list project groups", resp, err)
}

// Get returns a single group-project association by group ID.
func (s *ProjectGroupsScope) Get(
	ctx context.Context,
	groupID int64,
) (*types.ProjectGroupInfo, error) {
	logger := s.scope.OperationLogger("GetProjectGroup", slog.Int64("groupId", groupID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetProjectGroupByIdWithResponse(
		ctx,
		s.scope.ProjectName,
		groupID,
	)

	return core.Execute(ctx, logger, "get project group", resp, err)
}

// AddOrUpdate adds or updates a group-project association. This is a PUT with upsert semantics.
func (s *ProjectGroupsScope) AddOrUpdate(
	ctx context.Context,
	groupID int64,
	request types.AddProjectToGroupByIdRequest,
) (*types.SuccessfulUpdate, error) {
	logger := s.scope.OperationLogger("AddOrUpdateProjectGroup", slog.Int64("groupId", groupID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.AddGroupToProjectByIdWithResponse(
		ctx, s.scope.ProjectName, groupID, request,
	)

	return core.Execute(ctx, logger, "add or update project group", resp, err)
}

// Delete removes a group-project association by group ID.
func (s *ProjectGroupsScope) Delete(ctx context.Context, groupID int64) error {
	logger := s.scope.OperationLogger("DeleteProjectGroup", slog.Int64("groupId", groupID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.DeleteGroupFromProjectByIdWithResponse(
		ctx,
		s.scope.ProjectName,
		groupID,
	)

	return core.ExecuteVoid(ctx, logger, "delete project group", resp, err)
}

// ListProjectGroupsOption configures the pagination parameters for listing project groups.
type ListProjectGroupsOption interface {
	applyListProjectGroups(params *api.GetGroupsOfProjectParams)
}

type listProjectGroupsOptionFunc func(*api.GetGroupsOfProjectParams)

func (f listProjectGroupsOptionFunc) applyListProjectGroups(params *api.GetGroupsOfProjectParams) {
	f(params)
}

// WithProjectGroupPageSize sets the maximum number of groups per page.
func WithProjectGroupPageSize(size int32) listProjectGroupsOptionFunc {
	return listProjectGroupsOptionFunc(func(params *api.GetGroupsOfProjectParams) {
		params.Limit = &size
	})
}

// WithProjectGroupPageOffset sets the offset for project group pagination.
func WithProjectGroupPageOffset(offset int32) listProjectGroupsOptionFunc {
	return listProjectGroupsOptionFunc(func(params *api.GetGroupsOfProjectParams) {
		params.Offset = &offset
	})
}

// NewGroupsScope constructs a GroupsScope for wiring from the facade.
func NewGroupsScope(s core.Scope) *GroupsScope {
	return &GroupsScope{scope: s}
}

// NewProjectGroupsScope constructs a ProjectGroupsScope for wiring from the facade.
func NewProjectGroupsScope(s core.Scope) *ProjectGroupsScope {
	return &ProjectGroupsScope{scope: s}
}
