//go:build integration

package integration_test

import (
	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: ProjectSettings", Label("client", "project-settings"), func() {
	var c *client.Client

	BeforeEach(func() {
		c = newClient()
	})

	Context("Get", func() {
		It("returns settings with standard issue sub-types", func(ctx SpecContext) {
			settings, err := c.Project("superadmin_personal").Settings().Get(ctx)
			Expect(err).NotTo(HaveOccurred(), "Get should not fail")
			Expect(settings).NotTo(BeNil())
			Expect(settings.SubTypes).NotTo(BeEmpty(), "should have issue sub-types")

			expectedGroups := []string{
				"TO_INVESTIGATE",
				"PRODUCT_BUG",
				"AUTOMATION_BUG",
				"SYSTEM_ISSUE",
				"NO_DEFECT",
			}
			for _, group := range expectedGroups {
				Expect(settings.SubTypes).To(HaveKey(group),
					"should have %q issue sub-type group", group)
				Expect(settings.SubTypes[group]).NotTo(BeEmpty(),
					"%q group should have at least one sub-type", group)
			}
		})

		It("returns not found for a non-existent project", func(ctx SpecContext) {
			_, err := c.Project("nonexistent_project_12345").Settings().Get(ctx)
			Expect(err).To(HaveOccurred(), "Get should fail for unknown project")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error should be not-found, got: %v", err)
		})
	})

	Context("GetNotifications", func() {
		It("returns without error", func(ctx SpecContext) {
			notifications, err := c.Project("superadmin_personal").Settings().GetNotifications(ctx)
			Expect(err).NotTo(HaveOccurred(), "GetNotifications should not fail")
			Expect(notifications).NotTo(BeNil(), "notifications should not be nil")
		})
	})
})
