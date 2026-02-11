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

func (p *ProjectScope) logger() *slog.Logger {
	return p.client.logger.With("project", p.projectName)
}
