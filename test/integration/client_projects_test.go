//go:build integration

package integration_test

import (
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Projects", Label("client", "projects"), func() {
	var c *client.Client

	BeforeEach(func() {
		c = newClient()
	})

	Context("Get", func() {
		var projectName string

		BeforeEach(func(ctx SpecContext) {
			projectName = fmt.Sprintf("client_project_test_%d", time.Now().UnixNano())
			err := createProject(ctx, baseURL, bearerToken, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)
		}, NodeTimeout(30*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, baseURL, bearerToken, projectName)
			}
		}, NodeTimeout(15*time.Second))

		It("returns the project details", func(ctx SpecContext) {
			project, err := c.Projects().Get(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "Get should not fail")
			Expect(project).NotTo(BeNil())
			Expect(project.ProjectName).To(Equal(projectName))
		})

		It("returns not found for an unknown project", func(ctx SpecContext) {
			_, err := c.Projects().Get(ctx, "nonexistent_project_12345")
			Expect(err).To(HaveOccurred(), "getting unknown project should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for unknown project should be not-found, got: %v", err)
		})
	})

	Context("ListNames", func() {
		It("returns at least one project name", func(ctx SpecContext) {
			names, err := c.Projects().ListNames(ctx)
			Expect(err).NotTo(HaveOccurred(), "ListNames should not fail")
			Expect(names).NotTo(BeEmpty(), "should have at least one project name")
		})
	})
})
