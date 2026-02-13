package client

import "log/slog"

// ProjectScope provides access to resources within a specific Report Portal project.
type ProjectScope struct {
	client      *Client
	projectName string
}

// Project returns a ProjectScope for the named project. It never returns nil.
func (c *Client) Project(name string) *ProjectScope {
	return &ProjectScope{client: c, projectName: name}
}

// Launches returns a LaunchScope for querying launches in this project.
func (p *ProjectScope) Launches() *LaunchScope {
	return &LaunchScope{project: p}
}

// Items returns a TestItemScope for querying test items in this project.
func (p *ProjectScope) Items() *TestItemScope {
	return &TestItemScope{project: p}
}

// Logs returns a LogScope for querying log entries in this project.
func (p *ProjectScope) Logs() *LogScope {
	return &LogScope{project: p}
}

// Dashboards returns a DashboardScope for managing dashboards in this project.
func (p *ProjectScope) Dashboards() *DashboardScope {
	return &DashboardScope{project: p}
}

// Widgets returns a WidgetScope for managing widgets in this project.
func (p *ProjectScope) Widgets() *WidgetScope {
	return &WidgetScope{project: p}
}

// Filters returns a UserFilterScope for managing user filters in this project.
func (p *ProjectScope) Filters() *UserFilterScope {
	return &UserFilterScope{project: p}
}

// logger returns a logger scoped to this project.
func (p *ProjectScope) logger() *slog.Logger {
	return p.client.logger.With("project", p.projectName)
}
