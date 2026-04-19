//go:build integration

package integration_test

import (
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Activity Events", Label("client", "activity-events"), func() {
	var c *client.Client
	var projectName string

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_activity_events_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding launches to generate activity events")
		err = seedTestLaunches(ctx, baseURL, bearerToken, projectName, 2)
		Expect(err).NotTo(HaveOccurred(), "seeding test launches should succeed")
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, c, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("Search", func() {
		It("returns events when searching with empty criteria", func(ctx SpecContext) {
			activityEvents := c.ActivityEvents()

			var searchResult *types.PagedResponseActivityEventResource
			var lastSearchErr error

			waitForSeedResolution(ctx, activityResolutionAttempts, activityResolutionInterval, func() bool {
				searchResult, lastSearchErr = activityEvents.Search(
					ctx,
					types.SearchCriteriaRQ{},
					client.WithEventLimit(50),
					client.WithEventOffset(0),
					client.WithEventSort("createdAt"),
					client.WithEventOrder(client.EventOrderDesc),
				)
				if lastSearchErr != nil {
					return false
				}
				lastSearchErr = nil

				return searchResult != nil && len(searchResult.Items) > 0
			})

			Expect(lastSearchErr).NotTo(HaveOccurred(),
				"search should succeed while waiting for activity events")
			Expect(searchResult).NotTo(BeNil(),
				"search result should not be nil")
			Expect(searchResult.Items).NotTo(BeEmpty(),
				"search result should contain at least one event")

			firstEvent := searchResult.Items[0]
			Expect(firstEvent.EventName).NotTo(BeEmpty(),
				"event should have a non-empty event name")
			Expect(firstEvent.ObjectType).NotTo(BeEmpty(),
				"event should have a non-empty object type")
		}, NodeTimeout(120*time.Second))

		It("filters events by project name", func(ctx SpecContext) {
			activityEvents := c.ActivityEvents()
			searchCriteria := client.NewEventSearchCriteria(
				client.EventFilterByProjectName(projectName),
			)

			var searchResult *types.PagedResponseActivityEventResource
			var lastSearchErr error

			waitForSeedResolution(ctx, activityResolutionAttempts, activityResolutionInterval, func() bool {
				searchResult, lastSearchErr = activityEvents.Search(
					ctx,
					searchCriteria,
					client.WithEventLimit(50),
					client.WithEventOffset(0),
					client.WithEventSort("createdAt"),
					client.WithEventOrder(client.EventOrderDesc),
				)
				if lastSearchErr != nil {
					return false
				}
				lastSearchErr = nil

				return searchResult != nil && len(searchResult.Items) > 0
			})

			Expect(lastSearchErr).NotTo(HaveOccurred())
			Expect(searchResult).NotTo(BeNil())
			Expect(searchResult.Items).NotTo(BeEmpty(),
				"filtered search should return events for project %q", projectName)

			for _, event := range searchResult.Items {
				Expect(event.ProjectName).To(Equal(projectName),
					"all returned events should belong to project %q", projectName)
			}
		}, NodeTimeout(120*time.Second))

		It("returns events matching a seeded event-name filter", func(ctx SpecContext) {
			activityEvents := c.ActivityEvents()
			baseCriteria := client.NewEventSearchCriteria(client.EventFilterByProjectName(projectName))

			var projectEvents *types.PagedResponseActivityEventResource
			var lastSearchErr error
			eventName := ""

			waitForSeedResolution(ctx, activityResolutionAttempts, activityResolutionInterval, func() bool {
				projectEvents, lastSearchErr = activityEvents.Search(
					ctx,
					baseCriteria,
					client.WithEventLimit(50),
					client.WithEventOffset(0),
					client.WithEventSort("createdAt"),
					client.WithEventOrder(client.EventOrderDesc),
				)
				if lastSearchErr != nil {
					return false
				}
				lastSearchErr = nil

				if projectEvents == nil || len(projectEvents.Items) == 0 {
					return false
				}

				for _, event := range projectEvents.Items {
					if event.EventName != "" {
						eventName = event.EventName
						return true
					}
				}

				return false
			})

			Expect(lastSearchErr).NotTo(HaveOccurred())
			Expect(eventName).NotTo(BeEmpty(), "should resolve an event name from project activity data")

			filteredCriteria := client.NewEventSearchCriteria(
				client.EventFilterByProjectName(projectName),
				client.EventFilterByEventName(eventName),
			)
			var filteredResult *types.PagedResponseActivityEventResource

			waitForSeedResolution(ctx, activityResolutionAttempts, activityResolutionInterval, func() bool {
				filteredResult, lastSearchErr = activityEvents.Search(
					ctx,
					filteredCriteria,
					client.WithEventLimit(50),
					client.WithEventOffset(0),
					client.WithEventSort("createdAt"),
					client.WithEventOrder(client.EventOrderDesc),
				)
				if lastSearchErr != nil {
					return false
				}
				lastSearchErr = nil

				return filteredResult != nil && len(filteredResult.Items) > 0
			})

			Expect(lastSearchErr).NotTo(HaveOccurred())
			Expect(filteredResult).NotTo(BeNil())
			Expect(filteredResult.Items).NotTo(BeEmpty(),
				"filtered search should return events for eventName=%q", eventName)

			for _, event := range filteredResult.Items {
				Expect(event.EventName).To(Equal(eventName),
					"filtered events should all match event name %q", eventName)
				Expect(event.ProjectName).To(Equal(projectName),
					"filtered events should all belong to project %q", projectName)
			}
		}, NodeTimeout(120*time.Second))
	})
})
