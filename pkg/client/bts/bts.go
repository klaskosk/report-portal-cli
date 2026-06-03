package bts

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// BugTrackingScope provides operations on bug tracking system integrations within a project.
type BugTrackingScope struct {
	scope core.Scope
}

// GetSystemFields returns the available BTS system fields for the given integration.
func (s *BugTrackingScope) GetSystemFields(
	ctx context.Context,
	integrationID int64,
) ([]types.PostFormField, error) {
	logger := s.scope.OperationLogger(
		"GetBTSSystemFields",
		slog.Int64("integrationId", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.scope.Client.GetSetOfIntegrationSystemFieldsWithResponse(
		ctx,
		s.scope.ProjectName,
		integrationID,
		&api.GetSetOfIntegrationSystemFieldsParams{},
	)

	result, err := core.Execute(ctx, logger, "get bts system fields", response, err)
	if err != nil {
		return nil, err
	}

	return *result, nil
}

// GetAllowableIssueTypes returns the allowable issue types for the given BTS integration.
func (s *BugTrackingScope) GetAllowableIssueTypes(
	ctx context.Context,
	integrationID int64,
) ([]string, error) {
	logger := s.scope.OperationLogger(
		"GetBTSAllowableIssueTypes",
		slog.Int64("integrationId", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.scope.Client.GetAllowableIssueTypesWithResponse(
		ctx,
		s.scope.ProjectName,
		integrationID,
	)

	result, err := core.Execute(ctx, logger, "get bts allowable issue types", response, err)
	if err != nil {
		return nil, err
	}

	return *result, nil
}

// GetTicket returns a ticket from the BTS by its ID. The btsURL and btsProject parameters
// identify which BTS instance and project to query.
func (s *BugTrackingScope) GetTicket(
	ctx context.Context,
	ticketID string,
	btsURL string,
	btsProject string,
) (*types.Ticket, error) {
	logger := s.scope.OperationLogger("GetBTSTicket",
		slog.String("ticketId", ticketID),
		slog.String("btsUrl", btsURL),
		slog.String("btsProject", btsProject),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.scope.Client.GetTicketWithResponse(
		ctx,
		s.scope.ProjectName,
		ticketID,
		&api.GetTicketParams{
			BtsUrl:     btsURL,
			BtsProject: btsProject,
		},
	)

	return core.Execute(ctx, logger, "get bts ticket", response, err)
}

// CreateIssue creates a new issue in the BTS for the given integration.
func (s *BugTrackingScope) CreateIssue(
	ctx context.Context,
	integrationID int64,
	request types.PostTicketRQ,
) (*types.Ticket, error) {
	logger := s.scope.OperationLogger(
		"CreateBTSIssue",
		slog.Int64("integrationId", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.scope.Client.CreateIssueWithResponse(
		ctx,
		s.scope.ProjectName,
		integrationID,
		request,
	)

	return core.Execute(ctx, logger, "create bts issue", response, err)
}

// GetTicketIDs returns ticket IDs matching the given term within a launch.
func (s *BugTrackingScope) GetTicketIDs(
	ctx context.Context,
	launchID int64,
	term string,
) ([]string, error) {
	logger := s.scope.OperationLogger("GetBTSTicketIds",
		slog.Int64("launchId", launchID),
		slog.String("term", term),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.scope.Client.GetTicketIdsWithResponse(
		ctx,
		s.scope.ProjectName,
		&api.GetTicketIdsParams{
			Launch: launchID,
			Term:   term,
		},
	)

	result, err := core.Execute(ctx, logger, "get bts ticket ids", response, err)
	if err != nil {
		return nil, err
	}

	return *result, nil
}

// GetTicketIDsForProject returns ticket IDs matching the given term across the project.
func (s *BugTrackingScope) GetTicketIDsForProject(
	ctx context.Context,
	term string,
) ([]string, error) {
	logger := s.scope.OperationLogger("GetBTSTicketIdsForProject",
		slog.String("term", term),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.scope.Client.GetTicketIdsForProjectWithResponse(
		ctx,
		s.scope.ProjectName,
		&api.GetTicketIdsForProjectParams{
			Term: term,
		},
	)

	result, err := core.Execute(ctx, logger, "get bts ticket ids for project", response, err)
	if err != nil {
		return nil, err
	}

	return *result, nil
}
