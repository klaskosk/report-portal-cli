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
		c          *client.Client
		projectName string
		seeded     seededTestItems
	)

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_item_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding one launch with suite and child test items")
		seeded, err = seedTestItems(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to seed test items in %q", projectName)
	}, NodeTimeout(90*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, baseURL, bearerToken, projectName)
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
			Expect(result.Content).NotTo(BeEmpty(), "seeded launch should contain items")
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
			Expect(result.Content).NotTo(BeEmpty(), "suite should contain at least one child item")

			for _, item := range result.Content {
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
			Expect(result.Content).NotTo(BeEmpty())

			for _, item := range result.Content {
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
				Ids:         &[]int64{seeded.SuiteID},
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
})
