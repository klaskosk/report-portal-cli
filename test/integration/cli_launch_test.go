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

type launchListEntry struct {
	ID   int64  `json:"id"`
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

var _ = Describe("CLI: get launch", Label("cli", "launches"), Ordered, func() {
	var projectName string

	BeforeAll(func(ctx SpecContext) {
		projectName = fmt.Sprintf("cli_launch_test_%d", GinkgoRandomSeed())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		DeferCleanup(func(ctx SpecContext) {
			deleteProject(ctx, baseURL, bearerToken, projectName)
		}, NodeTimeout(15*time.Second))

		By("seeding demo data into " + projectName)
		err = seedDemoData(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to seed demo data into %q", projectName)
	}, NodeTimeout(60*time.Second))

	Context("listing launches", func() {
		DescribeTable(
			"output formats",
			func(args []string, description string, assert func(cliResult)) {
				result := runProjectLaunch(projectName, args...)
				expectCLISuccess(result, description)
				assert(result)
			},
			Entry("table output", []string{}, "rp get launch", func(result cliResult) {
				Expect(result.Stdout).To(ContainSubstring("ID"),
					"table output should contain an ID column header")
				Expect(result.Stdout).To(ContainSubstring("UUID"),
					"table output should contain a UUID column header")
				Expect(result.Stdout).To(ContainSubstring("NAME"),
					"table output should contain a NAME column header")
				Expect(result.Stdout).To(ContainSubstring("STATUS"),
					"table output should contain a STATUS column header")
			}),
			Entry(
				"json output",
				[]string{"-o", "json"},
				"rp get launch -o json",
				func(result cliResult) {
					requireLaunchList(result)
				},
			),
			Entry("go-template output", []string{"-o", `go-template={{range .}}{{.Name}} {{end}}`},
				"rp get launch -o go-template=...", func(result cliResult) {
					Expect(result.Stdout).NotTo(BeEmpty(),
						"go-template output should produce non-empty launch names")
				}),
		)
	})

	Context("getting a single launch by ID", func() {
		It("returns the launch", func() {
			launches := requireLaunchList(runProjectLaunch(projectName, "-o", "json"))
			launchID := launches[0].ID

			result := runProjectLaunch(projectName, strconv.FormatInt(launchID, 10))
			expectCLISuccess(result, fmt.Sprintf("rp get launch %d", launchID))
			Expect(result.Stdout).To(ContainSubstring("ID"),
				"single launch output should contain an ID field")
		})

		It("exits non-zero for a missing launch", func() {
			result := runProjectLaunch(projectName, "999999")
			expectCLIError(result, "rp get launch 999999")
			Expect(result.Stderr).To(ContainSubstring("Error"),
				"stderr should contain an error message for missing launch 999999")
		})
	})

	Context("getting a single launch by UUID", func() {
		It("returns the launch", func() {
			launches := requireLaunchList(runProjectLaunch(projectName, "-o", "json"))
			uuid := launches[0].UUID

			result := runProjectLaunch(projectName, uuid)
			expectCLISuccess(result, fmt.Sprintf("rp get launch %s", uuid))
			Expect(result.Stdout).To(ContainSubstring(uuid),
				"single launch output should contain the requested UUID %q", uuid)
		})
	})

	Context("error handling", func() {
		It("exits non-zero with bad credentials", func() {
			result := runCLIWithArgs("--url", baseURL, "--token", "invalid-token",
				"get", "launch", "-p", projectName)
			expectCLIError(result, "rp get launch (invalid token)")
		})

		It("exits non-zero with missing required flags", func() {
			result := runCLIWithArgs("get", "launch")
			expectCLIError(result, "rp get launch (missing url/token)")
		})

		DescribeTable("prints a user-friendly error to stderr for missing inputs",
			func(argsBuilder func() []string, expectedSubstring string) {
				GinkgoHelper()
				result := runCLIWithArgs(argsBuilder()...)
				expectCLIError(result, "rp get launch (missing required inputs)")
				Expect(result.Stderr).To(ContainSubstring("Error"),
					"stderr should contain an Error line")
				Expect(result.Stderr).To(ContainSubstring(expectedSubstring),
					"error message should mention %s", expectedSubstring)
			},
			Entry("missing URL", func() []string {
				return []string{"--token", "some-token", "get", "launch", "-p", projectName}
			}, "URL"),
			Entry("missing token", func() []string {
				return []string{"--url", baseURL, "get", "launch", "-p", projectName}
			}, "token"),
			Entry("missing project", func() []string {
				return []string{"--url", baseURL, "--token", bearerToken, "get", "launch"}
			}, "project"),
		)
	})

	Context("config resolution precedence", func() {
		It("accepts connection details via environment variables", func() {
			cmd := runCLIWithEnv(
				map[string]string{
					"RP_URL":   baseURL,
					"RP_TOKEN": bearerToken,
				},
				"get", "launch", "-p", projectName,
			)
			expectCLISuccess(cmd, "rp get launch (env vars)")
			Expect(cmd.Stdout).To(ContainSubstring("ID"),
				"output should still contain table headers")
		})

		It("flags override environment variables", func() {
			cmd := runCLIWithEnv(
				map[string]string{
					"RP_URL":   "http://wrong-host:9999",
					"RP_TOKEN": "wrong-token",
				},
				"--url", baseURL, "--token", bearerToken,
				"get", "launch", "-p", projectName,
			)
			expectCLISuccess(cmd, "rp get launch (flags override env)")
		})
	})
})

func runProjectLaunch(projectName string, args ...string) cliResult {
	GinkgoHelper()

	fullArgs := append([]string{"get", "launch"}, args...)
	fullArgs = append(fullArgs, "-p", projectName)

	return runCLI(fullArgs...)
}

func expectCLISuccess(result cliResult, description string) {
	GinkgoHelper()
	Expect(result.ExitCode).To(Equal(0),
		"%s should exit 0; stderr: %s", description, result.Stderr)
}

func expectCLIError(result cliResult, description string) {
	GinkgoHelper()
	Expect(result.ExitCode).NotTo(Equal(0),
		"%s should exit non-zero; stderr: %s", description, result.Stderr)
}

func requireLaunchList(result cliResult) []launchListEntry {
	GinkgoHelper()

	expectCLISuccess(result, "rp get launch -o json")

	var launches []launchListEntry
	Expect(json.Unmarshal([]byte(result.Stdout), &launches)).To(Succeed(),
		"stdout should be valid JSON array, got: %s", result.Stdout)
	Expect(launches).NotTo(BeEmpty(),
		"JSON output should contain at least one launch")

	return launches
}
