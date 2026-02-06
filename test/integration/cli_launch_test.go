//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CLI: get launch", Label("cli", "launches"), Ordered, func() {
	var projectName string

	BeforeAll(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		projectName = fmt.Sprintf("cli_launch_test_%d", GinkgoRandomSeed())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).NotTo(HaveOccurred(), "failed to create test project %q", projectName)

		By("seeding demo data into " + projectName)
		err = seedDemoData(ctx, baseURL, bearerToken, projectName)
		Expect(err).NotTo(HaveOccurred(), "failed to seed demo data into %q", projectName)
	})

	AfterAll(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		deleteProject(ctx, baseURL, bearerToken, projectName)
	})

	Context("listing launches", func() {
		It("outputs a table by default", func() {
			result := runCLI("get", "launch", "-p", projectName)
			Expect(result.ExitCode).To(Equal(0),
				"'rp get launch' should exit 0, got %d; stderr: %s", result.ExitCode, result.Stderr)
			Expect(result.Stdout).To(ContainSubstring("ID"),
				"table output should contain an ID column header")
			Expect(result.Stdout).To(ContainSubstring("UUID"),
				"table output should contain a UUID column header")
			Expect(result.Stdout).To(ContainSubstring("NAME"),
				"table output should contain a NAME column header")
			Expect(result.Stdout).To(ContainSubstring("STATUS"),
				"table output should contain a STATUS column header")
		})

		It("outputs JSON with --output json", func() {
			result := runCLI("get", "launch", "-p", projectName, "-o", "json")
			Expect(result.ExitCode).To(Equal(0),
				"'rp get launch -o json' should exit 0, got %d; stderr: %s", result.ExitCode, result.Stderr)

			var parsed []map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &parsed)).To(Succeed(),
				"stdout should be valid JSON array, got: %s", result.Stdout)
			Expect(parsed).NotTo(BeEmpty(),
				"JSON output should contain at least one launch")
		})

		It("supports go-template output", func() {
			result := runCLI("get", "launch", "-p", projectName,
				"-o", `go-template={{range .}}{{.Name}} {{end}}`)
			Expect(result.ExitCode).To(Equal(0),
				"'rp get launch -o go-template=...' should exit 0, got %d; stderr: %s",
				result.ExitCode, result.Stderr)
			Expect(result.Stdout).NotTo(BeEmpty(),
				"go-template output should produce non-empty launch names")
		})
	})

	Context("getting a single launch by ID", func() {
		It("returns the launch", func() {
			listResult := runCLI("get", "launch", "-p", projectName, "-o", "json")
			Expect(listResult.ExitCode).To(Equal(0),
				"listing launches to discover a known ID should exit 0; stderr: %s", listResult.Stderr)

			var launches []map[string]any
			Expect(json.Unmarshal([]byte(listResult.Stdout), &launches)).To(Succeed(),
				"list output should be valid JSON, got: %s", listResult.Stdout)
			Expect(launches).NotTo(BeEmpty(),
				"launch list should contain at least one launch for ID lookup")

			launchID := int64(launches[0]["id"].(float64))

			result := runCLI("get", "launch", strconv.FormatInt(launchID, 10), "-p", projectName)
			Expect(result.ExitCode).To(Equal(0),
				"'rp get launch %d' should exit 0, got %d; stderr: %s",
				launchID, result.ExitCode, result.Stderr)
			Expect(result.Stdout).To(ContainSubstring("ID"),
				"single launch output should contain an ID field")
		})

		It("exits non-zero for a missing launch", func() {
			result := runCLI("get", "launch", "999999", "-p", projectName)
			Expect(result.ExitCode).NotTo(Equal(0),
				"'rp get launch 999999' should exit non-zero for a missing launch")
			Expect(result.Stderr).To(ContainSubstring("Error"),
				"stderr should contain an error message for missing launch 999999")
		})
	})

	Context("getting a single launch by UUID", func() {
		It("returns the launch", func() {
			listResult := runCLI("get", "launch", "-p", projectName, "-o", "json")
			Expect(listResult.ExitCode).To(Equal(0),
				"listing launches to discover a known UUID should exit 0; stderr: %s", listResult.Stderr)

			var launches []map[string]any
			Expect(json.Unmarshal([]byte(listResult.Stdout), &launches)).To(Succeed(),
				"list output should be valid JSON, got: %s", listResult.Stdout)
			Expect(launches).NotTo(BeEmpty(),
				"launch list should contain at least one launch for UUID lookup")

			uuid := launches[0]["uuid"].(string)

			result := runCLI("get", "launch", uuid, "-p", projectName)
			Expect(result.ExitCode).To(Equal(0),
				"'rp get launch %s' should exit 0, got %d; stderr: %s",
				uuid, result.ExitCode, result.Stderr)
			Expect(result.Stdout).To(ContainSubstring(uuid),
				"single launch output should contain the requested UUID %q", uuid)
		})
	})

	Context("error handling", func() {
		It("exits non-zero with bad credentials", func() {
			result := runCLIWithArgs("--url", baseURL, "--token", "invalid-token",
				"get", "launch", "-p", projectName)
			Expect(result.ExitCode).NotTo(Equal(0),
				"'rp get launch' with invalid token should exit non-zero")
		})

		It("exits non-zero with missing required flags", func() {
			result := runCLIWithArgs("get", "launch")
			Expect(result.ExitCode).NotTo(Equal(0),
				"'rp get launch' without --url and --token should exit non-zero")
		})

		It("prints a user-friendly error to stderr when URL is missing", func() {
			result := runCLIWithArgs("--token", "some-token", "get", "launch", "-p", projectName)
			Expect(result.ExitCode).NotTo(Equal(0),
				"should exit non-zero when --url is missing")
			Expect(result.Stderr).To(ContainSubstring("Error"),
				"stderr should contain an Error line")
			Expect(result.Stderr).To(ContainSubstring("URL"),
				"error message should mention URL")
		})

		It("prints a user-friendly error to stderr when token is missing", func() {
			result := runCLIWithArgs("--url", baseURL, "get", "launch", "-p", projectName)
			Expect(result.ExitCode).NotTo(Equal(0),
				"should exit non-zero when --token is missing")
			Expect(result.Stderr).To(ContainSubstring("Error"),
				"stderr should contain an Error line")
			Expect(result.Stderr).To(ContainSubstring("token"),
				"error message should mention token")
		})

		It("prints a user-friendly error when project is missing", func() {
			result := runCLIWithArgs("--url", baseURL, "--token", bearerToken, "get", "launch")
			Expect(result.ExitCode).NotTo(Equal(0),
				"should exit non-zero when project is missing")
			Expect(result.Stderr).To(ContainSubstring("Error"),
				"stderr should contain an Error line")
			Expect(result.Stderr).To(ContainSubstring("project"),
				"error message should mention project")
		})
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
			Expect(cmd.ExitCode).To(Equal(0),
				"should succeed when URL/token come from env vars; stderr: %s", cmd.Stderr)
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
			Expect(cmd.ExitCode).To(Equal(0),
				"flags should override bad env vars; stderr: %s", cmd.Stderr)
		})
	})
})
