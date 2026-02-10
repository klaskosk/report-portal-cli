//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

var _ = Describe("CLI: get launch", Label("cli", "launches"), func() {
	var projectName string

	BeforeEach(func(ctx SpecContext) {
		projectName = fmt.Sprintf("cli_launch_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding test launches into " + projectName)
		err = seedTestLaunches(ctx, baseURL, bearerToken, projectName, 3)
		Expect(err).To(Succeed(), "failed to seed test launches into %q", projectName)
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName == "" {
			return
		}

		deleteProject(ctx, baseURL, bearerToken, projectName)
	}, NodeTimeout(15*time.Second))

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

	Context("getting a single launch", func() {
		It("returns a launch by ID", func() {
			launch := requireFirstLaunch(projectName)

			result := runProjectLaunch(projectName, strconv.FormatInt(launch.ID, 10))
			expectCLISuccess(result, fmt.Sprintf("rp get launch %d", launch.ID))
			Expect(result.Stdout).To(ContainSubstring("ID"),
				"single launch output should contain an ID field")
		})

		It("returns a launch by UUID", func() {
			launch := requireFirstLaunch(projectName)

			result := runProjectLaunch(projectName, launch.UUID)
			expectCLISuccess(result, fmt.Sprintf("rp get launch %s", launch.UUID))
			Expect(result.Stdout).To(ContainSubstring(launch.UUID),
				"single launch output should contain the requested UUID %q", launch.UUID)
		})

		It("exits non-zero for a missing launch", func() {
			result := runProjectLaunch(projectName, "999999")
			expectCLIError(result, "rp get launch 999999")
			Expect(result.Stderr).To(ContainSubstring("Error"),
				"stderr should contain an error message for missing launch 999999")
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

	Context("logging verbosity", func() {
		DescribeTable(
			"emits the expected log level",
			func(verbosity string, expectedLevel string) {
				result := runCLIWithArgs(
					"--url", baseURL,
					"--token", bearerToken,
					"--verbosity", verbosity,
					"get", "launch", "-p", projectName,
				)
				description := fmt.Sprintf("rp get launch --verbosity %s", verbosity)
				expectCLISuccess(result, description)
				expectLogLevel(result, expectedLevel, description)
			},
			Entry("info logs at verbosity 1", "1", "INFO"),
			Entry("debug logs at verbosity 2", "2", "DEBUG"),
			Entry("trace logs at verbosity 3", "3", "TRACE"),
		)
	})

	Context("insecure skip verify", func() {
		It("honors the --insecure-skip-verify flag", func() {
			result := runCLIWithArgs(
				"--url", baseURL,
				"--token", bearerToken,
				"--insecure-skip-verify",
				"--verbosity", "1",
				"get", "launch", "-p", projectName,
			)

			expectCLISuccess(result, "rp get launch --insecure-skip-verify")
			Expect(result.Stderr).To(ContainSubstring("insecureSkipVerify=true"))
		})

		It("honors RP_INSECURE_SKIP_VERIFY", func() {
			result := runCLIWithEnv(
				map[string]string{
					"RP_URL":                  baseURL,
					"RP_TOKEN":                bearerToken,
					"RP_INSECURE_SKIP_VERIFY": "true",
				},
				"--verbosity", "1",
				"get", "launch", "-p", projectName,
			)

			expectCLISuccess(result, "rp get launch (RP_INSECURE_SKIP_VERIFY)")
			Expect(result.Stderr).To(ContainSubstring("insecureSkipVerify=true"))
		})
	})

	Context("config file profiles", func() {
		var configPath string

		BeforeEach(func() {
			configYAML := fmt.Sprintf(`
default-profile: default
profiles:
  default:
    url: %q
    token: %q
    default-project: %q
  alternate:
    url: %q
    token: %q
    default-project: %q
`, baseURL, bearerToken, projectName, baseURL, bearerToken, projectName)

			var err error
			configPath, err = writeTempConfig(configYAML)
			Expect(err).NotTo(HaveOccurred(), "failed to write temp config")
			DeferCleanup(func() {
				if configPath != "" {
					Expect(os.RemoveAll(filepath.Dir(configPath))).To(Succeed(),
						"failed to remove temp config dir")
				}
			})
		})

		It("uses the default profile and project from --config", func() {
			result := runCLIWithArgs("--config", configPath, "get", "launch")
			expectCLISuccess(result, "rp get launch --config")
			Expect(result.Stdout).To(ContainSubstring("ID"),
				"output should contain table headers for the default project")
		})

		It("uses the profile override when provided", func() {
			result := runCLIWithArgs(
				"--config", configPath,
				"--profile", "alternate",
				"--verbosity", "1",
				"get", "launch",
			)
			expectCLISuccess(result, "rp get launch --profile alternate")
			Expect(result.Stderr).To(ContainSubstring("profile=alternate"),
				"logs should mention the selected profile")
		})

		It("errors when the requested profile is missing", func() {
			result := runCLIWithArgs(
				"--config", configPath,
				"--profile", "missing",
				"get", "launch",
			)
			expectCLIError(result, "rp get launch --profile missing")
			Expect(result.Stderr).To(ContainSubstring(`profile "missing" not found in config`))
		})
	})

	Context("project resolution with env vars", func() {
		It("uses RP_PROJECT when no -p flag is set", func() {
			result := runCLIWithEnv(
				map[string]string{
					"RP_URL":     baseURL,
					"RP_TOKEN":   bearerToken,
					"RP_PROJECT": projectName,
				},
				"get", "launch",
			)
			expectCLISuccess(result, "rp get launch (RP_PROJECT)")
			Expect(result.Stdout).To(ContainSubstring("ID"),
				"output should contain table headers for the env-provided project")
		})

		It("prefers -p over RP_PROJECT", func() {
			result := runCLIWithEnv(
				map[string]string{
					"RP_URL":     baseURL,
					"RP_TOKEN":   bearerToken,
					"RP_PROJECT": "nonexistent_project",
				},
				"get", "launch", "-p", projectName,
			)
			expectCLISuccess(result, "rp get launch (flag overrides RP_PROJECT)")
		})
	})

	It("prints errors as JSON when -o json is set", func() {
		result := runProjectLaunch(projectName, "-o", "json", "999999")
		expectCLIError(result, "rp get launch 999999 -o json")
		expectJSONError(result, "not found")
		Expect(result.Stderr).To(BeEmpty(), "stderr should be empty in JSON mode")
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

func requireFirstLaunch(projectName string) launchListEntry {
	GinkgoHelper()

	launches := requireLaunchList(runProjectLaunch(projectName, "-o", "json"))

	return launches[0]
}
