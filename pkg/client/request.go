package client

import (
	"context"
	"net/http"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
)

const (
	acceptReportPortalLaunchV2 = "application/x.reportportal.launch.v2+json"
	acceptReportPortalTestV2   = "application/x.reportportal.test.v2+json"
)

// withAcceptHeader returns a request editor that sets the Accept header for endpoints
// that offer a more stable versioned response format.
func withAcceptHeader(value string) api.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		req.Header.Set("Accept", value)

		return nil
	}
}
