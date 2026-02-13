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
				seededLogs.LogIDs[0],
			)
			Expect(err).NotTo(HaveOccurred(), "getting log by id should not fail")
			Expect(logEntry).NotTo(BeNil())
			Expect(logEntry.Id).To(Equal(seededLogs.LogIDs[0]))
		})

		It("returns not found for an invalid id", func(ctx SpecContext) {
			_, err := c.Project(projectName).Logs().GetByID(ctx, int64(999999))
			Expect(err).To(HaveOccurred(), "getting log by invalid id should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for invalid log id should be not-found, got: %v", err)
		})
	})

	Context("Batch Write Operations", func() {
		It("creates batch log entries without attachment", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().CreateBatch(ctx,
				client.BatchLogEntry{
					LaunchUUID: seeded.LaunchUUID,
					ItemUUID:   seeded.SuiteUUID,
					Level:      "INFO",
					Message:    "batch-log-entry-1",
					Time:       types.EpochMillis(time.Now()),
				},
				client.BatchLogEntry{
					LaunchUUID: seeded.LaunchUUID,
					ItemUUID:   seeded.SuiteUUID,
					Level:      "ERROR",
					Message:    "batch-log-entry-2",
					Time:       types.EpochMillis(time.Now()),
				},
			)
			Expect(err).NotTo(HaveOccurred(), "batch creating log entries should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Responses).NotTo(BeNil())
			Expect(*result.Responses).NotTo(BeEmpty(), "batch response should contain entries")
		})

		It("creates batch log entry with attachment", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().CreateBatch(ctx,
				client.BatchLogEntry{
					LaunchUUID:     seeded.LaunchUUID,
					ItemUUID:       seeded.SuiteUUID,
					Level:          "INFO",
					Message:        "log with attachment",
					Time:           types.EpochMillis(time.Now()),
					AttachmentName: "test.txt",
					AttachmentData: []byte("integration test attachment content"),
				},
			)
			Expect(err).NotTo(HaveOccurred(), "batch creating log with attachment should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Responses).NotTo(BeNil())
			Expect(*result.Responses).NotTo(BeEmpty())
		})
	})

	Context("Write Operations", func() {
		It("creates a log entry via client", func(ctx SpecContext) {
			message := "client-created log entry"
			level := types.SaveLogRQLevel("INFO")

			result, err := c.Project(projectName).Logs().Create(ctx, types.SaveLogRQ{
				LaunchUuid: seeded.LaunchUUID,
				ItemUuid:   &seeded.SuiteUUID,
				Message:    &message,
				Level:      &level,
				Time:       types.EpochMillis(time.Now()),
			})
			Expect(err).NotTo(HaveOccurred(), "creating log entry should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Id).NotTo(BeNil())

			// Wait for the log to appear in listing.
			var found bool
			waitForSeedResolution(ctx, seedResolutionAttempts, seedResolutionInterval, func() bool {
				logs, listErr := c.Project(projectName).Logs().List(ctx,
					client.WithLogItemID(seeded.SuiteID),
					client.WithLogPageSize(50),
				)
				if listErr != nil || logs == nil {
					return false
				}

				for _, entry := range logs.Content {
					if entry.Message != nil && *entry.Message == "client-created log entry" {
						found = true
						return true
					}
				}

				return false
			})
			Expect(found).To(BeTrue(), "created log entry should appear in listing")
		})

		It("deletes a log entry", func(ctx SpecContext) {
			Expect(seededLogs.LogIDs).NotTo(BeEmpty(), "expected seeded log IDs for deletion")

			logID := seededLogs.LogIDs[len(seededLogs.LogIDs)-1]
			_, err := c.Project(projectName).Logs().Delete(ctx, logID)
			Expect(err).NotTo(HaveOccurred(), "deleting log entry should not fail")

			// Verify it's gone.
			_, err = c.Project(projectName).Logs().GetByID(ctx, logID)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"deleted log should return not-found, got: %v", err)
		})
	})
})
