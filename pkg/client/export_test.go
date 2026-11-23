package client

import (
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// FailingOption returns an Option that always returns the given error. Test-only.
var FailingOption = func(err error) Option {
	return func(_ *config) error {
		return err
	}
}

// NewTestAPIError exposes core.NewAPIError for external test packages.
var NewTestAPIError = func(statusCode int, resp *types.ErrorRS) error {
	return core.NewAPIError(statusCode, resp)
}

// NewTestUnexpectedStatusError exposes core.NewUnexpectedStatusError for external test packages.
var NewTestUnexpectedStatusError = func(statusCode int) error {
	return core.NewUnexpectedStatusError(statusCode, nil)
}

// NewTestUnexpectedStatusErrorWithBody exposes core.NewUnexpectedStatusError with a
// body for external test packages.
var NewTestUnexpectedStatusErrorWithBody = func(statusCode int, body []byte) error {
	return core.NewUnexpectedStatusError(statusCode, body)
}

// NewTestActivityScope creates an ActivityScope for testing without requiring a full Client setup.
var NewTestActivityScope = func(c *Client, projectName string) *ActivityScope {
	return c.Project(projectName).Activities()
}

// NewTestActivityEventsScope creates an ActivityEventsScope for testing without requiring a full Client setup.
var NewTestActivityEventsScope = func(c *Client) *ActivityEventsScope {
	return c.ActivityEvents()
}

// NewTestInstanceBugTrackingScope returns an InstanceBugTrackingScope for tests
// without constructing a full Client.
var NewTestInstanceBugTrackingScope = func(c *Client) *InstanceBugTrackingScope {
	return c.InstanceBugTracking()
}

// NewTestProjectGroupsScope creates a ProjectGroupsScope for testing without requiring a full Client setup.
var NewTestProjectGroupsScope = func(c *Client, projectName string) *ProjectGroupsScope {
	return c.Project(projectName).Groups()
}

// NewTestLogTypesScope creates a LogTypesScope for testing without requiring a full Client setup.
var NewTestLogTypesScope = func(c *Client, projectName string) *LogTypesScope {
	return c.Project(projectName).LogTypes()
}
