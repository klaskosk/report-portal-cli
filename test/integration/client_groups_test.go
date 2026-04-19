//go:build integration

package integration_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

const rpErrorCodePaymentRequired = 5000

// Groups are an enterprise-only feature in Report Portal. The community
// edition used by CI returns 402 Payment Required for all group endpoints.
// These tests verify that the client surfaces the error correctly rather
// than panicking or returning misleading results.
var _ = Describe("Client: Groups", Label("client", "groups"), func() {
	var c *client.Client

	BeforeEach(func() {
		c = newClient()
	})

	expectPaymentRequired := func(err error) {
		GinkgoHelper()
		Expect(err).To(HaveOccurred(), "groups endpoint should fail on community edition")
		Expect(client.HasErrorCode(err, rpErrorCodePaymentRequired)).To(BeTrue(),
			"expected RP error code %d (payment required), got: %v", rpErrorCodePaymentRequired, err)
	}

	Context("CRUD Lifecycle", func() {
		It("returns payment required when creating a group", func(ctx SpecContext) {
			_, err := c.Groups().Create(ctx, types.CreateGroupRequest{
				Name: fmt.Sprintf("test_group_%d", time.Now().UnixNano()),
			})
			expectPaymentRequired(err)
		})

		It("returns payment required when getting a group", func(ctx SpecContext) {
			_, err := c.Groups().Get(ctx, 1)
			expectPaymentRequired(err)
		})

		It("returns payment required when updating a group", func(ctx SpecContext) {
			_, err := c.Groups().Update(ctx, 1, types.UpdateGroupRequest{Name: "updated"})
			expectPaymentRequired(err)
		})

		It("returns payment required when deleting a group", func(ctx SpecContext) {
			err := c.Groups().Delete(ctx, 1)
			expectPaymentRequired(err)
		})
	})

	Context("List", func() {
		It("returns payment required when listing groups", func(ctx SpecContext) {
			_, err := c.Groups().List(ctx)
			expectPaymentRequired(err)
		})

		It("returns payment required when listing with options", func(ctx SpecContext) {
			_, err := c.Groups().List(ctx,
				client.WithGroupPageSize(1),
				client.WithGroupOffset(0),
			)
			expectPaymentRequired(err)
		})
	})

	Context("User Management", func() {
		It("returns payment required when adding a user", func(ctx SpecContext) {
			_, err := c.Groups().AddUser(ctx, 1, 1)
			expectPaymentRequired(err)
		})

		It("returns payment required when getting a user", func(ctx SpecContext) {
			_, err := c.Groups().GetUser(ctx, 1, 1)
			expectPaymentRequired(err)
		})

		It("returns payment required when listing users", func(ctx SpecContext) {
			_, err := c.Groups().ListUsers(ctx, 1)
			expectPaymentRequired(err)
		})

		It("returns payment required when removing a user", func(ctx SpecContext) {
			err := c.Groups().RemoveUser(ctx, 1, 1)
			expectPaymentRequired(err)
		})
	})

	Context("Project Management", func() {
		It("returns payment required when adding a project", func(ctx SpecContext) {
			permission := types.AddProjectToGroupByIdRequestPermissionMEMBER
			_, err := c.Groups().AddProject(ctx, 1, 1, types.AddProjectToGroupByIdRequest{
				Permission: &permission,
			})
			expectPaymentRequired(err)
		})

		It("returns payment required when getting a project", func(ctx SpecContext) {
			_, err := c.Groups().GetProject(ctx, 1, 1)
			expectPaymentRequired(err)
		})

		It("returns payment required when listing projects", func(ctx SpecContext) {
			_, err := c.Groups().ListProjects(ctx, 1)
			expectPaymentRequired(err)
		})

		It("returns payment required when removing a project", func(ctx SpecContext) {
			err := c.Groups().RemoveProject(ctx, 1, 1)
			expectPaymentRequired(err)
		})
	})
})
