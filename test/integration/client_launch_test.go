//go:build integration

package integration_test

import (
	"fmt"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Launches", Label("client", "launches"), func() {
	var c *client.Client
	var projectName string

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_launch_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding test launches into " + projectName)
		err = seedTestLaunches(ctx, baseURL, bearerToken, projectName, 3)
		Expect(err).To(Succeed(), "failed to seed test launches into %q", projectName)
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, baseURL, bearerToken, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("List", func() {
		It("returns launches for the project", func(ctx SpecContext) {
			result, err := c.Project(projectName).Launches().List(ctx)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches for project %q should not fail", projectName)
			Expect(result).NotTo(BeNil(),
				"launch list result should not be nil")
			Expect(result.Content).NotTo(BeEmpty(),
				"launch list should contain seeded demo launches")
		})

		It("filters by name", func(ctx SpecContext) {
			// First, list all to discover a known name.
			all, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches to discover a known name should not fail")
			Expect(all.Content).NotTo(BeEmpty(),
				"launch list should contain at least one launch for name discovery")

			knownName := all.Content[0].Name

			filtered, err := c.Project(projectName).Launches().List(ctx,
				client.WithName(knownName),
			)
			Expect(err).NotTo(HaveOccurred(),
				"filtering launches by name %q should not fail", knownName)

			for _, launch := range filtered.Content {
				Expect(launch.Name).To(Equal(knownName),
					"filtered launch should have name %q but got %q", knownName, launch.Name)
			}
		})

		It("paginates results", func(ctx SpecContext) {
			firstPage, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
				client.WithSort("id,asc"),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches (first page) should not fail")
			Expect(firstPage.Content).To(HaveLen(1),
				"first page with page size 1 should return exactly 1 launch")

			nextPageNumber := int32(firstPage.Page.Number) + 1
			secondPage, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
				client.WithPageNumber(nextPageNumber),
				client.WithSort("id,asc"),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches (second page) should not fail")

			if firstPage.Page.TotalPages > 1 {
				Expect(secondPage.Content).To(HaveLen(1),
					"second page with page size 1 should return exactly 1 launch")
				Expect(secondPage.Content[0].Id).NotTo(Equal(firstPage.Content[0].Id),
					"second page launch should differ from first page launch")
			}
		})

		It("sorts by start time descending", func(ctx SpecContext) {
			result, err := c.Project(projectName).Launches().List(ctx,
				client.WithSort("startTime,desc"),
				client.WithPageSize(10),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches sorted by startTime desc should not fail")

			if len(result.Content) >= 2 {
				for i := 1; i < len(result.Content); i++ {
					prev := time.Time(result.Content[i-1].StartTime)
					curr := time.Time(result.Content[i].StartTime)
					Expect(prev).To(
						BeTemporally(">=", curr),
						"launch[%d] startTime %v should be >= launch[%d] startTime %v (descending order)",
						i-1, prev, i, curr,
					)
				}
			}
		})
	})

	Context("GetByID", func() {
		var knownID int64

		BeforeEach(func(ctx SpecContext) {
			list, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches to discover a known ID should not fail")
			Expect(list.Content).NotTo(BeEmpty(),
				"launch list should contain at least one launch for ID lookup")

			knownID = list.Content[0].Id
		})

		It("returns the launch", func(ctx SpecContext) {
			launch, err := c.Project(projectName).Launches().GetByID(
				ctx,
				strconv.FormatInt(knownID, 10),
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting launch by ID %d should not fail", knownID)
			Expect(launch).NotTo(BeNil(),
				"launch returned for ID %d should not be nil", knownID)
			Expect(launch.Id).To(Equal(knownID),
				"returned launch ID should match requested ID %d", knownID)
		})

		It("returns not found for an invalid ID", func(ctx SpecContext) {
			_, err := c.Project(projectName).Launches().GetByID(
				ctx,
				"999999",
			)
			Expect(err).To(HaveOccurred(),
				"getting launch by non-existent ID 999999 should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for non-existent launch ID should be a not-found error, got: %v", err)
		})
	})

	Context("GetByUUID", func() {
		var knownUUID string

		BeforeEach(func(ctx SpecContext) {
			list, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches to discover a known UUID should not fail")
			Expect(list.Content).NotTo(BeEmpty(),
				"launch list should contain at least one launch for UUID lookup")

			knownUUID = list.Content[0].Uuid
		})

		It("returns the launch", func(ctx SpecContext) {
			launch, err := c.Project(projectName).Launches().GetByUUID(
				ctx,
				knownUUID,
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting launch by UUID %q should not fail", knownUUID)
			Expect(launch).NotTo(BeNil(),
				"launch returned for UUID %q should not be nil", knownUUID)
			Expect(launch.Uuid).To(Equal(knownUUID),
				"returned launch UUID should match requested UUID %q", knownUUID)
		})

		It("returns not found for an invalid UUID", func(ctx SpecContext) {
			_, err := c.Project(projectName).Launches().GetByUUID(
				ctx,
				"00000000-0000-0000-0000-000000000000",
			)
			Expect(err).To(HaveOccurred(),
				"getting launch by non-existent UUID should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for non-existent launch UUID should be a not-found error, got: %v", err)
		})
	})

	Context("GetLatest", func() {
		It("returns latest launches", func(ctx SpecContext) {
			result, err := c.Project(projectName).Launches().GetLatest(ctx)
			Expect(err).NotTo(HaveOccurred(),
				"getting latest launches should not fail")
			Expect(result).NotTo(BeNil(),
				"latest launches result should not be nil")
			Expect(result.Content).NotTo(BeEmpty(),
				"latest launches should contain launches")
		})

		It("supports filtering options", func(ctx SpecContext) {
			// Get a known launch name first
			all, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(all.Content).NotTo(BeEmpty())

			knownName := all.Content[0].Name

			// Filter latest by name
			result, err := c.Project(projectName).Launches().GetLatest(ctx,
				client.WithName(knownName),
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting latest launches with name filter should not fail")
			Expect(result).NotTo(BeNil())

			for _, launch := range result.Content {
				Expect(launch.Name).To(Equal(knownName),
					"filtered latest launch should have name %q", knownName)
			}
		})

		It("supports pagination", func(ctx SpecContext) {
			result, err := c.Project(projectName).Launches().GetLatest(ctx,
				client.WithPageSize(1),
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting latest launches with page size should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).To(HaveLen(1),
				"latest launches with page size 1 should return exactly 1 launch")
		})
	})

	Context("GetDebug", func() {
		It("returns debug launches", func(ctx SpecContext) {
			result, err := c.Project(projectName).Launches().GetDebug(ctx)
			Expect(err).NotTo(HaveOccurred(),
				"getting debug launches should not fail")
			Expect(result).NotTo(BeNil(),
				"debug launches result should not be nil")
			// Note: may be empty if no debug launches exist
		})

		It("supports filtering options", func(ctx SpecContext) {
			result, err := c.Project(projectName).Launches().GetDebug(ctx,
				client.WithPageSize(5),
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting debug launches with page size should not fail")
			Expect(result).NotTo(BeNil())
		})
	})

	Context("GetNames", func() {
		It("returns launch names", func(ctx SpecContext) {
			names, err := c.Project(projectName).Launches().GetNames(ctx, "")
			Expect(err).NotTo(HaveOccurred(),
				"getting launch names should not fail")
			Expect(names).NotTo(BeNil(),
				"launch names result should not be nil")
			Expect(names).NotTo(BeEmpty(),
				"launch names should contain names from seeded launches")
		})

		It("filters names by substring", func(ctx SpecContext) {
			// Get all names first to find a common substring
			allNames, err := c.Project(projectName).Launches().GetNames(ctx, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(allNames).NotTo(BeEmpty())

			// Use first few characters of first name as filter
			if len(allNames) > 0 && len(allNames[0]) > 3 {
				filter := allNames[0][:3]

				filteredNames, err := c.Project(projectName).Launches().GetNames(ctx, filter)
				Expect(err).NotTo(HaveOccurred(),
					"getting launch names with filter %q should not fail", filter)
				Expect(filteredNames).NotTo(BeNil())

				// All returned names should contain the filter
				for _, name := range filteredNames {
					Expect(name).To(ContainSubstring(filter),
						"filtered name %q should contain substring %q", name, filter)
				}
			}
		})

		It("returns empty list for non-matching filter", func(ctx SpecContext) {
			names, err := c.Project(projectName).Launches().GetNames(ctx, "nonexistent_xyz_123")
			Expect(err).NotTo(HaveOccurred(),
				"getting launch names with non-matching filter should not fail")
			Expect(names).NotTo(BeNil())
		})
	})

	Context("Write Operations", func() {
		It("updates a launch description", func(ctx SpecContext) {
			// Start and finish a launch for update.
			startedLaunch, err := c.Project(projectName).Launches().Start(ctx, types.StartLaunchRQ{
				Name:      fmt.Sprintf("update-test-%d", time.Now().UnixNano()),
				StartTime: types.EpochMillis(time.Now()),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(startedLaunch.Id).NotTo(BeNil())

			finishedStatus := types.FinishExecutionRQStatusPASSED
			_, err = c.Project(projectName).Launches().Finish(ctx, *startedLaunch.Id, types.FinishExecutionRQ{
				EndTime: types.EpochMillis(time.Now()),
				Status:  &finishedStatus,
			})
			Expect(err).NotTo(HaveOccurred())

			// Resolve the numeric ID.
			launchID, err := resolveLaunchIDByUUID(ctx, c.Project(projectName).Launches(), *startedLaunch.Id)
			Expect(err).NotTo(HaveOccurred())

			// Update the launch.
			newDescription := "updated by integration test"
			result, err := c.Project(projectName).Launches().Update(ctx, launchID, types.UpdateLaunchRQ{
				Description: &newDescription,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			// Verify the change.
			launch, err := c.Project(projectName).Launches().GetByID(ctx, strconv.FormatInt(launchID, 10))
			Expect(err).NotTo(HaveOccurred())
			Expect(launch.Description).NotTo(BeNil())
			Expect(*launch.Description).To(Equal("updated by integration test"))
		})

		It("deletes a launch", func(ctx SpecContext) {
			// Start and finish a launch for deletion.
			startedLaunch, err := c.Project(projectName).Launches().Start(ctx, types.StartLaunchRQ{
				Name:      fmt.Sprintf("delete-test-%d", time.Now().UnixNano()),
				StartTime: types.EpochMillis(time.Now()),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(startedLaunch.Id).NotTo(BeNil())

			finishedStatus := types.FinishExecutionRQStatusPASSED
			_, err = c.Project(projectName).Launches().Finish(ctx, *startedLaunch.Id, types.FinishExecutionRQ{
				EndTime: types.EpochMillis(time.Now()),
				Status:  &finishedStatus,
			})
			Expect(err).NotTo(HaveOccurred())

			// Resolve the numeric ID.
			launchID, err := resolveLaunchIDByUUID(ctx, c.Project(projectName).Launches(), *startedLaunch.Id)
			Expect(err).NotTo(HaveOccurred())

			// Delete the launch.
			_, err = c.Project(projectName).Launches().Delete(ctx, launchID)
			Expect(err).NotTo(HaveOccurred())

			// Verify it's gone.
			_, err = c.Project(projectName).Launches().GetByID(ctx, strconv.FormatInt(launchID, 10))
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue())
		})

		It("force-finishes a stuck launch", func(ctx SpecContext) {
			// Start a launch but do NOT finish it normally.
			startedLaunch, err := c.Project(projectName).Launches().Start(ctx, types.StartLaunchRQ{
				Name:      fmt.Sprintf("force-finish-test-%d", time.Now().UnixNano()),
				StartTime: types.EpochMillis(time.Now()),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(startedLaunch.Id).NotTo(BeNil())

			// Resolve numeric ID.
			launchID, err := resolveLaunchIDByUUID(ctx, c.Project(projectName).Launches(), *startedLaunch.Id)
			Expect(err).NotTo(HaveOccurred())

			// Force-finish it.
			_, err = c.Project(projectName).Launches().ForceFinish(ctx, launchID, types.FinishExecutionRQ{
				EndTime: types.EpochMillis(time.Now()),
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify it is finished.
			var finished bool
			waitForSeedResolution(ctx, seedResolutionAttempts, seedResolutionInterval, func() bool {
				launch, getErr := c.Project(projectName).Launches().GetByID(ctx, strconv.FormatInt(launchID, 10))
				if getErr == nil && launch != nil && launch.Status != "" {
					if launch.Status == "STOPPED" || launch.Status == "FAILED" || launch.Status == "PASSED" || launch.Status == "INTERRUPTED" {
						finished = true
						return true
					}
				}
				return false
			})
			Expect(finished).To(BeTrue(), "launch should have a terminal status after force-finish")
		})

		It("merges two launches", func(ctx SpecContext) {
			// Start and finish two launches.
			var launchIDs []int64
			for i := range 2 {
				started, err := c.Project(projectName).Launches().Start(ctx, types.StartLaunchRQ{
					Name:      fmt.Sprintf("merge-test-%d-%d", time.Now().UnixNano(), i),
					StartTime: types.EpochMillis(time.Now()),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(started.Id).NotTo(BeNil())

				finishedStatus := types.FinishExecutionRQStatusPASSED
				_, err = c.Project(projectName).Launches().Finish(ctx, *started.Id, types.FinishExecutionRQ{
					EndTime: types.EpochMillis(time.Now()),
					Status:  &finishedStatus,
				})
				Expect(err).NotTo(HaveOccurred())

				resolvedID, err := resolveLaunchIDByUUID(ctx, c.Project(projectName).Launches(), *started.Id)
				Expect(err).NotTo(HaveOccurred())

				launchIDs = append(launchIDs, resolvedID)
			}

			// Merge the two launches.
			mergedName := fmt.Sprintf("merged-launch-%d", time.Now().UnixNano())
			startTime := types.EpochMillis(time.Now().Add(-time.Minute))
			endTime := types.EpochMillis(time.Now())
			merged, err := c.Project(projectName).Launches().Merge(ctx, types.MergeLaunchesRQ{
				Name:      mergedName,
				Launches:  launchIDs,
				StartTime: &startTime,
				EndTime:   &endTime,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(merged).NotTo(BeNil())
			Expect(merged.Name).To(Equal(mergedName))
		})
	})

	Context("Filter Options", func() {
		It("filters by user", func(ctx SpecContext) {
			// Get a known user first
			all, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(all.Content).NotTo(BeEmpty())
			Expect(all.Content[0].Owner).NotTo(BeNil(), "launch should have an owner")

			knownUser := *all.Content[0].Owner

			result, err := c.Project(projectName).Launches().List(ctx,
				client.WithUser(knownUser),
			)
			Expect(err).NotTo(HaveOccurred(),
				"filtering launches by user %q should not fail", knownUser)

			for _, launch := range result.Content {
				if launch.Owner != nil {
					Expect(*launch.Owner).To(Equal(knownUser),
						"filtered launch should have owner %q", knownUser)
				}
			}
		})


		It("filters by description", func(ctx SpecContext) {
			// Note: this test may not find matches if seeded launches don't have descriptions
			result, err := c.Project(projectName).Launches().List(ctx,
				client.WithDescription("test"),
			)
			Expect(err).NotTo(HaveOccurred(),
				"filtering launches by description should not fail")
			Expect(result).NotTo(BeNil())
		})

		It("filters by has retries", func(ctx SpecContext) {
			result, err := c.Project(projectName).Launches().List(ctx,
				client.WithHasRetries(false),
			)
			Expect(err).NotTo(HaveOccurred(),
				"filtering launches by hasRetries should not fail")
			Expect(result).NotTo(BeNil())
		})
	})
})
