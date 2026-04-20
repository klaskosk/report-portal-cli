//go:build integration

package integration_test

import (
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Bug Tracking", Label("client", "bts"), func() {
	var (
		c           *client.Client
		projectName string
	)

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_bts_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)
	}, NodeTimeout(30*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, c, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("Ticket IDs", func() {
		It("returns ticket IDs for a project without error", func(ctx SpecContext) {
			By("searching for ticket IDs in the project")
			ids, err := c.Project(projectName).BugTracking().GetTicketIdsForProject(ctx, "test")
			Expect(err).NotTo(HaveOccurred())
			Expect(ids).NotTo(BeNil())
		})

		It("returns ticket IDs for a launch", func(ctx SpecContext) {
			By("seeding a launch into " + projectName)
			err := seedTestLaunches(ctx, baseURL, bearerToken, projectName, 1)
			Expect(err).To(Succeed(), "failed to seed launch")

			By("listing launches to get the launch ID")
			list, err := c.Project(projectName).Launches().List(ctx)
			Expect(err).NotTo(HaveOccurred())
			launches := expectContent(list.Content, "launch list")

			By("searching for ticket IDs for the launch")
			ids, err := c.Project(projectName).BugTracking().GetTicketIds(ctx, launches[0].Id, "test")
			Expect(err).NotTo(HaveOccurred())
			Expect(ids).NotTo(BeNil())
		}, NodeTimeout(60*time.Second))
	})

	Context("System Fields", func() {
		It("returns not found for a non-existent integration", func(ctx SpecContext) {
			_, err := c.Project(projectName).BugTracking().GetSystemFields(ctx, 999999)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"non-existent integration should produce IsNotFound, got: %v", err)
		})
	})

	Context("Allowable Issue Types", func() {
		It("returns not found for a non-existent integration", func(ctx SpecContext) {
			_, err := c.Project(projectName).BugTracking().GetAllowableIssueTypes(ctx, 999999)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"non-existent integration should produce IsNotFound, got: %v", err)
		})
	})

	Context("Tickets", func() {
		It("returns not found for a non-existent ticket", func(ctx SpecContext) {
			_, err := c.Project(projectName).BugTracking().GetTicket(
				ctx,
				"NONEXISTENT-99999",
				"https://nonexistent.example.com",
				"NONEXISTENT",
			)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"non-existent ticket should produce IsNotFound, got: %v", err)
		})
	})
})
