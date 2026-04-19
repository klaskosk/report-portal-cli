//go:build integration

package integration_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

var _ = Describe("Client: Project Groups", Label("client", "project-groups"), func() {
	var c *client.Client
	var projectName string
	var groupID int64
	var groupName string

	BeforeEach(func(ctx SpecContext) {
		c = newClient()

		projectName = fmt.Sprintf("proj_groups_test_%d", time.Now().UnixNano())
		groupName = fmt.Sprintf("proj_group_%d", time.Now().UnixNano())

		By("creating a test project: " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("creating a test group: " + groupName)
		created, err := c.Groups().Create(ctx, types.CreateGroupRequest{Name: groupName})
		Expect(err).NotTo(HaveOccurred(), "failed to create test group")
		Expect(created).NotTo(BeNil())
		Expect(created.Id).NotTo(BeNil())
		groupID = *created.Id
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if groupID != 0 {
			err := c.Groups().Delete(ctx, groupID)
			if err != nil {
				GinkgoWriter.Printf("warning: failed to delete test group %d: %v\n", groupID, err)
			}
		}

		if projectName != "" {
			deleteProject(ctx, c, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("CRUD Lifecycle", func() {
		It("adds a group to a project and retrieves it", func(ctx SpecContext) {
			permission := types.AddProjectToGroupByIdRequestPermissionMEMBER

			By("adding the group to the project")
			_, err := c.Project(projectName).Groups().AddOrUpdate(ctx, groupID, types.AddProjectToGroupByIdRequest{
				Permission: &permission,
			})
			Expect(err).NotTo(HaveOccurred(), "AddOrUpdate should not fail")

			By("retrieving the group from the project")
			projectGroup, err := c.Project(projectName).Groups().Get(ctx, groupID)
			Expect(err).NotTo(HaveOccurred(), "Get should not fail")
			Expect(projectGroup).NotTo(BeNil())
			Expect(projectGroup.Id).NotTo(BeNil())
			Expect(*projectGroup.Id).To(Equal(groupID))
			Expect(projectGroup.Permission).NotTo(BeNil())
			Expect(string(*projectGroup.Permission)).To(Equal(string(types.ProjectGroupInfoPermissionMEMBER)))
		})

		It("updates a group association on a project", func(ctx SpecContext) {
			memberPermission := types.AddProjectToGroupByIdRequestPermissionMEMBER
			managerPermission := types.AddProjectToGroupByIdRequestPermissionPROJECTMANAGER

			By("adding the group with MEMBER permission")
			_, err := c.Project(projectName).Groups().AddOrUpdate(ctx, groupID, types.AddProjectToGroupByIdRequest{
				Permission: &memberPermission,
			})
			Expect(err).NotTo(HaveOccurred(), "initial AddOrUpdate should not fail")

			By("updating the group to PROJECT_MANAGER permission")
			_, err = c.Project(projectName).Groups().AddOrUpdate(ctx, groupID, types.AddProjectToGroupByIdRequest{
				Permission: &managerPermission,
			})
			Expect(err).NotTo(HaveOccurred(), "update AddOrUpdate should not fail")

			By("verifying the permission was updated")
			projectGroup, err := c.Project(projectName).Groups().Get(ctx, groupID)
			Expect(err).NotTo(HaveOccurred(), "Get should not fail after update")
			Expect(projectGroup).NotTo(BeNil())
			Expect(projectGroup.Permission).NotTo(BeNil())
			Expect(string(*projectGroup.Permission)).To(Equal(string(types.ProjectGroupInfoPermissionPROJECTMANAGER)))
		})

		It("deletes a group association from a project", func(ctx SpecContext) {
			permission := types.AddProjectToGroupByIdRequestPermissionMEMBER

			By("adding the group to the project")
			_, err := c.Project(projectName).Groups().AddOrUpdate(ctx, groupID, types.AddProjectToGroupByIdRequest{
				Permission: &permission,
			})
			Expect(err).NotTo(HaveOccurred(), "AddOrUpdate should not fail")

			By("deleting the group from the project")
			err = c.Project(projectName).Groups().Delete(ctx, groupID)
			Expect(err).NotTo(HaveOccurred(), "Delete should not fail")

			By("verifying the group association is gone")
			_, err = c.Project(projectName).Groups().Get(ctx, groupID)
			Expect(err).To(HaveOccurred(), "Get should fail after deletion")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error should be not-found, got: %v", err)
		})
	})

	Context("List", func() {
		It("returns groups associated with a project", func(ctx SpecContext) {
			permission := types.AddProjectToGroupByIdRequestPermissionMEMBER

			By("adding the group to the project")
			_, err := c.Project(projectName).Groups().AddOrUpdate(ctx, groupID, types.AddProjectToGroupByIdRequest{
				Permission: &permission,
			})
			Expect(err).NotTo(HaveOccurred(), "AddOrUpdate should not fail")

			By("listing groups for the project")
			result, err := c.Project(projectName).Groups().List(ctx)
			Expect(err).NotTo(HaveOccurred(), "List should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Items).NotTo(BeNil())
			Expect(*result.Items).NotTo(BeEmpty(), "should have at least one group")

			found := false
			for _, g := range *result.Items {
				if g.Id != nil && *g.Id == groupID {
					found = true

					break
				}
			}
			Expect(found).To(BeTrue(), "test group should appear in project group list")
		})
	})
})
