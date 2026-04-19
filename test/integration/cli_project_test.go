//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CLI: get project", Label("cli", "project"), func() {
	It("lists accessible projects", func() {
		result := runCLI("get", "project")
		expectCLISuccess(result, "rp get project")
		Expect(result.Stdout).To(ContainSubstring("PROJECT NAME"))
	})

	It("lists projects in JSON format", func() {
		result := runCLI("get", "project", "-o", "json")
		expectCLISuccess(result, "rp get project -o json")

		var names []string
		Expect(json.Unmarshal([]byte(result.Stdout), &names)).To(Succeed(),
			"stdout should be valid JSON array of strings, got: %s", result.Stdout)
		Expect(names).NotTo(BeEmpty())
	})

	Context("getting a specific project", func() {
		var (
			c           *client.Client
			projectName string
		)

		BeforeEach(func(ctx SpecContext) {
			c = newClient()
			projectName = fmt.Sprintf("cli_project_test_%d", time.Now().UnixNano())
			err := createProject(ctx, c, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)
		}, NodeTimeout(30*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, c, projectName)
			}
		}, NodeTimeout(15*time.Second))

		It("returns project details in table format", func() {
			result := runCLI("get", "project", projectName)
			expectCLISuccess(result, "rp get project "+projectName)
			Expect(result.Stdout).To(ContainSubstring(projectName))
		})

		It("returns project details in JSON format", func() {
			result := runCLI("get", "project", projectName, "-o", "json")
			expectCLISuccess(result, "rp get project -o json")

			var project map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &project)).To(Succeed(),
				"stdout should be valid JSON, got: %s", result.Stdout)
			Expect(project).To(HaveKey("projectName"))
			Expect(project["projectName"]).To(Equal(projectName))
		})
	})

	It("exits non-zero for a non-existent project", func() {
		result := runCLI("get", "project", "nonexistent_project_99999")
		expectCLIError(result, "rp get project nonexistent")
	})

	It("prints errors as JSON when -o json is set", func() {
		result := runCLI("get", "project", "nonexistent_project_99999", "-o", "json")
		expectCLIError(result, "rp get project nonexistent -o json")
		expectJSONError(result, "")
		Expect(result.Stdout).To(BeEmpty(), "stdout should be empty when JSON error goes to stderr")
	})
})
