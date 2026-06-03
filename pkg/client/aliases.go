package client

import (
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
	"github.com/klaskosk/report-portal-cli/pkg/client/onboarding"
	"github.com/klaskosk/report-portal-cli/pkg/client/plugin"
	"github.com/klaskosk/report-portal-cli/pkg/client/projects"
	"github.com/klaskosk/report-portal-cli/pkg/client/settings"
	"github.com/klaskosk/report-portal-cli/pkg/client/users"
	"github.com/klaskosk/report-portal-cli/pkg/client/widget"
)

type (
	APIError                         = core.APIError
	UnexpectedStatusError            = core.UnexpectedStatusError
	ActivityEventsScope              = activity.ActivityEventsScope
	ActivityScope                    = activity.ActivityScope
	BugTrackingScope                 = bts.BugTrackingScope
	DashboardScope                   = dashboard.DashboardScope
	DemoDataScope                    = demodata.DemoDataScope
	FileStorageScope                 = files.FileStorageScope
	GroupsScope                      = groups.GroupsScope
	InstanceBugTrackingScope         = bts.InstanceBugTrackingScope
	IntegrationsScope                = integrations.IntegrationsScope
	LaunchScope                      = launch.LaunchScope
	LogScope                         = log.LogScope
	LogTypesScope                    = logtypes.LogTypesScope
	OnboardingScope                  = onboarding.OnboardingScope
	PluginScope                      = plugin.PluginScope
	ProjectFilesScope                = files.ProjectFilesScope
	ProjectGroupsScope               = groups.ProjectGroupsScope
	ProjectIntegrationsScope         = integrations.ProjectIntegrationsScope
	ProjectSettingsScope             = settings.ProjectSettingsScope
	ProjectsScope                    = projects.ProjectsScope
	SettingsScope                    = settings.SettingsScope
	TestItemScope                    = item.TestItemScope
	UserFilterScope                  = filter.UserFilterScope
	UsersScope                       = users.UsersScope
	WidgetScope                      = widget.WidgetScope
	ExportProjectsOption             = projects.ExportProjectsOption
	GetAttributeKeysForProjectOption = item.GetAttributeKeysForProjectOption
	GetLogLocationsOption            = log.GetLogLocationsOption
	GetLogPageNumberOption           = log.GetLogPageNumberOption
	GetMyPhotoOption                 = files.GetMyPhotoOption
	GetNestedLogsOption              = log.GetNestedLogsOption
	GetProjectInfoOption             = projects.GetProjectInfoOption
	GetUserPhotoOption               = files.GetUserPhotoOption
	GetWidgetByCodeOption            = projects.GetWidgetByCodeOption
	GetWidgetDataOption              = widget.GetWidgetDataOption
	HistoryOption                    = item.HistoryOption
	ListActivitiesOption             = activity.ListActivitiesOption
	ListAssignableUsersOption        = projects.ListAssignableUsersOption
	ListClustersOption               = launch.ListClustersOption
	ListDashboardsOption             = dashboard.ListDashboardsOption
	ListFiltersOption                = filter.ListFiltersOption
	ListGroupProjectsOption          = groups.ListGroupProjectsOption
	ListGroupUsersOption             = groups.ListGroupUsersOption
	ListGroupsOption                 = groups.ListGroupsOption
	ListLaunchesOption               = launch.ListLaunchesOption
	ListLogsOption                   = log.ListLogsOption
	ListProjectGroupsOption          = groups.ListProjectGroupsOption
	ListProjectUsersOption           = projects.ListProjectUsersOption
	ListProjectsOption               = projects.ListProjectsOption
	ListTestItemsOption              = item.ListTestItemsOption
	ListTestItemsV2Option            = item.ListTestItemsV2Option
	ListUsersOption                  = users.ListUsersOption
	SearchActivityEventsOption       = activity.SearchActivityEventsOption
	SearchLogLocationsOption         = log.SearchLogLocationsOption
	SearchTestItemsOption            = item.SearchTestItemsOption
	SearchUsernamesOption            = projects.SearchUsernamesOption
	SearchUsersOption                = users.SearchUsersOption
	StatisticsOption                 = item.StatisticsOption
	BatchLogEntry                    = log.BatchLogEntry
	EventSortOrder                   = activity.EventSortOrder
	FileContent                      = files.FileContent
)

