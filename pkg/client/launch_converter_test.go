package client_test

import "github.com/klaskosk/report-portal-cli/pkg/internal/api"

// These compile-time-only functions verify that the three launch param
// types remain structurally identical. If a spec regeneration adds or
// removes a field from only one type, the conversion will fail to compile.
var _ = func(p api.GetProjectLaunchesUsingGETParams) api.GetLatestLaunchesUsingGETParams {
	return api.GetLatestLaunchesUsingGETParams(p)
}

var _ = func(p api.GetProjectLaunchesUsingGETParams) api.GetDebugLaunchesUsingGETParams {
	return api.GetDebugLaunchesUsingGETParams(p)
}
