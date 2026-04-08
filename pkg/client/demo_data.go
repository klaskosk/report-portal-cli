package client

import (
	"context"

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

	response, err := s.project.client.raw.GenerateWithResponse(
		ctx,
		s.project.projectName,
		demoDataRequest,
	)

	return execute(ctx, logger, "generate demo data", response, err)
}
