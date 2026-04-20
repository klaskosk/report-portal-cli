//go:build integration

package integration_test

import (
	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Instance Bug Tracking", Label("client", "instance-bts"), func() {
	var c *client.Client

	BeforeEach(func() {
		c = newClient()
	})

	Context("System Fields", func() {
		It("returns not found for a non-existent integration", func(ctx SpecContext) {
			_, err := c.InstanceBugTracking().GetSystemFields(ctx, 999999, "Bug")
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"non-existent integration should produce IsNotFound, got: %v", err)
		})
	})

	Context("Allowable Issue Types", func() {
		It("returns not found for a non-existent integration", func(ctx SpecContext) {
			_, err := c.InstanceBugTracking().GetAllowableIssueTypes(ctx, 999999)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"non-existent integration should produce IsNotFound, got: %v", err)
		})
	})
})
