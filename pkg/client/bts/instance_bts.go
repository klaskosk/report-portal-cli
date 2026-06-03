package bts

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// InstanceBugTrackingScope provides operations on bug tracking system integrations
// at the instance level, without requiring a project name.
type InstanceBugTrackingScope struct {
	scope core.Scope
}

// GetSystemFields returns the available BTS system fields for the given
// integration and issue type. Unlike the project-scoped variant, this method
// requires issueType as specified by the API.
func (s *InstanceBugTrackingScope) GetSystemFields(
	ctx context.Context,
	integrationID int64,
	issueType string,
) ([]types.PostFormField, error) {
	logger := s.scope.Logger.With(
		slog.String("operation", "GetInstanceBTSSystemFields"),
		slog.Int64("integrationId", integrationID),
		slog.String("issueType", issueType),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.scope.Client.GetSetOfIntegrationSystemFields1WithResponse(
		ctx,
		integrationID,
		&api.GetSetOfIntegrationSystemFields1Params{
			IssueType: issueType,
		},
	)

	result, err := core.Execute(ctx, logger, "get instance bts system fields", response, err)
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
	logger := s.scope.Logger.With(
		slog.String("operation", "GetInstanceBTSAllowableIssueTypes"),
		slog.Int64("integrationId", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.scope.Client.GetAllowableIssueTypes1WithResponse(
		ctx,
		integrationID,
	)

	result, err := core.Execute(
		ctx,
		logger,
		"get instance bts allowable issue types",
		response,
		err,
	)
	if err != nil {
		return nil, err
	}

	if *result == nil {
		return []string{}, nil
	}

	return *result, nil
}

// NewBugTrackingScope constructs a BugTrackingScope for wiring from the facade.
func NewBugTrackingScope(s core.Scope) *BugTrackingScope {
	return &BugTrackingScope{scope: s}
}

// NewInstanceBugTrackingScope constructs a InstanceBugTrackingScope for wiring from the facade.
func NewInstanceBugTrackingScope(s core.Scope) *InstanceBugTrackingScope {
	return &InstanceBugTrackingScope{scope: s}
}
