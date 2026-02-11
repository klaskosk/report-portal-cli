//go:build integration

package integration_test

import (
	"fmt"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
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
				strconv.FormatInt(seeded.SuiteID, 10),
			)
			Expect(err).NotTo(HaveOccurred(), "getting suite by id should not fail")
			Expect(item).NotTo(BeNil())
			Expect(item.Id).NotTo(BeNil())
			Expect(*item.Id).To(Equal(seeded.SuiteID))
		})

		It("returns not found for an invalid id", func(ctx SpecContext) {
			_, err := c.Project(projectName).Items().GetByID(ctx, "999999")
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
})
