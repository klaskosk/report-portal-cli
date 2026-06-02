package core

import (
	"context"
	"net/http"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
)

const (
	AcceptReportPortalLaunchV2 = "application/x.reportportal.launch.v2+json"
	AcceptReportPortalTestV2   = "application/x.reportportal.test.v2+json"
)

// WithAcceptHeader returns a request editor that sets the Accept header for endpoints
// that offer a more stable versioned response format.
func WithAcceptHeader(value string) api.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		req.Header.Set("Accept", value)

		return nil
	}
}
