package client

import (
	"context"
	"fmt"
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

	response, err := s.project.client.raw.GetSetOfIntegrationSystemFieldsUsingGET1WithResponse(
		ctx,
		s.project.projectName,
		integrationID,
		&api.GetSetOfIntegrationSystemFieldsUsingGET1Params{},
	)
	if err != nil {
		return nil, fmt.Errorf("get bts system fields: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("get bts system fields: %w", err)
	}

	if result == nil {
		return []types.PostFormField{}, nil
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

	response, err := s.project.client.raw.GetAllowableIssueTypesUsingGET1WithResponse(
		ctx,
		s.project.projectName,
		integrationID,
	)
	if err != nil {
		return nil, fmt.Errorf("get bts allowable issue types: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("get bts allowable issue types: %w", err)
	}

	if result == nil {
		return []string{}, nil
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

	response, err := s.project.client.raw.GetTicketUsingGETWithResponse(
		ctx,
		s.project.projectName,
		ticketID,
		&api.GetTicketUsingGETParams{
			BtsUrl:     btsURL,
			BtsProject: btsProject,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get bts ticket: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("get bts ticket: %w", err)
	}

	return result, nil
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

	response, err := s.project.client.raw.CreateIssueUsingPOSTWithResponse(
		ctx,
		s.project.projectName,
		integrationID,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create bts issue: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON201)
	if err != nil {
		return nil, fmt.Errorf("create bts issue: %w", err)
	}

	return result, nil
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

	response, err := s.project.client.raw.GetTicketIdsUsingGETWithResponse(
		ctx,
		s.project.projectName,
		&api.GetTicketIdsUsingGETParams{
			Launch: launchID,
			Term:   term,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get bts ticket ids: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("get bts ticket ids: %w", err)
	}

	if result == nil {
		return []string{}, nil
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

	response, err := s.project.client.raw.GetTicketIdsForProjectUsingGETWithResponse(
		ctx,
		s.project.projectName,
		&api.GetTicketIdsForProjectUsingGETParams{
			Term: term,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get bts ticket ids for project: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("get bts ticket ids for project: %w", err)
	}

	if result == nil {
		return []string{}, nil
	}

	return *result, nil
}
