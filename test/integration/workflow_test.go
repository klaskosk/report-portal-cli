//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// workflowLaunch is a minimal representation for workflow JSON assertions.
type workflowLaunch struct {
	ID          int64   `json:"id"`
	UUID        string  `json:"uuid"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	StartTime   int64   `json:"startTime"`
	EndTime     *int64  `json:"endTime"`
	Number      int64   `json:"number"`
	Description *string `json:"description"`
}

// workflowItem is a minimal representation for workflow JSON assertions.
type workflowItem struct {
	ID        *int64  `json:"id"`
	UUID      *string `json:"uuid"`
	Name      *string `json:"name"`
	Type      *string `json:"type"`
	Status    *string `json:"status"`
	StartTime *int64  `json:"startTime"`
	EndTime   *int64  `json:"endTime"`
	LaunchID  *int64  `json:"launchId"`
	Parent    *int64  `json:"parent"`
}

// workflowLog is a minimal representation for workflow JSON assertions.
type workflowLog struct {
	ID            int64          `json:"id"`
	UUID          string         `json:"uuid"`
	Level         *string        `json:"level"`
	Message       *string        `json:"message"`
	ItemID        *int64         `json:"itemId"`
	LaunchID      *int64         `json:"launchId"`
	BinaryContent map[string]any `json:"binaryContent"`
}

var _ = Describe("End-to-end read workflow", Label("workflow"), Ordered, func() {
	var (
		projectName         string
		seededTestData      seededTestItems
		attachmentDataID    int64

		// Identifiers discovered during the workflow steps and passed to subsequent steps.
		discoveredLaunchID string
		discoveredItemID   string
		discoveredLogID    string
	)

	BeforeAll(func(ctx SpecContext) {
		projectName = fmt.Sprintf("workflow_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding one launch with suite and child test items")
		seededTestData, err = seedTestItems(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to seed test items into %q", projectName)

		By("seeding log entries")
		_, err = seedTestLogEntries(ctx, baseURL, bearerToken, projectName, seededTestData, 2)
		Expect(err).To(Succeed(), "failed to seed log entries into %q", projectName)

		By("seeding a log entry with an attachment")
		attachmentDataID, err = seedLogWithAttachment(ctx, baseURL, bearerToken, projectName, seededTestData)
		if err != nil {
			GinkgoWriter.Printf("warning: could not seed attachment: %v\n", err)
		}
	}, NodeTimeout(120*time.Second))

	AfterAll(func(ctx SpecContext) {
		if projectName == "" {
			return
		}

		deleteProject(ctx, baseURL, bearerToken, projectName)
	}, NodeTimeout(15*time.Second))

	It("step 1: list launches in JSON", func() {
		result := runCLI("get", "launch", "-p", projectName, "-o", "json")
		expectCLISuccess(result, "rp get launch -o json")
		expectJSONArrayObjects(result.Stdout)

		var launches []workflowLaunch
		Expect(json.Unmarshal([]byte(result.Stdout), &launches)).To(Succeed(),
			"stdout should be valid JSON array, got: %s", result.Stdout)
		Expect(launches).NotTo(BeEmpty())

		// Validate contract fields on first entry.
		first := launches[0]
		Expect(first.ID).NotTo(BeZero(), "launch should have a non-zero id")
		Expect(first.UUID).NotTo(BeEmpty(), "launch should have a non-empty uuid")
		Expect(first.Name).NotTo(BeEmpty(), "launch should have a non-empty name")
		Expect(first.Status).NotTo(BeEmpty(), "launch should have a non-empty status")
		Expect(first.StartTime).NotTo(BeZero(), "launch should have a non-zero startTime")
		Expect(first.Number).NotTo(BeZero(), "launch should have a non-zero number")

		discoveredLaunchID = strconv.FormatInt(first.ID, 10)
	})

	It("step 2: get single launch in JSON", func() {
		Expect(discoveredLaunchID).NotTo(BeEmpty(), "discoveredLaunchID should have been set by step 1")

		result := runCLI("get", "launch", discoveredLaunchID, "-p", projectName, "-o", "json")
		expectCLISuccess(result, "rp get launch <id> -o json")
		expectJSONObject(result.Stdout)

		var launch workflowLaunch
		Expect(json.Unmarshal([]byte(result.Stdout), &launch)).To(Succeed(),
			"stdout should be valid JSON, got: %s", result.Stdout)
		Expect(launch.ID).To(Equal(mustParseInt64(discoveredLaunchID)))
		Expect(launch.UUID).NotTo(BeEmpty())
		Expect(launch.Status).NotTo(BeEmpty(), "single launch should have a status")
		Expect(launch.StartTime).NotTo(BeZero(), "single launch should have a startTime")
		Expect(launch.Number).NotTo(BeZero(), "single launch should have a number")
	})

	It("step 3: list items for launch in JSON", func() {
		Expect(discoveredLaunchID).NotTo(BeEmpty(), "discoveredLaunchID should have been set by step 1")

		result := runCLI("get", "item", "--launch", discoveredLaunchID, "-p", projectName, "-o", "json")
		expectCLISuccess(result, "rp get item --launch <id> -o json")
		expectJSONArrayObjects(result.Stdout)

		var items []workflowItem
		Expect(json.Unmarshal([]byte(result.Stdout), &items)).To(Succeed(),
			"stdout should be valid JSON array, got: %s", result.Stdout)
		Expect(items).NotTo(BeEmpty())

		// Validate contract fields.
		first := items[0]
		Expect(first.ID).NotTo(BeNil(), "item should have an id")
		Expect(*first.ID).NotTo(BeZero())
		Expect(first.UUID).NotTo(BeNil(), "item should have a uuid")
		Expect(*first.UUID).NotTo(BeEmpty(), "item uuid should be non-empty")
		Expect(first.Name).NotTo(BeNil(), "item should have a name")
		Expect(first.Type).NotTo(BeNil(), "item should have a type")
		Expect(first.Status).NotTo(BeNil(), "item should have a status")
		Expect(first.StartTime).NotTo(BeNil(), "item should have a startTime")
		Expect(first.LaunchID).NotTo(BeNil(), "item should have a launchId")

		// Prefer the seeded suite item because seedTestLogEntries attaches logs to it.
		selected := first
		for index := range items {
			if items[index].ID != nil && *items[index].ID == seededTestData.SuiteID {
				selected = items[index]

				break
			}
		}

		Expect(selected.ID).NotTo(BeNil(), "selected item should have an id")
		discoveredItemID = strconv.FormatInt(*selected.ID, 10)
	})

	It("step 4: get single item in JSON", func() {
		Expect(discoveredItemID).NotTo(BeEmpty(), "discoveredItemID should have been set by step 3")

		result := runCLI("get", "item", discoveredItemID, "-p", projectName, "-o", "json")
		expectCLISuccess(result, "rp get item <id> -o json")
		expectJSONObject(result.Stdout)

		var item workflowItem
		Expect(json.Unmarshal([]byte(result.Stdout), &item)).To(Succeed(),
			"stdout should be valid JSON, got: %s", result.Stdout)
		Expect(item.ID).NotTo(BeNil())
		Expect(*item.ID).To(Equal(mustParseInt64(discoveredItemID)))
		Expect(item.UUID).NotTo(BeNil(), "single item should have a uuid")
		Expect(item.Name).NotTo(BeNil(), "single item should have a name")
		Expect(item.Status).NotTo(BeNil(), "single item should have a status")
		Expect(item.StartTime).NotTo(BeNil(), "single item should have a startTime")
		Expect(item.LaunchID).NotTo(BeNil(), "single item should have a launchId")
	})

	It("step 5: list logs for item in JSON", func() {
		Expect(discoveredItemID).NotTo(BeEmpty(), "discoveredItemID should have been set by step 3")

		result := runCLI("get", "log", "--item", discoveredItemID, "-p", projectName, "-o", "json")
		expectCLISuccess(result, "rp get log --item <id> -o json")
		expectJSONArrayObjects(result.Stdout)

		var logs []workflowLog
		Expect(json.Unmarshal([]byte(result.Stdout), &logs)).To(Succeed(),
			"stdout should be valid JSON array, got: %s", result.Stdout)
		Expect(logs).NotTo(BeEmpty())

		// Validate contract fields.
		first := logs[0]
		Expect(first.ID).NotTo(BeZero(), "log should have a non-zero id")
		Expect(first.Level).NotTo(BeNil(), "log should have a level")
		Expect(first.Message).NotTo(BeNil(), "log should have a message")

		discoveredLogID = strconv.FormatInt(first.ID, 10)
	})

	It("step 6: get single log in JSON", func() {
		Expect(discoveredLogID).NotTo(BeEmpty(), "discoveredLogID should have been set by step 5")

		result := runCLI("get", "log", discoveredLogID, "-p", projectName, "-o", "json")
		expectCLISuccess(result, "rp get log <id> -o json")
		expectJSONObject(result.Stdout)

		var log workflowLog
		Expect(json.Unmarshal([]byte(result.Stdout), &log)).To(Succeed(),
			"stdout should be valid JSON, got: %s", result.Stdout)
		Expect(log.ID).To(Equal(mustParseInt64(discoveredLogID)))
	})

	It("step 7: download attachment", func() {
		if attachmentDataID == 0 {
			Skip("attachment seeding was not successful")
		}

		result := runCLI("download", strconv.FormatInt(attachmentDataID, 10), "-p", projectName)
		expectCLISuccess(result, "rp download <data-id>")
		Expect(result.Stdout).NotTo(BeEmpty(), "downloaded content should not be empty")
	})
})

// mustParseInt64 parses s as a base-10 int64, failing the test on error.
func mustParseInt64(s string) int64 {
	GinkgoHelper()

	v, err := strconv.ParseInt(s, 10, 64)
	Expect(err).NotTo(HaveOccurred(), "mustParseInt64(%q) failed", s)

	return v
}
