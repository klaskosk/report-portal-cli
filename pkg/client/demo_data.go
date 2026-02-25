package client

import (
	"context"
	"fmt"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// DemoDataScope provides operations for generating demo data within a project.
type DemoDataScope struct {
	project *ProjectScope
}

// Generate creates demo data (launches, items, dashboards) in the project.
// Returns the IDs of the created resources.
func (s *DemoDataScope) Generate(
	ctx context.Context,
	demoDataRequest types.DemoDataRq,
) (*types.DemoDataRs, error) {
	logger := s.project.operationLogger("GenerateDemoData")
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.GenerateUsingPOSTWithResponse(
		ctx,
		s.project.projectName,
		demoDataRequest,
	)
	if err != nil {
		return nil, fmt.Errorf("generate demo data: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("generate demo data: %w", err)
	}

	return result, nil
}
