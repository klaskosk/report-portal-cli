package launch_test

import "github.com/klaskosk/report-portal-cli/pkg/internal/api"

// These compile-time-only functions verify that the three launch param
// types remain structurally identical. If a spec regeneration adds or
// removes a field from only one type, the conversion will fail to compile.
var _ = func(p api.GetProjectLaunchesParams) api.GetLatestLaunchesParams {
	return api.GetLatestLaunchesParams(p)
}

var _ = func(p api.GetProjectLaunchesParams) api.GetDebugLaunchesParams {
	return api.GetDebugLaunchesParams(p)
}