var (
	IsNotFound                       = core.IsNotFound
	IsUnauthorized                   = core.IsUnauthorized
	IsForbidden                      = core.IsForbidden
	IsBadRequest                     = core.IsBadRequest
	IsConflict                       = core.IsConflict
	IsServerError                    = core.IsServerError
	HasStatusCode                    = core.HasStatusCode
	HasErrorCode                     = core.HasErrorCode
	BodyOf                           = core.BodyOf
	WithActivityPageNumber           = activity.WithActivityPageNumber
	WithActivityPageSize             = activity.WithActivityPageSize
	WithActivitySort                 = activity.WithActivitySort
	WithAssignableUserPageNumber     = projects.WithAssignableUserPageNumber
	WithAssignableUserPageSize       = projects.WithAssignableUserPageSize
	WithAssignableUserSort           = projects.WithAssignableUserSort
	WithClusterPageNumber            = launch.WithClusterPageNumber
	WithClusterPageSize              = launch.WithClusterPageSize
	WithClusterSort                  = launch.WithClusterSort
	WithDashboardPageNumber          = dashboard.WithDashboardPageNumber
	WithDashboardPageSize            = dashboard.WithDashboardPageSize
	WithDashboardSort                = dashboard.WithDashboardSort
	WithEventLimit                   = activity.WithEventLimit
	WithEventOffset                  = activity.WithEventOffset
	WithEventOrder                   = activity.WithEventOrder
	WithEventSort                    = activity.WithEventSort
	WithExportFilterLastRun          = projects.WithExportFilterLastRun
	WithExportFilterName             = projects.WithExportFilterName
	WithExportFilterType             = projects.WithExportFilterType
	WithExportView                   = projects.WithExportView
	WithFilterPageNumber             = filter.WithFilterPageNumber
	WithFilterPageSize               = filter.WithFilterPageSize
	WithFilterSort                   = filter.WithFilterSort
	WithGroupOffset                  = groups.WithGroupOffset
	WithGroupOrder                   = groups.WithGroupOrder
	WithGroupPageSize                = groups.WithGroupPageSize
	WithGroupProjectOffset           = groups.WithGroupProjectOffset
	WithGroupProjectPageSize         = groups.WithGroupProjectPageSize
	WithGroupSort                    = groups.WithGroupSort
	WithGroupUserOffset              = groups.WithGroupUserOffset
	WithGroupUserPageSize            = groups.WithGroupUserPageSize
	WithHistoryDepth                 = item.WithHistoryDepth
	WithHistoryFilterID              = item.WithHistoryFilterID
	WithHistoryItemName              = item.WithHistoryItemName
	WithHistoryItemStatus            = item.WithHistoryItemStatus
	WithHistoryLatest                = item.WithHistoryLatest
	WithHistoryLaunchID              = item.WithHistoryLaunchID
	WithHistoryLaunchesLimit         = item.WithHistoryLaunchesLimit
	WithHistoryPageNumber            = item.WithHistoryPageNumber
	WithHistoryPageSize              = item.WithHistoryPageSize
	WithHistoryType                  = item.WithHistoryType
	WithItemHasChildren              = item.WithItemHasChildren
	WithItemLaunchID                 = item.WithItemLaunchID
	WithItemName                     = item.WithItemName
	WithItemPageNumber               = item.WithItemPageNumber
	WithItemPageSize                 = item.WithItemPageSize
	WithItemParentID                 = item.WithItemParentID
	WithItemPath                     = item.WithItemPath
	WithItemSort                     = item.WithItemSort
	WithItemStatus                   = item.WithItemStatus
	WithItemType                     = item.WithItemType
	WithItemV2LaunchID               = item.WithItemV2LaunchID
	WithItemV2Name                   = item.WithItemV2Name
	WithItemV2PageNumber             = item.WithItemV2PageNumber
	WithItemV2PageSize               = item.WithItemV2PageSize
	WithItemV2ParentID               = item.WithItemV2ParentID
	WithItemV2Sort                   = item.WithItemV2Sort
	WithItemV2Status                 = item.WithItemV2Status
	WithLaunchAttribute              = launch.WithLaunchAttribute
	WithLaunchAttributeKey           = launch.WithLaunchAttributeKey
	WithLaunchDescription            = launch.WithLaunchDescription
	WithLaunchHasRetries             = launch.WithLaunchHasRetries
	WithLaunchMode                   = launch.WithLaunchMode
	WithLaunchName                   = launch.WithLaunchName
	WithLaunchNumber                 = launch.WithLaunchNumber
	WithLaunchPageNumber             = launch.WithLaunchPageNumber
	WithLaunchPageSize               = launch.WithLaunchPageSize
	WithLaunchSort                   = launch.WithLaunchSort
	WithLaunchStatus                 = launch.WithLaunchStatus
	WithLaunchUser                   = launch.WithLaunchUser
	WithLaunchUserID                 = launch.WithLaunchUserID
	WithLogItemID                    = log.WithLogItemID
	WithLogLaunchID                  = log.WithLogLaunchID
	WithLogLevel                     = log.WithLogLevel
	WithLogLocationsPageNumber       = log.WithLogLocationsPageNumber
	WithLogLocationsPageSize         = log.WithLogLocationsPageSize
	WithLogPageItemID                = log.WithLogPageItemID
	WithLogPageNumber                = log.WithLogPageNumber
	WithLogPageNumberPageSize        = log.WithLogPageNumberPageSize
	WithLogPageNumberSort            = log.WithLogPageNumberSort
	WithLogPageSize                  = log.WithLogPageSize
	WithLogSort                      = log.WithLogSort
	WithNestedLogsPageNumber         = log.WithNestedLogsPageNumber
	WithNestedLogsPageSize           = log.WithNestedLogsPageSize
	WithPhotoThumbnail               = files.WithPhotoThumbnail
	WithProjectGroupPageOffset       = groups.WithProjectGroupPageOffset
	WithProjectGroupPageSize         = groups.WithProjectGroupPageSize
	WithProjectInfoInterval          = projects.WithProjectInfoInterval
	WithProjectKeysLatest            = item.WithProjectKeysLatest
	WithProjectKeysLaunchesLimit     = item.WithProjectKeysLaunchesLimit
	WithProjectName                  = projects.WithProjectName
	WithProjectPageNumber            = projects.WithProjectPageNumber
	WithProjectPageSize              = projects.WithProjectPageSize
	WithProjectSort                  = projects.WithProjectSort
	WithProjectType                  = projects.WithProjectType
	WithSearchItemAttribute          = item.WithSearchItemAttribute
	WithSearchItemName               = item.WithSearchItemName
	WithSearchItemPageNumber         = item.WithSearchItemPageNumber
	WithSearchItemPageSize           = item.WithSearchItemPageSize
	WithSearchItemSort               = item.WithSearchItemSort
	WithSearchItemStatus             = item.WithSearchItemStatus
	WithSearchLogLocationsPageNumber = log.WithSearchLogLocationsPageNumber
	WithSearchLogLocationsPageSize   = log.WithSearchLogLocationsPageSize
	WithSearchUsernamesPageNumber    = projects.WithSearchUsernamesPageNumber
	WithSearchUsernamesPageSize      = projects.WithSearchUsernamesPageSize
	WithSearchUsernamesSort          = projects.WithSearchUsernamesSort
	WithSearchUsersPageNumber        = users.WithSearchUsersPageNumber
	WithSearchUsersPageSize          = users.WithSearchUsersPageSize
	WithSearchUsersSort              = users.WithSearchUsersSort
	WithStatisticsHasChildren        = item.WithStatisticsHasChildren
	WithStatisticsLaunchID           = item.WithStatisticsLaunchID
	WithStatisticsName               = item.WithStatisticsName
	WithStatisticsParentID           = item.WithStatisticsParentID
	WithStatisticsStatus             = item.WithStatisticsStatus
	WithStatisticsType               = item.WithStatisticsType
	WithUserPageNumber               = projects.WithUserPageNumber
	WithUserPageSize                 = projects.WithUserPageSize
	WithUserPhotoThumbnail           = files.WithUserPhotoThumbnail
	WithUserSort                     = projects.WithUserSort
	WithUsersActive                  = users.WithUsersActive
	WithUsersEmail                   = users.WithUsersEmail
	WithUsersPageNumber              = users.WithUsersPageNumber
	WithUsersPageSize                = users.WithUsersPageSize
	WithUsersRole                    = users.WithUsersRole
	WithUsersSort                    = users.WithUsersSort
	WithUsersType                    = users.WithUsersType
	WithWidgetAttributes             = widget.WithWidgetAttributes
	WithWidgetInterval               = projects.WithWidgetInterval
	NewEventSearchCriteria           = activity.NewEventSearchCriteria

	EventFilterByUser        = activity.EventFilterByUser
	EventFilterByObjectType  = activity.EventFilterByObjectType
	EventFilterByEventName   = activity.EventFilterByEventName
	EventFilterByProjectName = activity.EventFilterByProjectName

	ParseErrorBody = core.ParseErrorBody
)

const (
	EventOrderAsc  = activity.EventOrderAsc
	EventOrderDesc = activity.EventOrderDesc
)
