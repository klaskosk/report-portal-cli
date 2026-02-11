//go:build integration

package integration_test

import (
	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Users", Label("client", "users"), func() {
	var c *client.Client

	BeforeEach(func() {
		c = newClient()
	})

	Context("GetMyself", func() {
		It("returns the authenticated user", func(ctx SpecContext) {
			user, err := c.Users().GetMyself(ctx)
			Expect(err).NotTo(HaveOccurred(), "GetMyself should not fail")
			Expect(user).NotTo(BeNil())
			Expect(user.UserId).To(Equal(testAdminUser))
			Expect(user.Email).NotTo(BeEmpty())
		})
	})

	Context("GetByLogin", func() {
		It("returns the admin user", func(ctx SpecContext) {
			user, err := c.Users().GetByLogin(ctx, testAdminUser)
			Expect(err).NotTo(HaveOccurred(), "GetByLogin should not fail")
			Expect(user).NotTo(BeNil())
			Expect(user.UserId).To(Equal(testAdminUser))
		})

		It("returns not found for an unknown login", func(ctx SpecContext) {
			_, err := c.Users().GetByLogin(ctx, "nonexistent_user_12345")
			Expect(err).To(HaveOccurred(), "getting unknown user should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for unknown user should be not-found, got: %v", err)
		})
	})

	Context("ListProjects", func() {
		It("returns at least one project for the admin user", func(ctx SpecContext) {
			projects, err := c.Users().ListProjects(ctx, testAdminUser)
			Expect(err).NotTo(HaveOccurred(), "ListProjects should not fail")
			Expect(projects).NotTo(BeEmpty(),
				"admin user should have at least one project")
		})
	})
})
