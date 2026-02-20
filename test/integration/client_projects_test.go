//go:build integration

package integration_test

import (
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
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

	Context("List", func() {
		It("returns at least one project info entry", func(ctx SpecContext) {
			result, err := c.Projects().List(ctx)
			Expect(err).NotTo(HaveOccurred(), "List should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeEmpty(), "should have at least one project")
		})

		It("supports pagination options", func(ctx SpecContext) {
			result, err := c.Projects().List(ctx,
				client.WithProjectPageNumber(0),
				client.WithProjectPageSize(1),
			)
			Expect(err).NotTo(HaveOccurred(), "List with pagination should not fail")
			Expect(result).NotTo(BeNil())
			Expect(len(result.Content)).To(BeNumerically("<=", 1),
				"page size 1 should return at most 1 project")
		})
	})

	Context("SearchNames", func() {
		It("returns names matching a known prefix", func(ctx SpecContext) {
			names, err := c.Projects().SearchNames(ctx, "superadmin")
			Expect(err).NotTo(HaveOccurred(), "SearchNames should not fail")
			Expect(names).NotTo(BeEmpty(),
				"searching for 'superadmin' should find at least the default project")
		})

		It("returns empty for a non-matching term", func(ctx SpecContext) {
			names, err := c.Projects().SearchNames(ctx, "zzz_nonexistent_project_xyz")
			Expect(err).NotTo(HaveOccurred(), "SearchNames should not fail")
			Expect(names).To(BeEmpty(),
				"searching for a non-matching term should return no results")
		})
	})

	Context("GetInfo", func() {
		It("returns info for a known project", func(ctx SpecContext) {
			info, err := c.Projects().GetInfo(ctx, "superadmin_personal")
			Expect(err).NotTo(HaveOccurred(), "GetInfo should not fail")
			Expect(info).NotTo(BeNil())
			Expect(info.ProjectName).NotTo(BeNil())
			Expect(*info.ProjectName).To(Equal("superadmin_personal"))
		})

		It("returns not found for an unknown project", func(ctx SpecContext) {
			_, err := c.Projects().GetInfo(ctx, "nonexistent_project_12345")
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error should be not-found, got: %v", err)
		})
	})
})

var _ = Describe("Client: Projects Write", Label("client", "projects", "write"), func() {
	var c *client.Client

	BeforeEach(func() {
		c = newClient()
	})

	Context("Create and Delete lifecycle", func() {
		It("creates and deletes a project", func(ctx SpecContext) {
			projectName := fmt.Sprintf("client_create_test_%d", time.Now().UnixNano())

			created, err := c.Projects().Create(ctx, types.CreateProjectRQ{
				ProjectName: projectName,
				EntryType:   types.INTERNAL,
			})
			Expect(err).NotTo(HaveOccurred(), "Create should not fail")
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())

			project, err := c.Projects().Get(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "Get should not fail after creation")
			Expect(project).NotTo(BeNil())
			Expect(project.ProjectName).To(Equal(projectName))

			_, err = c.Projects().Delete(ctx, project.ProjectId)
			Expect(err).NotTo(HaveOccurred(), "Delete should not fail")
		}, NodeTimeout(30*time.Second))
	})

	Context("Update", func() {
		var projectName string

		BeforeEach(func(ctx SpecContext) {
			projectName = fmt.Sprintf("client_update_test_%d", time.Now().UnixNano())
			err := createProject(ctx, baseURL, bearerToken, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)
		}, NodeTimeout(30*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, baseURL, bearerToken, projectName)
			}
		}, NodeTimeout(15*time.Second))

		It("updates project configuration", func(ctx SpecContext) {
			result, err := c.Projects().Update(ctx, projectName, types.UpdateProjectRQ{
				Configuration: &types.ProjectConfigurationUpdate{
					Attributes: map[string]string{
						"analyzer.isAutoAnalyzerEnabled": "false",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred(), "Update should not fail")
			Expect(result).NotTo(BeNil())
		})
	})

	Context("ListUsers", func() {
		It("returns users for a known project", func(ctx SpecContext) {
			result, err := c.Projects().ListUsers(ctx, "superadmin_personal")
			Expect(err).NotTo(HaveOccurred(), "ListUsers should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeEmpty(),
				"superadmin_personal should have at least one user")
		})
	})
})
