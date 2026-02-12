package client

import (
	"context"
	"fmt"

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
