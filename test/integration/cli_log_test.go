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

type logListEntry struct {
	ID       int64   `json:"id"`
	UUID     string  `json:"uuid"`
	Level    *string `json:"level"`
	Message  *string `json:"message"`
	ItemID   *int64  `json:"itemId"`
	LaunchID *int64  `json:"launchId"`
}

var _ = Describe("CLI: get log", Label("cli", "logs"), func() {
	var (
		projectName    string
		seededTestData seededTestItems
		seededLogs     seededLogEntries
	)

	BeforeEach(func(ctx SpecContext) {
		projectName = fmt.Sprintf("cli_log_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding one launch with suite and child test items")
		seededTestData, err = seedTestItems(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to seed test items into %q", projectName)

		By("seeding log entries")
		seededLogs, err = seedTestLogEntries(ctx, baseURL, bearerToken, projectName, seededTestData, 3)
		Expect(err).To(Succeed(), "failed to seed log entries into %q", projectName)
		// RP 5.15 async indexing requires longer startup for item/log hierarchy
	}, NodeTimeout(240*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName == "" {
			return
		}

		deleteProject(ctx, baseURL, bearerToken, projectName)
	}, NodeTimeout(15*time.Second))

	Context("listing logs", func() {
		It("renders table output", func() {
			result := runProjectLog(
				projectName,
				"--item", strconv.FormatInt(seededTestData.SuiteID, 10),
			)
			expectCLISuccess(result, "rp get log --item")
			Expect(result.Stdout).To(ContainSubstring("ID"))
			Expect(result.Stdout).To(ContainSubstring("LEVEL"))
			Expect(result.Stdout).To(ContainSubstring("MESSAGE"))
		})

		It("renders json output with expected contract fields", func() {
			result := runProjectLog(projectName,
				"-o", "json",
				"--item", strconv.FormatInt(seededTestData.SuiteID, 10),
			)
			expectCLISuccess(result, "rp get log -o json --item")
			expectJSONArrayObjects(result.Stdout)
			logs := requireLogList(result)
			for _, l := range logs {
				Expect(l.ID).NotTo(BeZero(), "log should have a non-zero id")
				Expect(l.Level).NotTo(BeNil(), "log should have a level")
				Expect(l.Message).NotTo(BeNil(), "log should have a message")
			}
		})

		It("errors when listing without --item or --launch", func() {
			result := runProjectLog(projectName)
			expectCLIError(result, "rp get log (no scope)")
		})
	})

	Context("getting a single log", func() {
		It("returns a log by id", func() {
			Expect(seededLogs.LogIDs).NotTo(BeEmpty())

			logID := strconv.FormatInt(seededLogs.LogIDs[0], 10)
			result := runProjectLog(projectName, logID)
			expectCLISuccess(result, fmt.Sprintf("rp get log %s", logID))
			Expect(result.Stdout).To(ContainSubstring(logID))
		})

		It("returns json output with expected contract fields", func() {
			Expect(seededLogs.LogIDs).NotTo(BeEmpty())

			logID := strconv.FormatInt(seededLogs.LogIDs[0], 10)
			result := runProjectLog(projectName, logID, "-o", "json")
			expectCLISuccess(result, "rp get log <id> -o json")
			expectJSONObject(result.Stdout)

			var single logListEntry
			Expect(json.Unmarshal([]byte(result.Stdout), &single)).To(Succeed())
			Expect(single.ID).To(Equal(seededLogs.LogIDs[0]))
		})

		It("exits non-zero for a missing log", func() {
			result := runProjectLog(projectName, "999999")
			expectCLIError(result, "rp get log 999999")
		})
	})

	Context("error handling", func() {
		It("exits non-zero with bad credentials", func() {
			result := runCLIWithArgs("--url", baseURL, "--token", "invalid-token",
				"get", "log",
				"--item", strconv.FormatInt(seededTestData.SuiteID, 10),
				"-p", projectName)
			expectCLIError(result, "rp get log (invalid token)")
		})

		It("exits non-zero with missing project", func() {
			result := runCLIWithArgs(
				"--url", baseURL, "--token", bearerToken,
				"get", "log",
				"--item", strconv.FormatInt(seededTestData.SuiteID, 10))
			expectCLIError(result, "rp get log (missing project)")
			Expect(result.Stderr).To(ContainSubstring("project"),
				"error message should mention project")
		})

		It("exits non-zero for an invalid log level", func() {
			result := runProjectLog(
				projectName,
				"--item", strconv.FormatInt(seededTestData.SuiteID, 10),
				"--level", "oops",
			)
			expectCLIError(result, "rp get log --level oops")
			Expect(result.Stderr).To(ContainSubstring("invalid log level"))
		})
	})

	It("prints errors as JSON when -o json is set", func() {
		result := runProjectLog(projectName, "-o", "json", "999999")
		expectCLIError(result, "rp get log 999999 -o json")
		expectJSONError(result, "not found")
		Expect(result.Stdout).To(BeEmpty(), "stdout should be empty when JSON error goes to stderr")
	})
})

// runProjectLog runs the CLI with "get log" for the given project and extra args.
func runProjectLog(projectName string, args ...string) cliResult {
	GinkgoHelper()

	fullArgs := append([]string{"get", "log"}, args...)
	fullArgs = append(fullArgs, "-p", projectName)

	return runCLI(fullArgs...)
}

// requireLogList asserts the CLI succeeded and parses the JSON stdout as a log list.
func requireLogList(result cliResult) []logListEntry {
	GinkgoHelper()

	expectCLISuccess(result, "rp get log -o json")

	var logs []logListEntry
	Expect(json.Unmarshal([]byte(result.Stdout), &logs)).To(Succeed(),
		"stdout should be valid JSON array, got: %s", result.Stdout)
	Expect(logs).NotTo(BeEmpty(),
		"JSON output should contain at least one log entry")

	return logs
}
