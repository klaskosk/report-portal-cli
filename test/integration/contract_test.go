//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Contract: CLI JSON output stability", Label("contract"), func() {
	var (
		c              *client.Client
		projectName    string
		seededTestData seededTestItems
		seededLogs     seededLogEntries
	)

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("contract_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding launches, items, and logs")
		err = seedTestLaunches(ctx, baseURL, bearerToken, projectName, 1)
		Expect(err).To(Succeed(), "failed to seed test launches")

		seededTestData, err = seedTestItems(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to seed test items")

		seededLogs, err = seedTestLogEntries(ctx, c, projectName, seededTestData, 1)
		Expect(err).To(Succeed(), "failed to seed log entries")
	}, NodeTimeout(240*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, c, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("launch JSON output", func() {
		It("list output contains all contracted fields", func() {
			result := runProjectLaunch(projectName, "-o", "json")
			expectCLISuccess(result, "rp get launch -o json")

			var launches []map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &launches)).To(Succeed())
			Expect(launches).NotTo(BeEmpty())

			contractFields := []string{"id", "uuid", "name", "status", "startTime", "number"}
			for _, launch := range launches {
				for _, field := range contractFields {
					Expect(launch).To(HaveKey(field),
						"launch list JSON must include field %q", field)
				}
			}
		})

		It("single launch output contains all contracted fields", func() {
			launch := requireFirstLaunch(projectName)
			result := runProjectLaunch(projectName, strconv.FormatInt(launch.ID, 10), "-o", "json")
			expectCLISuccess(result, "rp get launch <id> -o json")

			var single map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &single)).To(Succeed())

			for _, field := range []string{"id", "uuid", "name", "status", "startTime", "number"} {
				Expect(single).To(HaveKey(field),
					"single launch JSON must include field %q", field)
			}
		})

		It("contracted fields have stable value types", func() {
			result := runProjectLaunch(projectName, "-o", "json")
			expectCLISuccess(result, "rp get launch -o json")

			var launches []map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &launches)).To(Succeed())
			Expect(launches).NotTo(BeEmpty())

			for _, launch := range launches {
				expectJSONNumber(launch, "id")
				expectJSONString(launch, "uuid")
				expectJSONString(launch, "name")
				expectJSONString(launch, "status")
			expectJSONString(launch, "startTime")
			expectJSONNumber(launch, "number")
			}
		})
	})

	Context("item JSON output", func() {
		It("list output contains all contracted fields", func() {
			result := runProjectItem(projectName,
				"-o", "json",
				"--launch", strconv.FormatInt(seededTestData.LaunchID, 10),
			)
			expectCLISuccess(result, "rp get item -o json")

			var items []map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &items)).To(Succeed())
			Expect(items).NotTo(BeEmpty())

			contractFields := []string{"id", "uuid", "name", "type", "status", "startTime", "launchId"}
			for _, item := range items {
				for _, field := range contractFields {
					Expect(item).To(HaveKey(field),
						"item list JSON must include field %q", field)
				}
			}
		})

		It("single item output contains all contracted fields", func() {
			result := runProjectItem(projectName,
				strconv.FormatInt(seededTestData.SuiteID, 10), "-o", "json",
			)
			expectCLISuccess(result, "rp get item <id> -o json")

			var single map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &single)).To(Succeed())

			for _, field := range []string{"id", "uuid", "name", "type", "status", "startTime", "launchId"} {
				Expect(single).To(HaveKey(field),
					"single item JSON must include field %q", field)
			}
		})

		It("contracted fields have stable value types", func() {
			result := runProjectItem(projectName,
				"-o", "json",
				"--launch", strconv.FormatInt(seededTestData.LaunchID, 10),
			)
			expectCLISuccess(result, "rp get item -o json")

			var items []map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &items)).To(Succeed())
			Expect(items).NotTo(BeEmpty())

			for _, item := range items {
				expectJSONNumber(item, "id")
				expectJSONString(item, "uuid")
				expectJSONString(item, "name")
				expectJSONString(item, "type")
				expectJSONString(item, "status")
			expectJSONString(item, "startTime")
			expectJSONNumber(item, "launchId")
			}
		})
	})

	Context("log JSON output", func() {
		It("list output contains all contracted fields", func() {
			result := runProjectLog(projectName,
				"-o", "json",
				"--item", strconv.FormatInt(seededTestData.SuiteID, 10),
			)
			expectCLISuccess(result, "rp get log -o json")

			var logs []map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &logs)).To(Succeed())
			Expect(logs).NotTo(BeEmpty())

			for _, log := range logs {
				Expect(log).To(HaveKey("id"), "log list JSON must include field \"id\"")
				Expect(log).To(HaveKey("uuid"), "log list JSON must include field \"uuid\"")
			}
		})

		It("single log output contains all contracted fields", func() {
			Expect(seededLogs.LogIDs).NotTo(BeEmpty())

			result := runProjectLog(projectName,
				strconv.FormatInt(seededLogs.LogIDs[0], 10), "-o", "json",
			)
			expectCLISuccess(result, "rp get log <id> -o json")

			var single map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &single)).To(Succeed())
			Expect(single).To(HaveKey("id"), "single log JSON must include field \"id\"")
			Expect(single).To(HaveKey("uuid"), "single log JSON must include field \"uuid\"")
		})

		It("contracted fields have stable value types", func() {
			result := runProjectLog(projectName,
				"-o", "json",
				"--item", strconv.FormatInt(seededTestData.SuiteID, 10),
			)
			expectCLISuccess(result, "rp get log -o json")

			var logs []map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &logs)).To(Succeed())
			Expect(logs).NotTo(BeEmpty())

			for _, log := range logs {
				expectJSONNumber(log, "id")
				expectJSONString(log, "uuid")
			}
		})
	})

	Context("project JSON output", func() {
		It("list output is a JSON array of strings", func() {
			result := runCLI("get", "project", "-o", "json")
			expectCLISuccess(result, "rp get project -o json")

			var names []string
			Expect(json.Unmarshal([]byte(result.Stdout), &names)).To(Succeed(),
				"project list JSON must be an array of strings")
			Expect(names).NotTo(BeEmpty())
		})

		It("single project output contains contracted fields", func() {
			result := runCLI("get", "project", projectName, "-o", "json")
			expectCLISuccess(result, "rp get project <name> -o json")

			var project map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &project)).To(Succeed())
			Expect(project).To(HaveKey("projectName"),
				"project detail JSON must include field \"projectName\"")
		})

		It("contracted fields have stable value types", func() {
			result := runCLI("get", "project", projectName, "-o", "json")
			expectCLISuccess(result, "rp get project <name> -o json")

			var project map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &project)).To(Succeed())
			expectJSONString(project, "projectName")
		})
	})

	Context("whoami JSON output", func() {
		It("contains contracted user fields", func() {
			result := runCLI("whoami", "-o", "json")
			expectCLISuccess(result, "rp whoami -o json")

			var user map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &user)).To(Succeed())
			Expect(user).To(HaveKey("userId"), "whoami JSON must include field \"userId\"")
			Expect(user).To(HaveKey("email"), "whoami JSON must include field \"email\"")
		})

		It("contracted fields have stable value types", func() {
			result := runCLI("whoami", "-o", "json")
			expectCLISuccess(result, "rp whoami -o json")

			var user map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &user)).To(Succeed())
			expectJSONString(user, "userId")
			expectJSONString(user, "email")
		})
	})

	Context("JSON error output format", func() {
		DescribeTable("writes stable JSON error to stderr",
			func(args []string, description string) {
				result := runCLIWithArgs(args...)
				expectCLIError(result, description)

				Expect(result.Stderr).NotTo(BeEmpty(), "stderr must contain error output")

				var payload map[string]string
				Expect(json.Unmarshal([]byte(result.Stderr), &payload)).To(Succeed(),
					"stderr must be valid JSON, got: %s", result.Stderr)
				Expect(payload).To(HaveKey("error"),
					"JSON error payload must have an \"error\" key")
				Expect(payload["error"]).NotTo(BeEmpty(),
					"error message must not be empty")
				Expect(result.Stdout).To(BeEmpty(),
					"stdout must be empty when error is written to stderr")
			},
			Entry("rp get launch (not found)",
				[]string{
					"--url", baseURL, "--token", bearerToken,
					"-o", "json",
					"get", "launch", "999999", "-p", "default_personal",
				},
				"launch not found"),
			Entry("rp get item (not found)",
				[]string{
					"--url", baseURL, "--token", bearerToken,
					"-o", "json",
					"get", "item", "999999", "-p", "default_personal",
				},
				"item not found"),
			Entry("rp whoami (bad token)",
				[]string{
					"--url", baseURL, "--token", "invalid-token",
					"-o", "json",
					"whoami",
				},
				"whoami auth error"),
		)

		DescribeTable("error JSON contains only the 'error' key",
			func(args []string, description string) {
				result := runCLIWithArgs(args...)
				expectCLIError(result, description)

				var payload map[string]any
				Expect(json.Unmarshal([]byte(result.Stderr), &payload)).To(Succeed(),
					"stderr must be valid JSON, got: %s", result.Stderr)
				Expect(payload).To(HaveLen(1),
					"JSON error payload must contain exactly one key")
				Expect(payload).To(HaveKey("error"),
					"the single key must be \"error\"")
			},
			Entry("not found error",
				[]string{
					"--url", baseURL, "--token", bearerToken,
					"-o", "json",
					"get", "launch", "999999", "-p", "default_personal",
				},
				"single-key error JSON (not found)"),
			Entry("auth error",
				[]string{
					"--url", baseURL, "--token", "invalid-token",
					"-o", "json",
					"whoami",
				},
				"single-key error JSON (auth)"),
		)
	})
})

// expectJSONNumber asserts that obj[key] is a JSON number (float64 after unmarshalling).
func expectJSONNumber(obj map[string]any, key string) {
	GinkgoHelper()

	Expect(obj).To(HaveKey(key), "expected key %q in JSON object", key)
	_, ok := obj[key].(float64)
	Expect(ok).To(BeTrue(),
		"expected %q to be a JSON number, got %T: %v", key, obj[key], obj[key])
}

// expectJSONString asserts that obj[key] is a JSON string.
func expectJSONString(obj map[string]any, key string) {
	GinkgoHelper()

	Expect(obj).To(HaveKey(key), "expected key %q in JSON object", key)
	_, ok := obj[key].(string)
	Expect(ok).To(BeTrue(),
		"expected %q to be a JSON string, got %T: %v", key, obj[key], obj[key])
}
