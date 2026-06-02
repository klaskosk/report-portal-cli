package client

import (
	"github.com/klaskosk/report-portal-cli/pkg/client/activity"
	"github.com/klaskosk/report-portal-cli/pkg/client/bts"
	"github.com/klaskosk/report-portal-cli/pkg/client/files"
	"github.com/klaskosk/report-portal-cli/pkg/client/groups"
	"github.com/klaskosk/report-portal-cli/pkg/client/integrations"
	"github.com/klaskosk/report-portal-cli/pkg/client/onboarding"
	"github.com/klaskosk/report-portal-cli/pkg/client/plugin"
	"github.com/klaskosk/report-portal-cli/pkg/client/projects"
	"github.com/klaskosk/report-portal-cli/pkg/client/settings"
	"github.com/klaskosk/report-portal-cli/pkg/client/users"
)

// Users returns a UsersScope for instance-level user operations.
func (c *Client) Users() *UsersScope {
	return users.NewUsersScope(c.scope())
}

// Projects returns a ProjectsScope for instance-level project operations.
func (c *Client) Projects() *ProjectsScope {
	return projects.NewProjectsScope(c.scope())
}

// Groups returns a GroupsScope for instance-level group operations.
func (c *Client) Groups() *GroupsScope {
	return groups.NewGroupsScope(c.scope())
}

// Integrations returns an IntegrationsScope for instance-level global integration operations.
func (c *Client) Integrations() *IntegrationsScope {
	return integrations.NewIntegrationsScope(c.scope())
}

// Files returns a FileStorageScope for instance-level file storage operations.
func (c *Client) Files() *FileStorageScope {
	return files.NewFileStorageScope(c.scope())
}

// Settings returns a SettingsScope for server-level settings operations.
func (c *Client) Settings() *SettingsScope {
	return settings.NewSettingsScope(c.scope())
}

// InstanceBugTracking returns an InstanceBugTrackingScope for instance-level BTS operations.
func (c *Client) InstanceBugTracking() *InstanceBugTrackingScope {
	return bts.NewInstanceBugTrackingScope(c.scope())
}

// Plugins returns a PluginScope for instance-level plugin operations.
func (c *Client) Plugins() *PluginScope {
	return plugin.NewPluginScope(c.scope())
}

// Onboarding returns an OnboardingScope for first-run onboarding operations.
func (c *Client) Onboarding() *OnboardingScope {
	return onboarding.NewOnboardingScope(c.scope())
}

// ActivityEvents returns an ActivityEventsScope for instance-level activity event queries.
func (c *Client) ActivityEvents() *ActivityEventsScope {
	return activity.NewActivityEventsScope(c.scope())
}
