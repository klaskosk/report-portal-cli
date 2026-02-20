//go:build integration

package integration_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Bug Tracking", Label("client", "bts"), func() {
	Context("Ticket IDs", func() {
		It("returns ticket IDs for a project without error", func(ctx SpecContext) {
			c := newClient()
			projectName := fmt.Sprintf("client_bts_test_%d", time.Now().UnixNano())

			By("creating isolated project " + projectName)
			err := createProject(ctx, baseURL, bearerToken, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)

			defer deleteProject(ctx, baseURL, bearerToken, projectName)

			By("searching for ticket IDs in the project")
			ids, err := c.Project(projectName).BugTracking().GetTicketIdsForProject(ctx, "test")
			Expect(err).NotTo(HaveOccurred())
			// A fresh project should have no tickets, but the endpoint should succeed
			Expect(ids).NotTo(BeNil())
		}, NodeTimeout(60*time.Second))
	})
})
