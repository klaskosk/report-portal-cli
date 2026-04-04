package client

import "github.com/klaskosk/report-portal-cli/pkg/types"

// NewTestAPIError exposes newAPIError for external test packages.
var NewTestAPIError = func(statusCode int, resp *types.ErrorRS) error {
	return newAPIError(statusCode, resp)
}

// NewTestUnexpectedStatusError exposes newUnexpectedStatusError for external test packages.
var NewTestUnexpectedStatusError = func(statusCode int) error {
	return newUnexpectedStatusError(statusCode, nil)
}

// NewTestActivityScope creates an ActivityScope for testing without requiring a full Client setup.
var NewTestActivityScope = func(c *Client, projectName string) *ActivityScope {
	return c.Project(projectName).Activities()
}

// NewTestActivityEventsScope creates an ActivityEventsScope for testing without requiring a full Client setup.
var NewTestActivityEventsScope = func(c *Client) *ActivityEventsScope {
	return c.ActivityEvents()
}
