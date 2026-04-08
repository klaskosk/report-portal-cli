package client

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// ProjectGroupsScope provides operations on group-project associations within a specific project.
type ProjectGroupsScope struct {
	project *ProjectScope
}

// List returns a paginated list of groups associated with the project.
func (s *ProjectGroupsScope) List(ctx context.Context, options ...ListProjectGroupsOption) (*types.ProjectGroupsPage, error) {
	logger := s.project.operationLogger("ListProjectGroups")

	params := &api.GetGroupsOfProjectParams{}
	for _, option := range options {
		option.applyListProjectGroups(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.GetGroupsOfProjectWithResponse(ctx, s.project.projectName, params)

	return execute(ctx, logger, "list project groups", resp, err)
}

// Get returns a single group-project association by group ID.
func (s *ProjectGroupsScope) Get(ctx context.Context, groupID int64) (*types.ProjectGroupInfo, error) {
	logger := s.project.operationLogger("GetProjectGroup", slog.Int64("groupId", groupID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.GetProjectGroupByIdWithResponse(ctx, s.project.projectName, groupID)

	return execute(ctx, logger, "get project group", resp, err)
}

// AddOrUpdate adds or updates a group-project association. This is a PUT with upsert semantics.
func (s *ProjectGroupsScope) AddOrUpdate(
	ctx context.Context,
	groupID int64,
	request types.AddProjectToGroupByIdRequest,
) (*types.SuccessfulUpdate, error) {
	logger := s.project.operationLogger("AddOrUpdateProjectGroup", slog.Int64("groupId", groupID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.AddGroupToProjectByIdWithResponse(
		ctx, s.project.projectName, groupID, request,
	)

	return execute(ctx, logger, "add or update project group", resp, err)
}

// Delete removes a group-project association by group ID.
func (s *ProjectGroupsScope) Delete(ctx context.Context, groupID int64) error {
	logger := s.project.operationLogger("DeleteProjectGroup", slog.Int64("groupId", groupID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.DeleteGroupFromProjectByIdWithResponse(ctx, s.project.projectName, groupID)

	return executeVoid(ctx, logger, "delete project group", resp, err)
}

// ListProjectGroupsOption configures the pagination parameters for listing project groups.
type ListProjectGroupsOption interface {
	applyListProjectGroups(*api.GetGroupsOfProjectParams)
}

type listProjectGroupsOptionFunc func(*api.GetGroupsOfProjectParams)

func (f listProjectGroupsOptionFunc) applyListProjectGroups(params *api.GetGroupsOfProjectParams) {
	f(params)
}

// WithProjectGroupPageSize sets the maximum number of groups per page.
func WithProjectGroupPageSize(size int32) ListProjectGroupsOption {
	return listProjectGroupsOptionFunc(func(params *api.GetGroupsOfProjectParams) {
		params.Limit = &size
	})
}

// WithProjectGroupPageOffset sets the offset for project group pagination.
func WithProjectGroupPageOffset(offset int32) ListProjectGroupsOption {
	return listProjectGroupsOptionFunc(func(params *api.GetGroupsOfProjectParams) {
		params.Offset = &offset
	})
}
