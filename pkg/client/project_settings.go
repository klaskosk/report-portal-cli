package client

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// ProjectSettingsScope provides read operations on project settings such as
// issue sub-types and notification rules.
type ProjectSettingsScope struct {
	project *ProjectScope
}

// Settings returns a ProjectSettingsScope for reading configuration of this project.
func (p *ProjectScope) Settings() *ProjectSettingsScope {
	return &ProjectSettingsScope{project: p}
}

// Get returns the full project settings including issue sub-types.
func (s *ProjectSettingsScope) Get(ctx context.Context) (*types.ProjectSettingsResource, error) {
	logger := s.project.operationLogger("GetProjectSettings")
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.GetProjectSettingsWithResponse(ctx, s.project.projectName)

	return execute[types.ProjectSettingsResource](ctx, logger, "get project settings", resp, err)
}

// GetNotifications returns the notification rules configured for this project.
func (s *ProjectSettingsScope) GetNotifications(
	ctx context.Context,
) ([]types.SenderCaseDTO, error) {
	logger := s.project.operationLogger("GetNotifications")
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.GetNotificationsWithResponse(ctx, s.project.projectName)

	result, err := execute[[]types.SenderCaseDTO](ctx, logger, "get notifications", resp, err)
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
	logger := s.project.operationLogger("CreateNotification")
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.CreateNotificationWithResponse(
		ctx, s.project.projectName, request,
	)

	return execute[types.EntryCreatedRS](ctx, logger, "create notification", resp, err)
}

// UpdateNotification modifies an existing notification rule.
func (s *ProjectSettingsScope) UpdateNotification(
	ctx context.Context,
	request types.SenderCaseDTO,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("UpdateNotification")
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.UpdateNotificationWithResponse(
		ctx, s.project.projectName, request,
	)

	return execute[types.OperationCompletionRS](ctx, logger, "update notification", resp, err)
}

// DeleteNotification removes a notification rule by its ID.
func (s *ProjectSettingsScope) DeleteNotification(
	ctx context.Context,
	notificationID int64,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("DeleteNotification", slog.Int64("id", notificationID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.DeleteNotificationWithResponse(
		ctx, s.project.projectName, notificationID,
	)

	return execute[types.OperationCompletionRS](ctx, logger, "delete notification", resp, err)
}

// CreatePatternTemplate creates a pattern template for this project.
func (s *ProjectSettingsScope) CreatePatternTemplate(
	ctx context.Context,
	request types.CreatePatternTemplateRQ,
) (*types.EntryCreatedRS, error) {
	logger := s.project.operationLogger("CreatePatternTemplate")
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.CreatePatternTemplateWithResponse(
		ctx, s.project.projectName, request,
	)

	return execute[types.EntryCreatedRS](ctx, logger, "create pattern template", resp, err)
}

// UpdatePatternTemplate modifies an existing pattern template.
func (s *ProjectSettingsScope) UpdatePatternTemplate(
	ctx context.Context,
	patternTemplateID int64,
	request types.UpdatePatternTemplateRQ,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger(
		"UpdatePatternTemplate",
		slog.Int64("id", patternTemplateID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.UpdatePatternTemplateWithResponse(
		ctx, s.project.projectName, patternTemplateID, request,
	)

	return execute[types.OperationCompletionRS](ctx, logger, "update pattern template", resp, err)
}

// DeletePatternTemplate removes a pattern template by its ID.
func (s *ProjectSettingsScope) DeletePatternTemplate(
	ctx context.Context,
	patternTemplateID int64,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger(
		"DeletePatternTemplate",
		slog.Int64("id", patternTemplateID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.DeletePatternTemplateWithResponse(
		ctx, s.project.projectName, patternTemplateID,
	)

	return execute[types.OperationCompletionRS](ctx, logger, "delete pattern template", resp, err)
}

// CreateIssueSubType creates a custom issue sub-type for this project.
func (s *ProjectSettingsScope) CreateIssueSubType(
	ctx context.Context,
	request types.CreateIssueSubTypeRQ,
) (*types.IssueSubTypeCreatedRS, error) {
	logger := s.project.operationLogger("CreateIssueSubType")
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.CreateProjectIssueSubTypeWithResponse(
		ctx, s.project.projectName, request,
	)

	return execute[types.IssueSubTypeCreatedRS](ctx, logger, "create issue sub-type", resp, err)
}

// UpdateIssueSubType modifies an existing issue sub-type.
func (s *ProjectSettingsScope) UpdateIssueSubType(
	ctx context.Context,
	request types.UpdateOneIssueSubTypeRQ,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("UpdateIssueSubType")
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.UpdateProjectIssueSubTypeWithResponse(
		ctx, s.project.projectName,
		types.UpdateIssueSubTypeRQ{Ids: []types.UpdateOneIssueSubTypeRQ{request}},
	)

	return execute[types.OperationCompletionRS](ctx, logger, "update issue sub-type", resp, err)
}

// DeleteIssueSubType removes a custom issue sub-type by its ID.
func (s *ProjectSettingsScope) DeleteIssueSubType(
	ctx context.Context,
	issueSubTypeID int64,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("DeleteIssueSubType", slog.Int64("id", issueSubTypeID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.DeleteProjectIssueSubTypeWithResponse(
		ctx, s.project.projectName, issueSubTypeID,
	)

	return execute[types.OperationCompletionRS](ctx, logger, "delete issue sub-type", resp, err)
}
