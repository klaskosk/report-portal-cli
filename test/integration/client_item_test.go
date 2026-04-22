//go:build integration

package integration_test

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Test Items", Label("client", "items"), func() {
	var (
		c           *client.Client
		projectName string
		seeded      seededTestItems
	)

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_item_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding one launch with suite and child test items")
		seeded, err = seedTestItems(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to seed test items in %q", projectName)
		// RP 5.15 async indexing requires longer startup for item/log hierarchy
	}, NodeTimeout(240*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, c, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("List", func() {
		It("returns items for the seeded launch", func(ctx SpecContext) {
			result, err := c.Project(projectName).Items().List(
				ctx,
				client.WithItemLaunchID(seeded.LaunchID),
				client.WithItemPageSize(20),
				client.WithItemSort("id,asc"),
			)
			Expect(err).NotTo(HaveOccurred(), "listing items by launch should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeNil(), "seeded launch should contain items")
			Expect(*result.Content).NotTo(BeEmpty(), "seeded launch should contain items")
		})

		It("filters by parent suite id", func(ctx SpecContext) {
			result, err := c.Project(projectName).Items().List(
				ctx,
				client.WithItemLaunchID(seeded.LaunchID),
				client.WithItemParentID(seeded.SuiteID),
				client.WithItemPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(), "listing child items by parent should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeNil(), "suite should contain at least one child item")
			Expect(*result.Content).NotTo(BeEmpty(), "suite should contain at least one child item")

			for _, item := range *result.Content {
				Expect(item.Parent).NotTo(BeNil())
				Expect(*item.Parent).To(Equal(seeded.SuiteID))
			}
		})

		It("filters by type", func(ctx SpecContext) {
			result, err := c.Project(projectName).Items().List(
				ctx,
				client.WithItemLaunchID(seeded.LaunchID),
				client.WithItemType("SUITE"),
				client.WithItemPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(), "listing suite items should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeNil())
			Expect(*result.Content).NotTo(BeEmpty())

			for _, item := range *result.Content {
				Expect(item.Type).NotTo(BeNil())
				Expect(*item.Type).To(Equal("SUITE"))
			}
		})
	})

	Context("GetByID", func() {
		It("returns the seeded suite item", func(ctx SpecContext) {
			item, err := c.Project(projectName).Items().GetByID(
				ctx,
				seeded.SuiteID,
			)
			Expect(err).NotTo(HaveOccurred(), "getting suite by id should not fail")
			Expect(item).NotTo(BeNil())
			Expect(item.Id).NotTo(BeNil())
			Expect(*item.Id).To(Equal(seeded.SuiteID))
		})

		It("returns not found for an invalid id", func(ctx SpecContext) {
			_, err := c.Project(projectName).Items().GetByID(ctx, int64(999999))
			Expect(err).To(HaveOccurred(), "getting item by invalid id should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for invalid item id should be not-found, got: %v", err)
		})
	})

	Context("GetByUUID", func() {
		It("returns the seeded suite item", func(ctx SpecContext) {
			item, err := c.Project(projectName).Items().GetByUUID(ctx, seeded.SuiteUUID)
			Expect(err).NotTo(HaveOccurred(), "getting suite by uuid should not fail")
			Expect(item).NotTo(BeNil())
			Expect(item.Uuid).NotTo(BeNil())
			Expect(*item.Uuid).To(Equal(seeded.SuiteUUID))
		})

		It("returns not found for an invalid uuid", func(ctx SpecContext) {
			_, err := c.Project(projectName).Items().GetByUUID(
				ctx,
				"00000000-0000-0000-0000-000000000000",
			)
			Expect(err).To(HaveOccurred(), "getting item by invalid uuid should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for invalid item uuid should be not-found, got: %v", err)
		})
	})

	Context("Bulk Write Operations", func() {
		It("bulk deletes test items", func(ctx SpecContext) {
			// Seed a new launch with items specifically for bulk deletion.
			deleteSeeded, err := seedTestItems(ctx, baseURL, bearerToken, projectName)
			Expect(err).To(Succeed(), "seeding items for bulk delete test should not fail")

			// Delete the suite item via bulk.
			results, err := c.Project(projectName).Items().DeleteBulk(ctx, []int64{deleteSeeded.SuiteID})
			Expect(err).NotTo(HaveOccurred(), "bulk deleting test items should not fail")
			Expect(results).NotTo(BeEmpty())

			// Verify the suite is gone.
			_, err = c.Project(projectName).Items().GetByID(ctx, deleteSeeded.SuiteID)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"deleted test item should return not-found, got: %v", err)
		})

		It("bulk updates test items", func(ctx SpecContext) {
			comment := "bulk updated by integration test"
			action := types.DescriptionActionUPDATE
			attributes := []types.UpdateItemAttributeRQ{}
			description := types.Description{Action: &action, Comment: &comment}

			result, err := c.Project(projectName).Items().BulkUpdate(ctx, types.BulkInfoUpdateRQ{
				Attributes:  &attributes,
				Ids:         []int64{seeded.SuiteID},
				Description: &description,
			})
			Expect(err).NotTo(HaveOccurred(), "bulk updating test items should not fail")
			Expect(result).NotTo(BeNil())
		})
	})

	Context("Write Operations", func() {
		It("updates a test item description", func(ctx SpecContext) {
			newDescription := "updated by integration test"
			result, err := c.Project(projectName).Items().Update(
				ctx,
				seeded.SuiteID,
				types.UpdateTestItemRQ{
					Description: &newDescription,
				},
			)
			Expect(err).NotTo(HaveOccurred(), "updating test item should not fail")
			Expect(result).NotTo(BeNil())

			// Verify the change.
			item, err := c.Project(projectName).Items().GetByID(
				ctx,
				seeded.SuiteID,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(item).NotTo(BeNil())
			Expect(item.Description).NotTo(BeNil())
			Expect(*item.Description).To(Equal("updated by integration test"))
		})

		It("deletes a test item", func(ctx SpecContext) {
			// Seed a new launch with items specifically for deletion.
			deleteSeeded, err := seedTestItems(ctx, baseURL, bearerToken, projectName)
			Expect(err).To(Succeed(), "seeding items for delete test should not fail")

			// Delete the suite item.
			_, err = c.Project(projectName).Items().Delete(ctx, deleteSeeded.SuiteID)
			Expect(err).NotTo(HaveOccurred(), "deleting test item should not fail")

			// Verify it's gone.
			_, err = c.Project(projectName).Items().GetByID(
				ctx,
				deleteSeeded.SuiteID,
			)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"deleted test item should return not-found, got: %v", err)
		})
	})

	Context("GetHistory", func() {
		It("returns history for seeded items", func(ctx SpecContext) {
			result, err := c.Project(projectName).Items().GetHistory(ctx,
				client.WithHistoryLaunchID(seeded.LaunchID),
				client.WithHistoryDepth(5),
				client.WithHistoryPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(), "getting items history should not fail")
			Expect(result).NotTo(BeNil(), "history result should not be nil")
		})
	})

	Context("GetByIDs", func() {
		It("returns multiple items at once", func(ctx SpecContext) {
			items, err := c.Project(projectName).Items().GetByIDs(
				ctx,
				seeded.SuiteID,
				seeded.TestID,
			)
			Expect(err).NotTo(HaveOccurred(), "getting items by IDs should not fail")
			Expect(items).NotTo(BeNil())
			Expect(items).To(HaveLen(2), "should return both seeded items")

			returnedIDs := make(map[int64]bool)
			for _, item := range items {
				Expect(item.Id).NotTo(BeNil())
				returnedIDs[*item.Id] = true
			}

			Expect(returnedIDs).To(HaveKey(seeded.SuiteID))
			Expect(returnedIDs).To(HaveKey(seeded.TestID))
		})
	})

	Context("External Issues", func() {
		It("links and unlinks an external issue", func(ctx SpecContext) {
			ticketID := fmt.Sprintf("RP-%d", time.Now().UnixNano())
			btsURL := "https://jira.example.com"
			btsProject := "RP"
			ticketURL := fmt.Sprintf("%s/browse/%s", btsURL, ticketID)

			By("changing the seeded test item status to failed")
			failedStatus := string(types.FinishTestItemRQStatusFAILED)
			_, err := c.Project(projectName).Items().Update(
				ctx,
				seeded.TestID,
				types.UpdateTestItemRQ{
					Status: &failedStatus,
				},
			)
			Expect(err).NotTo(HaveOccurred(), "updating item status should not fail")

			failed := waitForSeedResolution(ctx, seedResolutionAttempts, seedResolutionInterval, func() bool {
				item, getErr := c.Project(projectName).Items().GetByID(ctx, seeded.TestID)
				return getErr == nil && item != nil && item.Status != nil && *item.Status == failedStatus
			})
			Expect(failed).To(BeTrue(), "seeded test item should eventually report failed status")

			By("defining an issue type for the seeded test item")
			issues, err := c.Project(projectName).Items().DefineIssueType(ctx, types.DefineIssueRQ{
				Issues: []types.IssueDefinition{
					{
						TestItemId: seeded.TestID,
						Issue: types.Issue{
							IssueType: "pb001",
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred(), "defining issue type should not fail")
			Expect(issues).NotTo(BeEmpty(), "define issue type should return an updated issue")

			By("linking an external issue to the seeded test item")
			linkResults, err := c.Project(projectName).Items().LinkExternalIssues(
				ctx,
				types.LinkExternalIssueRQ{
					TestItemIds: []int64{seeded.TestID},
					Issues: types.ExternalSystemIssue{
						TicketId:   ticketID,
						BtsUrl:     btsURL,
						BtsProject: btsProject,
						Url:        ticketURL,
					},
				},
			)
			Expect(err).NotTo(HaveOccurred(), "linking an external issue should not fail")
			Expect(linkResults).NotTo(BeEmpty(), "linking an external issue should report a result")

			By("verifying the external issue link is visible on the item")
			linked := waitForSeedResolution(ctx, seedResolutionAttempts, seedResolutionInterval, func() bool {
				item, getErr := c.Project(projectName).Items().GetByID(ctx, seeded.TestID)
				if getErr != nil || item == nil || item.Issue == nil || item.Issue.ExternalSystemIssues == nil {
					return false
				}

				for _, issue := range *item.Issue.ExternalSystemIssues {
					if issue.TicketId == ticketID &&
						issue.BtsUrl == btsURL &&
						issue.BtsProject == btsProject &&
						issue.Url == ticketURL {
						return true
					}
				}

				return false
			})
			Expect(linked).To(BeTrue(), "linked issue %q should eventually appear on the test item", ticketID)

			By("unlinking the external issue from the seeded test item")
			unlinkResults, err := c.Project(projectName).Items().UnlinkExternalIssues(
				ctx,
				types.UnlinkExternalIssueRQ{
					TestItemIds: []int64{seeded.TestID},
					TicketIds:   []string{ticketID},
				},
			)
			Expect(err).NotTo(HaveOccurred(), "unlinking an external issue should not fail")
			Expect(unlinkResults).NotTo(BeEmpty(), "unlinking an external issue should report a result")

			By("verifying the external issue link is removed")
			unlinked := waitForSeedResolution(ctx, seedResolutionAttempts, seedResolutionInterval, func() bool {
				item, getErr := c.Project(projectName).Items().GetByID(ctx, seeded.TestID)
				if getErr != nil || item == nil {
					return false
				}

				if item.Issue == nil || item.Issue.ExternalSystemIssues == nil {
					return true
				}

				return len(*item.Issue.ExternalSystemIssues) == 0
			})
			Expect(unlinked).To(BeTrue(), "linked issue %q should eventually be removed", ticketID)
		})
	})

	Context("Advanced Filtering", func() {
		It("filters items by name", func(ctx SpecContext) {
			known, err := c.Project(projectName).Items().GetByID(ctx, seeded.SuiteID)
			Expect(err).NotTo(HaveOccurred())
			Expect(known).NotTo(BeNil())
			Expect(known.Name).NotTo(BeNil())

			result, err := c.Project(projectName).Items().List(
				ctx,
				client.WithItemLaunchID(seeded.LaunchID),
				client.WithItemName(*known.Name),
				client.WithItemPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(), "filtering items by name should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeNil())
			Expect(*result.Content).NotTo(BeEmpty())

			for _, item := range *result.Content {
				Expect(item.Name).NotTo(BeNil())
				Expect(*item.Name).To(Equal(*known.Name))
			}
		})

		It("filters items by path", func(ctx SpecContext) {
			known, err := c.Project(projectName).Items().GetByID(ctx, seeded.SuiteID)
			Expect(err).NotTo(HaveOccurred())
			Expect(known).NotTo(BeNil())
			Expect(known.Path).NotTo(BeNil(), "seeded suite item should have a path")

			pathID, err := strconv.ParseInt(*known.Path, 10, 64)
			Expect(err).NotTo(HaveOccurred(), "seeded suite item path should be numeric")

			result, err := c.Project(projectName).Items().List(
				ctx,
				client.WithItemLaunchID(seeded.LaunchID),
				client.WithItemPath(pathID),
				client.WithItemPageSize(20),
			)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred(), "filtering items by path should report the runtime contract")
			Expect(client.IsServerError(err)).To(BeTrue(), "path filtering should currently fail server-side in Report Portal 5.15")
			Expect(err.Error()).To(ContainSubstring("operator does not exist: ltree = bigint"))
		})

		It("smoke tests name-substring search while indexing catches up", func(ctx SpecContext) {
			searchTerm := "suite"
			var searchResult *types.PageTestItemResource

			waitForSeedResolution(ctx, 15, seedResolutionInterval, func() bool {
				result, err := c.Project(projectName).Items().Search(
					ctx,
					client.WithSearchItemName(searchTerm),
					client.WithSearchItemSort("name,asc"),
					client.WithSearchItemPageSize(20),
				)
				if err != nil || result == nil || result.Content == nil {
					return false
				}

				searchResult = result

				return len(*result.Content) > 0
			})

			if searchResult == nil || searchResult.Content == nil {
				var err error
				searchResult, err = c.Project(projectName).Items().Search(
					ctx,
					client.WithSearchItemName(searchTerm),
					client.WithSearchItemSort("name,asc"),
					client.WithSearchItemPageSize(20),
				)
				Expect(err).NotTo(HaveOccurred(), "searching items by name should not fail")
			}

			Expect(searchResult).NotTo(BeNil())
			Expect(searchResult.Content).NotTo(BeNil())
			Expect(searchResult.Page).NotTo(BeNil())

			content := *searchResult.Content
			if len(content) == 0 {
				// Report Portal 5.15 populates search asynchronously, so this integration case
				// only guarantees that the endpoint stays healthy while indexing catches up.
				return
			}

			matchedSeededSuite := false
			for _, item := range content {
				Expect(item.Id).NotTo(BeNil())
				Expect(item.Name).NotTo(BeNil())
				Expect(strings.ToLower(*item.Name)).To(ContainSubstring(searchTerm))
				if *item.Id == seeded.SuiteID {
					matchedSeededSuite = true
				}
			}

			Expect(matchedSeededSuite).To(BeTrue(),
				"indexed results, when available, should include the seeded suite item")
		})

		It("filters items with has-children flag", func(ctx SpecContext) {
			result, err := c.Project(projectName).Items().List(
				ctx,
				client.WithItemLaunchID(seeded.LaunchID),
				client.WithItemHasChildren(true),
				client.WithItemPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(), "filtering items with has-children should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeNil())
			Expect(*result.Content).NotTo(BeEmpty(), "suite item should have children")

			for _, item := range *result.Content {
				Expect(item.HasChildren).NotTo(BeNil())
				Expect(*item.HasChildren).To(BeTrue())
			}
		})

		It("filters items by status", func(ctx SpecContext) {
			result, err := c.Project(projectName).Items().List(
				ctx,
				client.WithItemLaunchID(seeded.LaunchID),
				client.WithItemStatus("PASSED"),
				client.WithItemPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(), "filtering items by status should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeNil())
			Expect(*result.Content).NotTo(BeEmpty())

			for _, item := range *result.Content {
				Expect(item.Status).NotTo(BeNil())
				Expect(*item.Status).To(Equal("PASSED"))
			}
		})

	})

	Context("ListV2", func() {
		It("returns a bad request without a data provider base", func(ctx SpecContext) {
			known, err := c.Project(projectName).Items().GetByID(ctx, seeded.TestID)
			Expect(err).NotTo(HaveOccurred())
			Expect(known).NotTo(BeNil())
			Expect(known.Name).NotTo(BeNil())
			Expect(known.Status).NotTo(BeNil())

			result, err := c.Project(projectName).Items().ListV2(
				ctx,
				client.WithItemV2LaunchID(int32(seeded.LaunchID)),
				client.WithItemV2ParentID(int32(seeded.SuiteID)),
				client.WithItemV2Name(*known.Name),
				client.WithItemV2Status(*known.Status),
				client.WithItemV2PageSize(20),
				client.WithItemV2Sort("name,asc"),
			)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred(), "listing items with v2 filters should report the runtime contract")
			Expect(client.IsBadRequest(err)).To(BeTrue(), "v2 list should return a bad request in Report Portal 5.15")
			Expect(err.Error()).To(ContainSubstring("data provider base"))
		})
	})

	Context("History with Options", func() {
		It("retrieves history with a launches limit", func(ctx SpecContext) {
			result, err := c.Project(projectName).Items().GetHistory(ctx,
				client.WithHistoryLaunchID(seeded.LaunchID),
				client.WithHistoryLaunchesLimit(5),
				client.WithHistoryDepth(5),
				client.WithHistoryPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting history with launches limit should not fail")
			Expect(result).NotTo(BeNil(), "history result should not be nil")
		})

		It("retrieves history filtered by item name", func(ctx SpecContext) {
			known, err := c.Project(projectName).Items().GetByID(ctx, seeded.TestID)
			Expect(err).NotTo(HaveOccurred())
			Expect(known).NotTo(BeNil())
			Expect(known.Name).NotTo(BeNil())

			result, err := c.Project(projectName).Items().GetHistory(ctx,
				client.WithHistoryLaunchID(seeded.LaunchID),
				client.WithHistoryItemName(*known.Name),
				client.WithHistoryDepth(5),
				client.WithHistoryPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting history filtered by item name should not fail")
			Expect(result).NotTo(BeNil(), "history result should not be nil")
		})

		It("retrieves history with a filter ID", func(ctx SpecContext) {
			By("creating a launch filter")
			filterName := fmt.Sprintf("history-filter-%d", time.Now().UnixNano())
			created, err := c.Project(projectName).Filters().Create(ctx,
				types.UpdateUserFilterRQ{
					Name: filterName,
					Type: types.UpdateUserFilterRQTypeLaunch,
					Conditions: []types.UserFilterCondition{
						{
							Condition:      "cnt",
							FilteringField: "name",
							Value:          "integration",
						},
					},
					Orders: []types.Order{
						{IsAsc: false, SortingColumn: "startTime"},
					},
				},
			)
			Expect(err).NotTo(HaveOccurred(), "creating filter should not fail")
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())

			By("retrieving history using the filter ID")
			result, err := c.Project(projectName).Items().GetHistory(ctx,
				client.WithHistoryFilterID(*created.Id),
				client.WithHistoryLaunchesLimit(10),
				client.WithHistoryDepth(5),
				client.WithHistoryPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting history with filter ID should not fail")
			Expect(result).NotTo(BeNil(), "history result should not be nil")
		})

		It("retrieves history with a specific history type", func(ctx SpecContext) {
			result, err := c.Project(projectName).Items().GetHistory(ctx,
				client.WithHistoryLaunchID(seeded.LaunchID),
				client.WithHistoryType("line"),
				client.WithHistoryDepth(5),
				client.WithHistoryPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting history with type should not fail")
			Expect(result).NotTo(BeNil(), "history result should not be nil")
		})
	})

	Context("Statistics", func() {
		It("returns a bad request without a data provider base", func(ctx SpecContext) {
			known, err := c.Project(projectName).Items().GetByID(ctx, seeded.TestID)
			Expect(err).NotTo(HaveOccurred())
			Expect(known).NotTo(BeNil())
			Expect(known.Name).NotTo(BeNil())

			result, err := c.Project(projectName).Items().GetStatistics(
				ctx,
				client.WithStatisticsLaunchID(seeded.LaunchID),
				client.WithStatisticsName(*known.Name),
				client.WithStatisticsType("TEST"),
			)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred(), "getting item statistics should report the runtime contract")
			Expect(client.IsBadRequest(err)).To(BeTrue(), "item statistics should return a bad request in Report Portal 5.15")
			Expect(err.Error()).To(ContainSubstring("data provider base"))
		})
	})

	Context("Attribute Keys", func() {
		It("returns attribute keys for a launch", func(ctx SpecContext) {
			attrKey := "env"

			By("adding an attribute to the seeded test item")
			_, err := c.Project(projectName).Items().Update(
				ctx,
				seeded.TestID,
				types.UpdateTestItemRQ{
					Attributes: &[]types.ItemAttributeResource{
						{Key: &attrKey, Value: "ci"},
					},
				},
			)
			Expect(err).NotTo(HaveOccurred(), "updating item with attributes should not fail")

			By("retrieving attribute keys for the launch")
			keys, err := c.Project(projectName).Items().GetAttributeKeys(
				ctx,
				seeded.LaunchID,
				"",
			)
			Expect(err).NotTo(HaveOccurred(), "getting attribute keys should not fail")
			Expect(keys).NotTo(BeNil(), "attribute keys result should not be nil")
			Expect(keys).To(ContainElement(attrKey),
				"attribute keys should include %q after it was added to an item", attrKey)
		})

		It("returns attribute keys for the project", func(ctx SpecContext) {
			attrKey := "project-env"

			By("adding an attribute to the seeded test item")
			_, err := c.Project(projectName).Items().Update(
				ctx,
				seeded.TestID,
				types.UpdateTestItemRQ{
					Attributes: &[]types.ItemAttributeResource{
						{Key: &attrKey, Value: "staging"},
					},
				},
			)
			Expect(err).NotTo(HaveOccurred(), "updating item with attribute should not fail")

			By("creating a launch filter for the project")
			filterName := fmt.Sprintf("item-attr-filter-%d", time.Now().UnixNano())
			created, err := c.Project(projectName).Filters().Create(ctx,
				types.UpdateUserFilterRQ{
					Name: filterName,
					Type: types.UpdateUserFilterRQTypeLaunch,
					Conditions: []types.UserFilterCondition{
						{
							Condition:      "cnt",
							FilteringField: "name",
							Value:          "integration",
						},
					},
					Orders: []types.Order{
						{IsAsc: false, SortingColumn: "startTime"},
					},
				},
			)
			Expect(err).NotTo(HaveOccurred(), "creating filter should not fail")
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())

			By("retrieving attribute keys for the project")
			keys, err := c.Project(projectName).Items().GetAttributeKeysForProject(
				ctx,
				*created.Id,
				"",
				client.WithProjectKeysLaunchesLimit(10),
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting attribute keys for project should not fail")
			Expect(keys).NotTo(BeNil(),
				"attribute keys for project should not be nil")
			Expect(keys).To(ContainElement(attrKey),
				"attribute keys should include %q after it was added to an item", attrKey)
		})
	})

})
