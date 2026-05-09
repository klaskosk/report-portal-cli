//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type launchListEntry struct {
	ID          int64   `json:"id"`
	UUID        string  `json:"uuid"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	StartTime   string  `json:"startTime"`
	EndTime     *string `json:"endTime"`
	Number      int64   `json:"number"`
	Description *string `json:"description"`
}

var _ = Describe("CLI: launch", Label("cli", "launches"), func() {
	var (
		c           *client.Client
		projectName string
	)

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("cli_launch_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding test launches into " + projectName)
		err = seedTestLaunches(ctx, baseURL, bearerToken, projectName, 3)
		Expect(err).To(Succeed(), "failed to seed test launches into %q", projectName)
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName == "" {
			return
		}

		deleteProject(ctx, c, projectName)
	}, NodeTimeout(15*time.Second))

	Context("listing launches", func() {
		DescribeTable(
			"output formats",
			func(args []string, description string, assert func(cliResult)) {
				result := runProjectLaunchList(projectName, args...)
				expectCLISuccess(result, description)
				assert(result)
			},
			Entry("table output", []string{}, "rp launch list", func(result cliResult) {
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
				"rp launch list -o json",
				func(result cliResult) {
					expectJSONArrayObjects(result.Stdout)
					launches := requireLaunchList(result)
					for _, l := range launches {
						Expect(l.ID).NotTo(BeZero(), "launch should have a non-zero id")
						Expect(l.UUID).NotTo(BeEmpty(), "launch should have a non-empty uuid")
						Expect(l.Name).NotTo(BeEmpty(), "launch should have a non-empty name")
						Expect(l.Status).NotTo(BeEmpty(), "launch should have a non-empty status")
						Expect(l.StartTime).NotTo(BeEmpty(), "launch should have a non-empty startTime")
						Expect(l.Number).NotTo(BeZero(), "launch should have a non-zero number")
					}
				},
			),
			Entry("go-template output", []string{"-o", `go-template={{range .}}{{.Name}} {{end}}`},
				"rp launch list -o go-template=...", func(result cliResult) {
					Expect(result.Stdout).NotTo(BeEmpty(),
						"go-template output should produce non-empty launch names")
				}),
		)
	})

	Context("getting a single launch", func() {
		It("returns a launch by ID", func() {
			launch := requireFirstLaunch(projectName)

			result := runProjectLaunchGet(projectName, strconv.FormatInt(launch.ID, 10))
			expectCLISuccess(result, fmt.Sprintf("rp launch get %d", launch.ID))
			Expect(result.Stdout).To(ContainSubstring("ID"),
				"single launch output should contain an ID field")
		})

		It("returns a launch by UUID", func() {
			launch := requireFirstLaunch(projectName)

			result := runProjectLaunchGet(projectName, launch.UUID)
			expectCLISuccess(result, fmt.Sprintf("rp launch get %s", launch.UUID))
			Expect(result.Stdout).To(ContainSubstring(launch.UUID),
				"single launch output should contain the requested UUID %q", launch.UUID)
		})

		It("returns json output with expected contract fields", func() {
			launch := requireFirstLaunch(projectName)
			result := runProjectLaunchGet(projectName, strconv.FormatInt(launch.ID, 10), "-o", "json")
			expectCLISuccess(result, "rp launch get <id> -o json")
			expectJSONObject(result.Stdout)

			var single launchListEntry
			Expect(json.Unmarshal([]byte(result.Stdout), &single)).To(Succeed())
			Expect(single.ID).To(Equal(launch.ID))
			Expect(single.UUID).To(Equal(launch.UUID))
		})

		It("exits non-zero for a missing launch", func() {
			result := runProjectLaunchGet(projectName, "999999")
			expectCLIError(result, "rp launch get 999999")
			Expect(result.Stderr).To(ContainSubstring("Error"),
				"stderr should contain an error message for missing launch 999999")
		})
	})

	Context("error handling", func() {
		It("exits non-zero with bad credentials", func() {
			result := runCLIWithArgs("--url", baseURL, "--token", "invalid-token",
				"launch", "list", "-p", projectName)
			expectCLIError(result, "rp launch list (invalid token)")
		})

		It("exits non-zero with missing required flags", func() {
			result := runCLIWithArgs("launch", "list")
			expectCLIError(result, "rp launch list (missing url/token)")
		})

		DescribeTable("prints a user-friendly error to stderr for missing inputs",
			func(argsBuilder func() []string, expectedSubstring string) {
				GinkgoHelper()
				result := runCLIWithArgs(argsBuilder()...)
				expectCLIError(result, "rp launch list (missing required inputs)")
				Expect(result.Stderr).To(ContainSubstring("Error"),
					"stderr should contain an Error line")
				Expect(result.Stderr).To(ContainSubstring(expectedSubstring),
					"error message should mention %s", expectedSubstring)
			},
			Entry("missing URL", func() []string {
				return []string{"--token", "some-token", "launch", "list", "-p", projectName}
			}, "URL"),
			Entry("missing token", func() []string {
				return []string{"--url", baseURL, "launch", "list", "-p", projectName}
			}, "token"),
			Entry("missing project", func() []string {
				return []string{"--url", baseURL, "--token", bearerToken, "launch", "list"}
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
				"launch", "list", "-p", projectName,
			)
			expectCLISuccess(cmd, "rp launch list (env vars)")
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
				"launch", "list", "-p", projectName,
			)
			expectCLISuccess(cmd, "rp launch list (flags override env)")
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
					"launch", "list", "-p", projectName,
				)
				description := fmt.Sprintf("rp launch list --verbosity %s", verbosity)
				expectCLISuccess(result, description)
				expectLogLevel(result, expectedLevel, description)
			},
			Entry("info logs at verbosity 1", "1", "INFO"),
			Entry("debug logs at verbosity 2", "2", "DEBUG"),
			Entry("debug logs at verbosity 3", "3", "DEBUG"),
		)
	})

	Context("insecure skip verify", func() {
		It("honors the --insecure-skip-verify flag", func() {
			result := runCLIWithArgs(
				"--url", baseURL,
				"--token", bearerToken,
				"--insecure-skip-verify",
				"--verbosity", "1",
				"launch", "list", "-p", projectName,
			)

			expectCLISuccess(result, "rp launch list --insecure-skip-verify")
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
				"launch", "list", "-p", projectName,
			)

			expectCLISuccess(result, "rp launch list (RP_INSECURE_SKIP_VERIFY)")
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
			result := runCLIWithArgs("--config", configPath, "launch", "list")
			expectCLISuccess(result, "rp launch list --config")
			Expect(result.Stdout).To(ContainSubstring("ID"),
				"output should contain table headers for the default project")
		})

		It("uses the profile override when provided", func() {
			result := runCLIWithArgs(
				"--config", configPath,
				"--profile", "alternate",
				"--verbosity", "1",
				"launch", "list",
			)
			expectCLISuccess(result, "rp launch list --profile alternate")
			Expect(result.Stderr).To(ContainSubstring("profile=alternate"),
				"logs should mention the selected profile")
		})

		It("errors when the requested profile is missing", func() {
			result := runCLIWithArgs(
				"--config", configPath,
				"--profile", "missing",
				"launch", "list",
			)
			expectCLIError(result, "rp launch list --profile missing")
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
				"launch", "list",
			)
			expectCLISuccess(result, "rp launch list (RP_PROJECT)")
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
				"launch", "list", "-p", projectName,
			)
			expectCLISuccess(result, "rp launch list (flag overrides RP_PROJECT)")
		})
	})

	It("prints errors as JSON when -o json is set", func() {
		result := runProjectLaunchGet(projectName, "999999", "-o", "json")
		expectCLIError(result, "rp launch get 999999 -o json")
		expectJSONError(result, "not found")
		Expect(result.Stdout).To(BeEmpty(), "stdout should be empty when JSON error goes to stderr")
	})

	Context("filter flags", func() {
		It("filters launches by user", func() {
			result := runProjectLaunchList(projectName, "--user", "default")
			expectCLISuccess(result, "rp launch list --user default")
			Expect(result.Stdout).To(ContainSubstring("ID"),
				"filtered output should contain table headers")
		})

		It("filters launches with --latest flag", func() {
			result := runProjectLaunchList(projectName, "--latest")
			expectCLISuccess(result, "rp launch list --latest")
			Expect(result.Stdout).To(ContainSubstring("ID"),
				"latest launches output should contain table headers")
		})

		It("filters launches with --debug flag", func() {
			result := runProjectLaunchList(projectName, "--debug")
			expectCLISuccess(result, "rp launch list --debug")
			// May be empty but should succeed
		})

		It("fails when both --latest and --debug are used", func() {
			result := runProjectLaunchList(projectName, "--latest", "--debug")
			expectCLIError(result, "rp launch list --latest --debug")
			// Error message may be in stdout or stderr depending on Cobra settings
			combined := result.Stdout + result.Stderr
			Expect(combined).To(ContainSubstring("cannot use both"),
				"error should mention conflicting flags")
		})

		It("filters launches with --description flag", func() {
			result := runProjectLaunchList(projectName, "--description", "test")
			expectCLISuccess(result, "rp launch list --description test")
		})

		It("filters launches with --has-retries flag", func() {
			result := runProjectLaunchList(projectName, "--has-retries")
			expectCLISuccess(result, "rp launch list --has-retries")
		})

		It("applies page-size flag", func() {
			result := runProjectLaunchList(projectName, "--page-size", "5")
			expectCLISuccess(result, "rp launch list --page-size 5")
			Expect(result.Stdout).To(ContainSubstring("ID"),
				"page-size filtered output should contain table headers")
		})

		It("outputs JSON with filter flags", func() {
			result := runProjectLaunchList(projectName, "-o", "json", "--page-size", "5")
			expectCLISuccess(result, "rp launch list -o json with filters")
			requireLaunchList(result)
		})

		It("uses --latest with pagination", func() {
			result := runProjectLaunchList(projectName, "--latest", "--page-size", "2")
			expectCLISuccess(result, "rp launch list --latest --page-size 2")
		})
	})
})

// runProjectLaunchList runs the CLI with "launch list" for the given project and extra args.
func runProjectLaunchList(projectName string, args ...string) cliResult {
	GinkgoHelper()

	fullArgs := append([]string{"launch", "list"}, args...)
	fullArgs = append(fullArgs, "-p", projectName)

	return runCLI(fullArgs...)
}

// runProjectLaunchGet runs the CLI with "launch get" for the given project and ID/UUID.
func runProjectLaunchGet(projectName string, idOrUUID string, args ...string) cliResult {
	GinkgoHelper()

	fullArgs := []string{"launch", "get", idOrUUID}
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, "-p", projectName)

	return runCLI(fullArgs...)
}

// requireLaunchList asserts the CLI succeeded and parses the JSON stdout as a launch list.
func requireLaunchList(result cliResult) []launchListEntry {
	GinkgoHelper()

	expectCLISuccess(result, "rp launch list -o json")

	var launches []launchListEntry
	Expect(json.Unmarshal([]byte(result.Stdout), &launches)).To(Succeed(),
		"stdout should be valid JSON array, got: %s", result.Stdout)
	Expect(launches).NotTo(BeEmpty(),
		"JSON output should contain at least one launch")

	return launches
}

// requireFirstLaunch lists launches for the project and returns the first entry.
func requireFirstLaunch(projectName string) launchListEntry {
	GinkgoHelper()

	launches := requireLaunchList(runProjectLaunchList(projectName, "-o", "json"))

	return launches[0]
}
