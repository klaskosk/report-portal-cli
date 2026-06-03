package settings

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// ProjectSettingsScope provides read operations on project settings such as
// issue sub-types and notification rules.
type ProjectSettingsScope struct {
	scope core.Scope
}

// Get returns the full project settings including issue sub-types.
func (s *ProjectSettingsScope) Get(ctx context.Context) (*types.ProjectSettingsResource, error) {
	logger := s.scope.OperationLogger("GetProjectSettings")
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetProjectSettingsWithResponse(ctx, s.scope.ProjectName)

	return core.Execute[types.ProjectSettingsResource](
		ctx,
		logger,
		"get project settings",
		resp,
		err,
	)
}

// GetNotifications returns the notification rules configured for this project.
func (s *ProjectSettingsScope) GetNotifications(
	ctx context.Context,
) ([]types.SenderCaseDTO, error) {
	logger := s.scope.OperationLogger("GetNotifications")
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetNotificationsWithResponse(ctx, s.scope.ProjectName)

	result, err := core.Execute[[]types.SenderCaseDTO](ctx, logger, "get notifications", resp, err)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return []types.SenderCaseDTO{}, nil
	}

	return *result, nil
}

// CreateNotification creates a notification rule for this project.
func (s *ProjectSettingsScope) CreateNotification(
	ctx context.Context,
	request types.SenderCaseDTO,
) (*types.EntryCreatedRS, error) {
	logger := s.scope.OperationLogger("CreateNotification")
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.CreateNotificationWithResponse(
		ctx, s.scope.ProjectName, request,
	)

	return core.Execute[types.EntryCreatedRS](ctx, logger, "create notification", resp, err)
}

// UpdateNotification modifies an existing notification rule.
func (s *ProjectSettingsScope) UpdateNotification(
	ctx context.Context,
	request types.SenderCaseDTO,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.OperationLogger("UpdateNotification")
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.UpdateNotificationWithResponse(
		ctx, s.scope.ProjectName, request,
	)

	return core.Execute[types.OperationCompletionRS](ctx, logger, "update notification", resp, err)
}

// DeleteNotification removes a notification rule by its ID.
func (s *ProjectSettingsScope) DeleteNotification(
	ctx context.Context,
	notificationID int64,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.OperationLogger("DeleteNotification", slog.Int64("id", notificationID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.DeleteNotificationWithResponse(
		ctx, s.scope.ProjectName, notificationID,
	)

	return core.Execute[types.OperationCompletionRS](ctx, logger, "delete notification", resp, err)
}

// CreatePatternTemplate creates a pattern template for this project.
func (s *ProjectSettingsScope) CreatePatternTemplate(
	ctx context.Context,
	request types.CreatePatternTemplateRQ,
) (*types.EntryCreatedRS, error) {
	logger := s.scope.OperationLogger("CreatePatternTemplate")
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.CreatePatternTemplateWithResponse(
		ctx, s.scope.ProjectName, request,
	)

	return core.Execute[types.EntryCreatedRS](ctx, logger, "create pattern template", resp, err)
}

// UpdatePatternTemplate modifies an existing pattern template.
func (s *ProjectSettingsScope) UpdatePatternTemplate(
	ctx context.Context,
	patternTemplateID int64,
	request types.UpdatePatternTemplateRQ,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.OperationLogger(
		"UpdatePatternTemplate",
		slog.Int64("id", patternTemplateID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.UpdatePatternTemplateWithResponse(
		ctx, s.scope.ProjectName, patternTemplateID, request,
	)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"update pattern template",
		resp,
		err,
	)
}

// DeletePatternTemplate removes a pattern template by its ID.
func (s *ProjectSettingsScope) DeletePatternTemplate(
	ctx context.Context,
	patternTemplateID int64,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.OperationLogger(
		"DeletePatternTemplate",
		slog.Int64("id", patternTemplateID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.DeletePatternTemplateWithResponse(
		ctx, s.scope.ProjectName, patternTemplateID,
	)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"delete pattern template",
		resp,
		err,
	)
}

// CreateIssueSubType creates a custom issue sub-type for this project.
func (s *ProjectSettingsScope) CreateIssueSubType(
	ctx context.Context,
	request types.CreateIssueSubTypeRQ,
) (*types.IssueSubTypeCreatedRS, error) {
	logger := s.scope.OperationLogger("CreateIssueSubType")
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.CreateProjectIssueSubTypeWithResponse(
		ctx, s.scope.ProjectName, request,
	)

	return core.Execute[types.IssueSubTypeCreatedRS](
		ctx,
		logger,
		"create issue sub-type",
		resp,
		err,
	)
}

// UpdateIssueSubType modifies an existing issue sub-type.
func (s *ProjectSettingsScope) UpdateIssueSubType(
	ctx context.Context,
	request types.UpdateOneIssueSubTypeRQ,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.OperationLogger("UpdateIssueSubType")
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.UpdateProjectIssueSubTypeWithResponse(
		ctx, s.scope.ProjectName,
		types.UpdateIssueSubTypeRQ{Ids: []types.UpdateOneIssueSubTypeRQ{request}},
	)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"update issue sub-type",
		resp,
		err,
	)
}

// DeleteIssueSubType removes a custom issue sub-type by its ID.
func (s *ProjectSettingsScope) DeleteIssueSubType(
	ctx context.Context,
	issueSubTypeID int64,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.OperationLogger("DeleteIssueSubType", slog.Int64("id", issueSubTypeID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.DeleteProjectIssueSubTypeWithResponse(
		ctx, s.scope.ProjectName, issueSubTypeID,
	)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"delete issue sub-type",
		resp,
		err,
	)
}

// NewSettingsScope constructs a SettingsScope for wiring from the facade.
func NewSettingsScope(s core.Scope) *SettingsScope {
	return &SettingsScope{scope: s}
}

// NewProjectSettingsScope constructs a ProjectSettingsScope for wiring from the facade.
func NewProjectSettingsScope(s core.Scope) *ProjectSettingsScope {
	return &ProjectSettingsScope{scope: s}
}
