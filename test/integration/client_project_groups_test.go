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

// Project-level group associations are an enterprise-only feature. The
// community edition returns 402 Payment Required for all project-groups
// endpoints. These tests verify the client surfaces the error correctly.
var _ = Describe("Client: Project Groups", Label("client", "project-groups"), func() {
	var c *client.Client
	var projectName string

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("proj_groups_test_%d", time.Now().UnixNano())

		By("creating a test project: " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, c, projectName)
		}
	}, NodeTimeout(15*time.Second))

	expectPaymentRequired := func(err error) {
		GinkgoHelper()
		Expect(err).To(HaveOccurred(), "project-groups endpoint should fail on community edition")
		Expect(client.HasErrorCode(err, rpErrorCodePaymentRequired)).To(BeTrue(),
			"expected RP error code %d (payment required), got: %v", rpErrorCodePaymentRequired, err)
	}

	Context("CRUD Lifecycle", func() {
		It("returns payment required when adding a group to a project", func(ctx SpecContext) {
			permission := types.AddProjectToGroupByIdRequestPermissionMEMBER
			_, err := c.Project(projectName).Groups().AddOrUpdate(ctx, 1, types.AddProjectToGroupByIdRequest{
				Permission: &permission,
			})
			expectPaymentRequired(err)
		})

		It("returns payment required when getting a group from a project", func(ctx SpecContext) {
			_, err := c.Project(projectName).Groups().Get(ctx, 1)
			expectPaymentRequired(err)
		})

		It("returns payment required when deleting a group from a project", func(ctx SpecContext) {
			err := c.Project(projectName).Groups().Delete(ctx, 1)
			expectPaymentRequired(err)
		})
	})

	Context("List", func() {
		It("returns payment required when listing project groups", func(ctx SpecContext) {
			_, err := c.Project(projectName).Groups().List(ctx)
			expectPaymentRequired(err)
		})
	})
})
