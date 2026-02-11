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

var _ = Describe("Client: Logs", Label("client", "logs"), func() {
	var (
		c           *client.Client
		projectName string
		seeded      seededTestItems
		seededLogs  seededLogEntries
	)

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_log_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding one launch with suite and child test items")
		seeded, err = seedTestItems(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to seed test items in %q", projectName)

		By("seeding log entries for the suite item")
		seededLogs, err = seedTestLogEntries(ctx, baseURL, bearerToken, projectName, seeded, 3)
		Expect(err).To(Succeed(), "failed to seed log entries in %q", projectName)
	}, NodeTimeout(120*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, baseURL, bearerToken, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("List", func() {
		It("returns logs filtered by item ID", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().List(
				ctx,
				client.WithLogItemID(seeded.SuiteID),
				client.WithLogPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(), "listing logs by item should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeEmpty(), "seeded item should have log entries")
		})

		It("returns empty results for an item with no logs", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().List(
				ctx,
				client.WithLogItemID(999999),
				client.WithLogPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(), "listing logs for missing item should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).To(BeEmpty())
		})
	})

	Context("GetByID", func() {
		It("returns a known log entry", func(ctx SpecContext) {
			Expect(seededLogs.LogIDs).NotTo(BeEmpty(), "expected seeded log IDs")

			logEntry, err := c.Project(projectName).Logs().GetByID(
				ctx,
				strconv.FormatInt(seededLogs.LogIDs[0], 10),
			)
			Expect(err).NotTo(HaveOccurred(), "getting log by id should not fail")
			Expect(logEntry).NotTo(BeNil())
			Expect(logEntry.Id).To(Equal(seededLogs.LogIDs[0]))
		})

		It("returns not found for an invalid id", func(ctx SpecContext) {
			_, err := c.Project(projectName).Logs().GetByID(ctx, "999999")
			Expect(err).To(HaveOccurred(), "getting log by invalid id should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for invalid log id should be not-found, got: %v", err)
		})
	})
})
