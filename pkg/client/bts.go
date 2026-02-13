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
	logger := s.project.operationLogger("GetBTSSystemFields", slog.Int64("integrationId", integrationID))
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

	result, err := handleResponse(response, "get bts system fields", response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, err
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
	logger := s.project.operationLogger("GetBTSAllowableIssueTypes", slog.Int64("integrationId", integrationID))
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

	result, err := handleResponse(response, "get bts allowable issue types", response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, err
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

	return handleResponse(response, "get bts ticket", response.JSONDefault, response.JSON200)
}
