package client

import "github.com/klaskosk/report-portal-cli/pkg/types"

// NewTestAPIError exposes newAPIError for external test packages.
var NewTestAPIError = func(operation string, statusCode int, resp *types.ErrorRS) error {
	return newAPIError(operation, statusCode, resp)
}
