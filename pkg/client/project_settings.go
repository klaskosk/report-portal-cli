package client

import (
	"context"
	"fmt"
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

	response, err := s.project.client.raw.GetProjectSettingsUsingGETWithResponse(
		ctx,
		s.project.projectName,
	)
	if err != nil {
		return nil, fmt.Errorf("get project settings: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "get project settings", response.JSONDefault, response.JSON200)
}

// GetNotifications returns the notification rules configured for this project.
func (s *ProjectSettingsScope) GetNotifications(ctx context.Context) ([]types.SenderCaseDTO, error) {
	logger := s.project.operationLogger("GetNotifications")
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.GetNotificationsUsingGETWithResponse(
		ctx,
		s.project.projectName,
	)
	if err != nil {
		return nil, fmt.Errorf("get notifications: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, "get notifications", response.JSONDefault, response.JSON200)
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

	response, err := s.project.client.raw.CreateNotificationUsingPOSTWithResponse(
		ctx,
		s.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "create notification", response.JSONDefault, response.JSON201)
}

// UpdateNotification modifies an existing notification rule.
func (s *ProjectSettingsScope) UpdateNotification(
	ctx context.Context,
	request types.SenderCaseDTO,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("UpdateNotification")
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.UpdateNotificationUsingPUTWithResponse(
		ctx,
		s.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("update notification: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "update notification", response.JSONDefault, response.JSON201)
}

// DeleteNotification removes a notification rule by its ID.
func (s *ProjectSettingsScope) DeleteNotification(
	ctx context.Context,
	id int64,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("DeleteNotification", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.DeleteNotificationUsingDELETEWithResponse(
		ctx,
		s.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("delete notification: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "delete notification", response.JSONDefault, response.JSON200)
}

// CreatePatternTemplate creates a pattern template for this project.
func (s *ProjectSettingsScope) CreatePatternTemplate(
	ctx context.Context,
	request types.CreatePatternTemplateRQ,
) (*types.EntryCreatedRS, error) {
	logger := s.project.operationLogger("CreatePatternTemplate")
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.CreatePatternTemplateUsingPOSTWithResponse(
		ctx,
		s.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create pattern template: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "create pattern template", response.JSONDefault, response.JSON201)
}

// UpdatePatternTemplate modifies an existing pattern template.
func (s *ProjectSettingsScope) UpdatePatternTemplate(
	ctx context.Context,
	id int64,
	request types.UpdatePatternTemplateRQ,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("UpdatePatternTemplate", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.UpdatePatternTemplateUsingPUTWithResponse(
		ctx,
		s.project.projectName,
		id,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("update pattern template: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "update pattern template", response.JSONDefault, response.JSON200)
}

// DeletePatternTemplate removes a pattern template by its ID.
func (s *ProjectSettingsScope) DeletePatternTemplate(
	ctx context.Context,
	id int64,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("DeletePatternTemplate", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.DeletePatternTemplateUsingDELETEWithResponse(
		ctx,
		s.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("delete pattern template: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "delete pattern template", response.JSONDefault, response.JSON200)
}

// CreateIssueSubType creates a custom issue sub-type for this project.
func (s *ProjectSettingsScope) CreateIssueSubType(
	ctx context.Context,
	request types.CreateIssueSubTypeRQ,
) (*types.IssueSubTypeCreatedRS, error) {
	logger := s.project.operationLogger("CreateIssueSubType")
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.CreateProjectIssueSubTypeUsingPOSTWithResponse(
		ctx,
		s.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create issue sub-type: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "create issue sub-type", response.JSONDefault, response.JSON201)
}

// UpdateIssueSubType modifies an existing issue sub-type.
func (s *ProjectSettingsScope) UpdateIssueSubType(
	ctx context.Context,
	request types.UpdateOneIssueSubTypeRQ,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("UpdateIssueSubType")
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.UpdateProjectIssueSubTypeUsingPUTWithResponse(
		ctx,
		s.project.projectName,
		types.UpdateIssueSubTypeRQ{Ids: []types.UpdateOneIssueSubTypeRQ{request}},
	)
	if err != nil {
		return nil, fmt.Errorf("update issue sub-type: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "update issue sub-type", response.JSONDefault, response.JSON200)
}

// DeleteIssueSubType removes a custom issue sub-type by its ID.
func (s *ProjectSettingsScope) DeleteIssueSubType(
	ctx context.Context,
	id int64,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("DeleteIssueSubType", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.DeleteProjectIssueSubTypeUsingDELETEWithResponse(
		ctx,
		s.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("delete issue sub-type: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "delete issue sub-type", response.JSONDefault, response.JSON200)
}
