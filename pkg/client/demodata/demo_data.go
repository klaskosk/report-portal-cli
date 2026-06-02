package demodata

import (
	"context"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// DemoDataScope provides operations for generating demo data within a project.
type DemoDataScope struct {
	scope core.Scope
}

// Generate creates demo data (launches, items, dashboards) in the project.
// Returns the IDs of the created resources.
func (s *DemoDataScope) Generate(
	ctx context.Context,
	demoDataRequest types.DemoDataRq,
) (*types.DemoDataRs, error) {
	logger := s.scope.OperationLogger("GenerateDemoData")
	logger.InfoContext(ctx, "API request")

	response, err := s.scope.Client.GenerateWithResponse(
		ctx,
		s.scope.ProjectName,
		demoDataRequest,
	)

	return core.Execute(ctx, logger, "generate demo data", response, err)
}

// NewDemoDataScope constructs a DemoDataScope for wiring from the facade.
func NewDemoDataScope(s core.Scope) *DemoDataScope {
	return &DemoDataScope{scope: s}
}
