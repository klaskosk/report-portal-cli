package client

import (
	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// NewTestAPIError exposes newAPIError for external test packages.
var NewTestAPIError = func(operation string, statusCode int, resp *types.ErrorRS) error {
	return newAPIError(operation, statusCode, resp)
}

// NewTestUnexpectedStatusError exposes newUnexpectedStatusError for external test packages.
var NewTestUnexpectedStatusError = func(operation string, statusCode int) error {
	return newUnexpectedStatusError(operation, statusCode)
}

// ConvertToLatestParams exposes convertToLatestParams for external test packages.
var ConvertToLatestParams = func(src *api.GetProjectLaunchesUsingGETParams) *api.GetLatestLaunchesUsingGETParams {
	return convertToLatestParams(src)
}

// ConvertToDebugParams exposes convertToDebugParams for external test packages.
var ConvertToDebugParams = func(src *api.GetProjectLaunchesUsingGETParams) *api.GetDebugLaunchesUsingGETParams {
	return convertToDebugParams(src)
}

// NewTestActivityScope creates an ActivityScope for testing without requiring a full Client setup.
var NewTestActivityScope = func(c *Client, projectName string) *ActivityScope {
	return c.Project(projectName).Activities()
}

// NewTestActivityEventsScope creates an ActivityEventsScope for testing without requiring a full Client setup.
var NewTestActivityEventsScope = func(c *Client) *ActivityEventsScope {
	return c.ActivityEvents()
}
