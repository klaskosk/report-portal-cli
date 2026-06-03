package client

import (
	"context"

	"github.com/klaskosk/report-portal-cli/pkg/client/activity"
	"github.com/klaskosk/report-portal-cli/pkg/client/bts"
	"github.com/klaskosk/report-portal-cli/pkg/client/dashboard"
	"github.com/klaskosk/report-portal-cli/pkg/client/demodata"
	"github.com/klaskosk/report-portal-cli/pkg/client/files"
	"github.com/klaskosk/report-portal-cli/pkg/client/filter"
	"github.com/klaskosk/report-portal-cli/pkg/client/groups"
	"github.com/klaskosk/report-portal-cli/pkg/client/integrations"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/client/item"
	"github.com/klaskosk/report-portal-cli/pkg/client/launch"
	"github.com/klaskosk/report-portal-cli/pkg/client/log"
	"github.com/klaskosk/report-portal-cli/pkg/client/logtypes"
	"github.com/klaskosk/report-portal-cli/pkg/client/settings"
	"github.com/klaskosk/report-portal-cli/pkg/client/widget"
)

// ProjectScope provides access to resources within a specific Report Portal project.
type ProjectScope struct {
	scope core.Scope
}

// Project returns a ProjectScope for the named project. It never returns nil.
func (c *Client) Project(name string) *ProjectScope {
	return &ProjectScope{scope: c.scope().WithProject(name)}
}

// Launches returns a LaunchScope for querying launches in this project.
func (p *ProjectScope) Launches() *LaunchScope {
	return launch.NewLaunchScope(p.scope)
}

// Items returns a TestItemScope for querying test items in this project.
func (p *ProjectScope) Items() *TestItemScope {
	return item.NewItemScope(p.scope)
}

// Logs returns a LogScope for querying log entries in this project.
func (p *ProjectScope) Logs() *LogScope {
	return log.NewLogScope(p.scope)
}

// Dashboards returns a DashboardScope for managing dashboards in this project.
func (p *ProjectScope) Dashboards() *DashboardScope {
	return dashboard.NewDashboardScope(p.scope)
}

// Widgets returns a WidgetScope for managing widgets in this project.
func (p *ProjectScope) Widgets() *WidgetScope {
	return widget.NewWidgetScope(p.scope)
}

// Filters returns a UserFilterScope for managing user filters in this project.
func (p *ProjectScope) Filters() *UserFilterScope {
	return filter.NewUserFilterScope(p.scope)
}

// Integrations returns a ProjectIntegrationsScope for managing integrations in this project.
func (p *ProjectScope) Integrations() *ProjectIntegrationsScope {
	return integrations.NewProjectIntegrationsScope(p.scope)
}

// BugTracking returns a BugTrackingScope for BTS operations in this project.
func (p *ProjectScope) BugTracking() *BugTrackingScope {
	return bts.NewBugTrackingScope(p.scope)
}

// DemoData returns a DemoDataScope for generating demo data in this project.
func (p *ProjectScope) DemoData() *DemoDataScope {
	return demodata.NewDemoDataScope(p.scope)
}

// Activities returns an ActivityScope for querying activity logs in this project.
func (p *ProjectScope) Activities() *ActivityScope {
	return activity.NewActivityScope(p.scope)
}

// Groups returns a ProjectGroupsScope for managing group-project associations in this project.
func (p *ProjectScope) Groups() *ProjectGroupsScope {
	return groups.NewProjectGroupsScope(p.scope)
}

// LogTypes returns a LogTypesScope for managing log types in this project.
func (p *ProjectScope) LogTypes() *LogTypesScope {
	return logtypes.NewLogTypesScope(p.scope)
}

// Settings returns a ProjectSettingsScope for reading configuration of this project.
func (p *ProjectScope) Settings() *ProjectSettingsScope {
	return settings.NewProjectSettingsScope(p.scope)
}

// GetUserPhoto retrieves a user's profile photo by login within this project.
func (p *ProjectScope) GetUserPhoto(
	ctx context.Context,
	login string,
	options ...GetUserPhotoOption,
) (*FileContent, error) {
	return files.NewProjectFilesScope(p.scope).GetUserPhoto(ctx, login, options...)
}

// DownloadFile downloads a file attachment by its numeric data store ID.
func (p *ProjectScope) DownloadFile(
	ctx context.Context,
	dataID int64,
) (*FileContent, error) {
	return files.NewProjectFilesScope(p.scope).DownloadFile(ctx, dataID)
}
