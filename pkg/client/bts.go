package client

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// BugTrackingScope provides operations on bug tracking system integrations within a project.
type BugTrackingScope struct {
	project *ProjectScope
}

// GetSystemFields returns the available BTS system fields for the given integration.
func (s *BugTrackingScope) GetSystemFields(
	ctx context.Context,
	integrationID int64,
) ([]types.PostFormField, error) {
	logger := s.project.operationLogger(
		"GetBTSSystemFields",
		slog.Int64("integrationId", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.GetSetOfIntegrationSystemFieldsWithResponse(
		ctx,
		s.project.projectName,
		integrationID,
		&api.GetSetOfIntegrationSystemFieldsParams{},
	)

	result, err := execute(ctx, logger, "get bts system fields", response, err)
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
	logger := s.project.operationLogger(
		"GetBTSAllowableIssueTypes",
		slog.Int64("integrationId", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.GetAllowableIssueTypesWithResponse(
		ctx,
		s.project.projectName,
		integrationID,
	)

	result, err := execute(ctx, logger, "get bts allowable issue types", response, err)
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
	logger := s.project.operationLogger("GetBTSTicket",
		slog.String("ticketId", ticketID),
		slog.String("btsUrl", btsURL),
		slog.String("btsProject", btsProject),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.GetTicketWithResponse(
		ctx,
		s.project.projectName,
		ticketID,
		&api.GetTicketParams{
			BtsUrl:     btsURL,
			BtsProject: btsProject,
		},
	)

	return execute(ctx, logger, "get bts ticket", response, err)
}

// CreateIssue creates a new issue in the BTS for the given integration.
func (s *BugTrackingScope) CreateIssue(
	ctx context.Context,
	integrationID int64,
	request types.PostTicketRQ,
) (*types.Ticket, error) {
	logger := s.project.operationLogger(
		"CreateBTSIssue",
		slog.Int64("integrationId", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.CreateIssueWithResponse(
		ctx,
		s.project.projectName,
		integrationID,
		request,
	)

	return execute(ctx, logger, "create bts issue", response, err)
}

// GetTicketIds returns ticket IDs matching the given term within a launch.
func (s *BugTrackingScope) GetTicketIds(
	ctx context.Context,
	launchID int64,
	term string,
) ([]string, error) {
	logger := s.project.operationLogger("GetBTSTicketIds",
		slog.Int64("launchId", launchID),
		slog.String("term", term),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.GetTicketIdsWithResponse(
		ctx,
		s.project.projectName,
		&api.GetTicketIdsParams{
			Launch: launchID,
			Term:   term,
		},
	)

	result, err := execute(ctx, logger, "get bts ticket ids", response, err)
	if err != nil {
		return nil, err
	}

	return *result, nil
}

// GetTicketIdsForProject returns ticket IDs matching the given term across the project.
func (s *BugTrackingScope) GetTicketIdsForProject(
	ctx context.Context,
	term string,
) ([]string, error) {
	logger := s.project.operationLogger("GetBTSTicketIdsForProject",
		slog.String("term", term),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.GetTicketIdsForProjectWithResponse(
		ctx,
		s.project.projectName,
		&api.GetTicketIdsForProjectParams{
			Term: term,
		},
	)

	result, err := execute(ctx, logger, "get bts ticket ids for project", response, err)
	if err != nil {
		return nil, err
	}

	return *result, nil
}
