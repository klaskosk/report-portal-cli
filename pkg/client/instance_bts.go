package client

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// InstanceBugTrackingScope provides operations on bug tracking system integrations
// at the instance level, without requiring a project name.
type InstanceBugTrackingScope struct {
	client *Client
}

// GetSystemFields returns the available BTS system fields for the given
// integration and issue type. Unlike the project-scoped variant, this method
// requires issueType as specified by the API.
func (s *InstanceBugTrackingScope) GetSystemFields(
	ctx context.Context,
	integrationID int64,
	issueType string,
) ([]types.PostFormField, error) {
	logger := s.client.logger.With(
		slog.String("operation", "GetInstanceBTSSystemFields"),
		slog.Int64("integrationId", integrationID),
		slog.String("issueType", issueType),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetSetOfIntegrationSystemFields1WithResponse(
		ctx,
		integrationID,
		&api.GetSetOfIntegrationSystemFields1Params{
			IssueType: issueType,
		},
	)

	result, err := execute(ctx, logger, "get instance bts system fields", response, err)
	if err != nil {
		return nil, err
	}

	if *result == nil {
		return []types.PostFormField{}, nil
	}

	return *result, nil
}

// GetAllowableIssueTypes returns the allowable issue types for the given
// BTS integration at the instance level.
func (s *InstanceBugTrackingScope) GetAllowableIssueTypes(
	ctx context.Context,
	integrationID int64,
) ([]string, error) {
	logger := s.client.logger.With(
		slog.String("operation", "GetInstanceBTSAllowableIssueTypes"),
		slog.Int64("integrationId", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetAllowableIssueTypes1WithResponse(
		ctx,
		integrationID,
	)

	result, err := execute(ctx, logger, "get instance bts allowable issue types", response, err)
	if err != nil {
		return nil, err
	}

	if *result == nil {
		return []string{}, nil
	}

	return *result, nil
}
