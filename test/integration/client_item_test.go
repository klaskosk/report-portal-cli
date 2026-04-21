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

		It("searches items by name substring", func(ctx SpecContext) {
			searchTerm := "integration-suite"

			result, err := c.Project(projectName).Items().Search(
				ctx,
				client.WithSearchItemName(searchTerm),
				client.WithSearchItemSort("name,asc"),
				client.WithSearchItemPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(), "searching items by name should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeNil())

			// Search depends on async OpenSearch indexing which may not complete
			// within the test window. When results are present, validate every
			// returned item matches the search filter.
			for _, item := range *result.Content {
				Expect(item.Name).NotTo(BeNil())
				Expect(*item.Name).To(ContainSubstring(searchTerm))
			}
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
