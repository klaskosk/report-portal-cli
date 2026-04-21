//go:build integration

package integration_test

import (
	"fmt"
	"strings"
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
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding one launch with suite and child test items")
		seeded, err = seedTestItems(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to seed test items in %q", projectName)

		By("seeding log entries for the suite item")
		seededLogs, err = seedTestLogEntries(ctx, c, projectName, seeded, 3)
		Expect(err).To(Succeed(), "failed to seed log entries in %q", projectName)
	}, NodeTimeout(240*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, c, projectName)
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
			Expect(result.Content).NotTo(BeNil(), "seeded item should have log entries")
			Expect(*result.Content).NotTo(BeEmpty(), "seeded item should have log entries")
		})

		It("returns empty results for an item with no logs", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().List(
				ctx,
				client.WithLogItemID(999999),
				client.WithLogPageSize(20),
			)
			Expect(err).NotTo(HaveOccurred(), "listing logs for missing item should not fail")
			Expect(result).NotTo(BeNil())
			if result.Content != nil {
				Expect(*result.Content).To(BeEmpty())
			}
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
					Time:       time.Now(),
				},
				client.BatchLogEntry{
					LaunchUUID: seeded.LaunchUUID,
					ItemUUID:   seeded.SuiteUUID,
					Level:      "ERROR",
					Message:    "batch-log-entry-2",
					Time:       time.Now(),
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
					Time:           time.Now(),
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
			level := "INFO"

			result, err := c.Project(projectName).Logs().Create(ctx, types.SaveLogRQ{
				LaunchUuid: seeded.LaunchUUID,
				ItemUuid:   &seeded.SuiteUUID,
				Message:    &message,
				Level:      &level,
				Time:       time.Now(),
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
				if listErr != nil || logs == nil || logs.Content == nil {
					return false
				}

				for _, entry := range *logs.Content {
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

	Context("Search", func() {
		It("searches logs for a seeded item without error", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().Search(ctx,
				seeded.SuiteID,
				types.SearchLogRq{},
			)
			if err != nil {
				Expect(client.IsConflict(err)).To(BeTrue(),
					"searching logs should only fail when analyzer integration is unavailable, got: %v", err)
				return
			}
			Expect(result).NotTo(BeNil(), "search result should not be nil")
		})
	})

	Context("GetLogsUnder", func() {
		It("retrieves logs grouped by item ID", func(ctx SpecContext) {
			itemIDs := []int64{seeded.SuiteID}
			result, err := c.Project(projectName).Logs().GetLogsUnder(ctx,
				types.GetLogsUnderRq{ItemIds: itemIDs},
			)
			Expect(err).NotTo(HaveOccurred(), "getting logs under items should not fail")
			Expect(result).NotTo(BeNil(), "logs under result should not be nil")
		})
	})

	Context("Advanced Search", func() {
		It("searches logs with current-launch mode", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().Search(ctx,
				seeded.SuiteID,
				types.SearchLogRq{SearchMode: types.CurrentLaunch},
			)
			if err != nil {
				Expect(client.IsConflict(err)).To(BeTrue(),
					"searching logs should only fail when analyzer is unavailable, got: %v", err)
				return
			}
			Expect(result).NotTo(BeNil(), "search result should not be nil")
		})

		It("searches logs with launch-name mode", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().Search(ctx,
				seeded.SuiteID,
				types.SearchLogRq{SearchMode: types.LaunchName},
			)
			if err != nil {
				Expect(client.IsConflict(err)).To(BeTrue(),
					"searching logs should only fail when analyzer is unavailable, got: %v", err)
				return
			}
			Expect(result).NotTo(BeNil(), "search result should not be nil")
		})

		It("returns empty results for an item with no matching logs", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().Search(ctx,
				seeded.TestID,
				types.SearchLogRq{SearchMode: types.CurrentLaunch},
			)
			if err != nil {
				Expect(client.IsConflict(err)).To(BeTrue(),
					"searching logs should only fail when analyzer is unavailable, got: %v", err)
				return
			}
			Expect(result).NotTo(BeNil())
		})
	})

	Context("Nested Logs", func() {
		It("retrieves logs under a parent item with explicit log level", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().GetLogsUnder(ctx,
				types.GetLogsUnderRq{
					ItemIds:  []int64{seeded.SuiteID},
					LogLevel: "ERROR",
				},
			)
			Expect(err).NotTo(HaveOccurred(), "getting logs under items with level filter should not fail")
			Expect(result).NotTo(BeNil())

			key := fmt.Sprintf("%d", seeded.SuiteID)
			if logs, ok := result[key]; ok {
				for _, entry := range logs {
					if entry.Level != nil {
						upper := strings.ToUpper(*entry.Level)
						Expect(upper).To(
							BeElementOf("ERROR", "FATAL"),
							"only ERROR-level (or higher) logs should be returned, got: %s", *entry.Level,
						)
					}
				}
			}
		})

		It("returns empty map for an item with no logs", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().GetLogsUnder(ctx,
				types.GetLogsUnderRq{ItemIds: []int64{999999}},
			)
			Expect(err).NotTo(HaveOccurred(), "getting logs under unknown item should not fail")
			Expect(result).NotTo(BeNil())

			key := fmt.Sprintf("%d", 999999)
			if logs, ok := result[key]; ok {
				Expect(logs).To(BeEmpty(), "unknown item should have no logs")
			}
		})

		It("retrieves logs for multiple items at once", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().GetLogsUnder(ctx,
				types.GetLogsUnderRq{
					ItemIds: []int64{seeded.SuiteID, seeded.TestID},
				},
			)
			Expect(err).NotTo(HaveOccurred(), "getting logs under multiple items should not fail")
			Expect(result).NotTo(BeNil())
		})
	})

	Context("Batch Creation with Levels", func() {
		It("creates logs with different log levels and verifies them", func(ctx SpecContext) {
			levels := []string{"ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
			entries := make([]client.BatchLogEntry, len(levels))
			for i, level := range levels {
				entries[i] = client.BatchLogEntry{
					LaunchUUID: seeded.LaunchUUID,
					ItemUUID:   seeded.SuiteUUID,
					Level:      level,
					Message:    fmt.Sprintf("batch-level-test-%s", level),
					Time:       time.Now(),
				}
			}

			By("creating log entries with ERROR, WARN, INFO, DEBUG, TRACE levels")
			result, err := c.Project(projectName).Logs().CreateBatch(ctx, entries...)
			Expect(err).NotTo(HaveOccurred(), "batch creating logs with different levels should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Responses).NotTo(BeNil())
			Expect(*result.Responses).To(HaveLen(len(levels)),
				"batch response should contain one entry per level")

			By("listing logs and verifying levels appear")
			resolved := waitForSeedResolution(ctx, seedResolutionAttempts, seedResolutionInterval, func() bool {
				logs, listErr := c.Project(projectName).Logs().List(ctx,
					client.WithLogItemID(seeded.SuiteID),
					client.WithLogPageSize(50),
				)
				if listErr != nil || logs == nil || logs.Content == nil {
					return false
				}

				foundLevels := map[string]bool{}
				for _, entry := range *logs.Content {
					if entry.Level != nil && entry.Message != nil {
						for _, level := range levels {
							if *entry.Message == fmt.Sprintf("batch-level-test-%s", level) {
								foundLevels[level] = true
							}
						}
					}
				}

				return len(foundLevels) == len(levels)
			})
			Expect(resolved).To(BeTrue(), "all batch-created messages should appear in listing")

			logs, err := c.Project(projectName).Logs().List(ctx,
				client.WithLogItemID(seeded.SuiteID),
				client.WithLogPageSize(100),
			)
			Expect(err).NotTo(HaveOccurred(), "listing logs after batch create should not fail")
			Expect(logs).NotTo(BeNil())
			Expect(logs.Content).NotTo(BeNil())

			for _, level := range levels {
				wantMsg := fmt.Sprintf("batch-level-test-%s", level)
				var matched bool
				for _, entry := range *logs.Content {
					if entry.Message == nil || *entry.Message != wantMsg {
						continue
					}
					matched = true
					Expect(entry.Level).NotTo(BeNil(), "log level should be present for message %q", wantMsg)
					Expect(strings.EqualFold(*entry.Level, level)).To(BeTrue(),
						"message %q should have level %s, got %s", wantMsg, level, *entry.Level)
				}
				Expect(matched).To(BeTrue(), "expected message %q in listing", wantMsg)
			}
		})

		It("creates a large batch of log entries", func(ctx SpecContext) {
			const batchSize = 20
			entries := make([]client.BatchLogEntry, batchSize)
			for i := range batchSize {
				entries[i] = client.BatchLogEntry{
					LaunchUUID: seeded.LaunchUUID,
					ItemUUID:   seeded.SuiteUUID,
					Level:      "INFO",
					Message:    fmt.Sprintf("large-batch-entry-%d", i+1),
					Time:       time.Now(),
				}
			}

			result, err := c.Project(projectName).Logs().CreateBatch(ctx, entries...)
			Expect(err).NotTo(HaveOccurred(), "large batch creation should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Responses).NotTo(BeNil())
			Expect(*result.Responses).To(HaveLen(batchSize))
		})

		It("creates batch with multiple attachments", func(ctx SpecContext) {
			entries := []client.BatchLogEntry{
				{
					LaunchUUID:     seeded.LaunchUUID,
					ItemUUID:       seeded.SuiteUUID,
					Level:          "INFO",
					Message:        "multi-attach-1",
					Time:           time.Now(),
					AttachmentName: "file1.txt",
					AttachmentData: []byte("content of file 1"),
				},
				{
					LaunchUUID:     seeded.LaunchUUID,
					ItemUUID:       seeded.SuiteUUID,
					Level:          "ERROR",
					Message:        "multi-attach-2",
					Time:           time.Now(),
					AttachmentName: "file2.log",
					AttachmentData: []byte("content of file 2"),
				},
			}

			result, err := c.Project(projectName).Logs().CreateBatch(ctx, entries...)
			Expect(err).NotTo(HaveOccurred(), "batch with multiple attachments should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Responses).NotTo(BeNil())
			Expect(*result.Responses).To(HaveLen(2))
		})
	})

	Context("Page Number", func() {
		It("returns a bad-request error because no sort parameter is defined", func(ctx SpecContext) {
			Expect(seededLogs.LogIDs).NotTo(BeEmpty(), "expected seeded log IDs")

			_, err := c.Project(projectName).Logs().GetPageNumber(ctx,
				seededLogs.LogIDs[0],
				client.WithLogPageItemID(seeded.SuiteID),
				client.WithLogPageNumberPageSize(10),
			)
			Expect(err).To(HaveOccurred(), "GetPageNumber should fail without a sort parameter")
			Expect(client.IsBadRequest(err)).To(BeTrue(),
				"error should be bad-request, got: %v", err)
		})

		It("returns the page index for a seeded log when sort and page size match listing", func(ctx SpecContext) {
			Expect(seededLogs.LogIDs).NotTo(BeEmpty(), "expected seeded log IDs")

			const sort = "logTime,desc"
			pageSize := int32(2)
			logID := seededLogs.LogIDs[0]

			listed, err := c.Project(projectName).Logs().List(ctx,
				client.WithLogItemID(seeded.SuiteID),
				client.WithLogSort(sort),
				client.WithLogPageSize(100),
			)
			Expect(err).NotTo(HaveOccurred(), "listing logs for page calculation should not fail")
			Expect(listed).NotTo(BeNil())
			Expect(listed.Content).NotTo(BeNil())

			index := -1
			for i, entry := range *listed.Content {
				if entry.Id == logID {
					index = i
					break
				}
			}
			Expect(index).To(BeNumerically(">=", 0), "seeded log should appear in sorted listing")

			// Report Portal returns a 1-based page index for this endpoint.
			expectedNumber := index/int(pageSize) + 1

			payload, err := c.Project(projectName).Logs().GetPageNumber(ctx, logID,
				client.WithLogPageItemID(seeded.SuiteID),
				client.WithLogPageNumberPageSize(pageSize),
				client.WithLogPageNumberSort(sort),
			)
			Expect(err).NotTo(HaveOccurred(),
				"GetPageNumber should succeed with sort aligned to listing, got: %v", err)
			Expect(payload).NotTo(BeNil())
			Expect(payload).To(HaveKey("number"))
			Expect(payload["number"]).To(Equal(float64(expectedNumber)),
				"page number should match 1-based page derived from sorted list index and page size")
		})
	})

	Context("Log Locations", func() {
		It("returns locations for a known parent item", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().GetLogLocations(ctx,
				seeded.SuiteID,
			)
			Expect(err).NotTo(HaveOccurred(), "getting log locations should not fail")
			Expect(result).NotTo(BeNil(), "log locations result should not be nil")
		})

		It("returns empty locations for an item with no error logs", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().GetLogLocations(ctx,
				seeded.TestID,
			)
			Expect(err).NotTo(HaveOccurred(), "getting log locations for item without errors should not fail")
			Expect(result).NotTo(BeNil())
		})

		It("respects pagination options", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().GetLogLocations(ctx,
				seeded.SuiteID,
				client.WithLogLocationsPageSize(1),
				client.WithLogLocationsPageNumber(0),
			)
			Expect(err).NotTo(HaveOccurred(), "getting log locations with pagination should not fail")
			Expect(result).NotTo(BeNil())
		})
	})

	Context("Advanced Filtering", func() {
		It("filters logs by level", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().List(ctx,
				client.WithLogItemID(seeded.SuiteID),
				client.WithLogLevel("ERROR"),
				client.WithLogPageSize(50),
			)
			Expect(err).NotTo(HaveOccurred(), "filtering logs by level should not fail")
			Expect(result).NotTo(BeNil())

			if result.Content != nil {
				for _, entry := range *result.Content {
					if entry.Level != nil {
						Expect(strings.EqualFold(*entry.Level, "ERROR")).To(BeTrue(),
							"all returned logs should have ERROR level, got: %s", *entry.Level)
					}
				}
			}
		})

		It("filters logs by launch ID", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().List(ctx,
				client.WithLogLaunchID(seeded.LaunchID),
				client.WithLogPageSize(50),
			)
			Expect(err).NotTo(HaveOccurred(), "filtering logs by launch ID should not fail")
			Expect(result).NotTo(BeNil())
			if result.Content != nil {
				Expect(*result.Content).NotTo(BeEmpty(),
					"seeded launch should have log entries")
			}
		})

		It("paginates through logs", func(ctx SpecContext) {
			By("fetching all logs to verify total count")
			allLogs, err := c.Project(projectName).Logs().List(ctx,
				client.WithLogItemID(seeded.SuiteID),
				client.WithLogPageSize(50),
			)
			Expect(err).NotTo(HaveOccurred(), "fetching all logs should not fail")
			Expect(allLogs).NotTo(BeNil())
			Expect(allLogs.Content).NotTo(BeNil())
			totalCount := len(*allLogs.Content)
			Expect(totalCount).To(BeNumerically(">=", 3),
				"seeded logs should have at least 3 entries")

			By("fetching with small page size to verify pagination metadata")
			smallPage, err := c.Project(projectName).Logs().List(ctx,
				client.WithLogItemID(seeded.SuiteID),
				client.WithLogPageSize(2),
				client.WithLogPageNumber(0),
			)
			Expect(err).NotTo(HaveOccurred(), "fetching small page should not fail")
			Expect(smallPage).NotTo(BeNil())
			Expect(smallPage.Content).NotTo(BeNil())
			Expect(*smallPage.Content).To(HaveLen(2),
				"page size 2 should return exactly 2 entries when total >= 3")
			Expect(smallPage.Page).NotTo(BeNil(), "page metadata should be present")
			Expect(smallPage.Page.TotalElements).NotTo(BeNil())
			Expect(*smallPage.Page.TotalElements).To(BeNumerically(">=", 3))
		})

		It("sorts logs by time descending", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().List(ctx,
				client.WithLogItemID(seeded.SuiteID),
				client.WithLogSort("logTime,desc"),
				client.WithLogPageSize(50),
			)
			Expect(err).NotTo(HaveOccurred(), "sorting logs should not fail")
			Expect(result).NotTo(BeNil())
			if result.Content != nil && len(*result.Content) > 1 {
				entries := *result.Content
				for i := 1; i < len(entries); i++ {
					if entries[i-1].Time != nil && entries[i].Time != nil {
						Expect(entries[i-1].Time.Before(*entries[i].Time)).To(BeFalse(),
							"entries should be ordered by time descending")
					}
				}
			}
		})

		It("returns empty results for a non-existent level", func(ctx SpecContext) {
			result, err := c.Project(projectName).Logs().List(ctx,
				client.WithLogItemID(seeded.SuiteID),
				client.WithLogLevel("FATAL"),
				client.WithLogPageSize(50),
			)
			Expect(err).NotTo(HaveOccurred(), "filtering by unused level should not fail")
			Expect(result).NotTo(BeNil())
			if result.Content != nil {
				Expect(*result.Content).To(BeEmpty(),
					"no seeded logs have FATAL level")
			}
		})
	})
})
