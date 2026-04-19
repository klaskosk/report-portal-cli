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

var _ = Describe("Client: Groups", Label("client", "groups"), func() {
	var c *client.Client
	var groupID int64
	var groupName string

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		groupName = fmt.Sprintf("test_group_%d", time.Now().UnixNano())

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
	}, NodeTimeout(15*time.Second))

	Context("CRUD Lifecycle", func() {
		It("creates a group and retrieves it by ID", func(ctx SpecContext) {
			group, err := c.Groups().Get(ctx, groupID)
			Expect(err).NotTo(HaveOccurred(), "Get should not fail")
			Expect(group).NotTo(BeNil())
			Expect(group.Id).NotTo(BeNil())
			Expect(*group.Id).To(Equal(groupID))
			Expect(group.Name).NotTo(BeNil())
			Expect(*group.Name).To(Equal(groupName))
		})

		It("updates a group name", func(ctx SpecContext) {
			updatedName := fmt.Sprintf("updated_group_%d", time.Now().UnixNano())

			By("updating the group")
			_, err := c.Groups().Update(ctx, groupID, types.UpdateGroupRequest{Name: updatedName})
			Expect(err).NotTo(HaveOccurred(), "Update should not fail")

			By("verifying the update")
			group, err := c.Groups().Get(ctx, groupID)
			Expect(err).NotTo(HaveOccurred(), "Get should not fail after update")
			Expect(group).NotTo(BeNil())
			Expect(group.Name).NotTo(BeNil())
			Expect(*group.Name).To(Equal(updatedName))
		})

		It("deletes a group", func(ctx SpecContext) {
			deleteName := fmt.Sprintf("delete_group_%d", time.Now().UnixNano())

			By("creating a group to delete")
			created, err := c.Groups().Create(ctx, types.CreateGroupRequest{Name: deleteName})
			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())
			deleteID := *created.Id

			By("deleting the group")
			err = c.Groups().Delete(ctx, deleteID)
			Expect(err).NotTo(HaveOccurred(), "Delete should not fail")

			By("verifying the group is gone")
			_, err = c.Groups().Get(ctx, deleteID)
			Expect(err).To(HaveOccurred(), "Get should fail after deletion")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error should be not-found, got: %v", err)
		})

		It("returns not found for a nonexistent group", func(ctx SpecContext) {
			_, err := c.Groups().Get(ctx, 999999999)
			Expect(err).To(HaveOccurred(), "Get should fail for nonexistent group")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error should be not-found, got: %v", err)
		})
	})

	Context("List", func() {
		It("returns at least one group", func(ctx SpecContext) {
			result, err := c.Groups().List(ctx)
			Expect(err).NotTo(HaveOccurred(), "List should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Items).NotTo(BeNil())
			Expect(*result.Items).NotTo(BeEmpty(), "should have at least one group")
		})

		It("supports pagination options", func(ctx SpecContext) {
			result, err := c.Groups().List(ctx,
				client.WithGroupPageSize(1),
				client.WithGroupOffset(0),
			)
			Expect(err).NotTo(HaveOccurred(), "List with pagination should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Items).NotTo(BeNil())
			Expect(len(*result.Items)).To(BeNumerically("<=", 1),
				"page size 1 should return at most 1 group")
		})
	})

	Context("User Management", func() {
		var userID int64

		BeforeEach(func(ctx SpecContext) {
			By("resolving the admin user's numeric ID")
			user, err := c.Users().GetMyself(ctx)
			Expect(err).NotTo(HaveOccurred(), "GetMyself should not fail")
			Expect(user).NotTo(BeNil())
			userID = user.Id
		}, NodeTimeout(15*time.Second))

		It("adds and retrieves a user in a group", func(ctx SpecContext) {
			By("adding the user to the group")
			_, err := c.Groups().AddUser(ctx, groupID, userID)
			Expect(err).NotTo(HaveOccurred(), "AddUser should not fail")

			By("retrieving the user from the group")
			groupUser, err := c.Groups().GetUser(ctx, groupID, userID)
			Expect(err).NotTo(HaveOccurred(), "GetUser should not fail")
			Expect(groupUser).NotTo(BeNil())
			Expect(groupUser.Id).NotTo(BeNil())
			Expect(*groupUser.Id).To(Equal(userID))
		})

		It("lists users in a group", func(ctx SpecContext) {
			By("adding the user to the group")
			_, err := c.Groups().AddUser(ctx, groupID, userID)
			Expect(err).NotTo(HaveOccurred(), "AddUser should not fail")

			By("listing users in the group")
			result, err := c.Groups().ListUsers(ctx, groupID)
			Expect(err).NotTo(HaveOccurred(), "ListUsers should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Items).NotTo(BeNil())
			Expect(*result.Items).NotTo(BeEmpty(), "should have at least one user")

			found := false
			for _, u := range *result.Items {
				if u.Id != nil && *u.Id == userID {
					found = true

					break
				}
			}
			Expect(found).To(BeTrue(), "admin user should appear in group user list")
		})

		It("removes a user from a group", func(ctx SpecContext) {
			By("adding the user to the group")
			_, err := c.Groups().AddUser(ctx, groupID, userID)
			Expect(err).NotTo(HaveOccurred(), "AddUser should not fail")

			By("removing the user from the group")
			err = c.Groups().RemoveUser(ctx, groupID, userID)
			Expect(err).NotTo(HaveOccurred(), "RemoveUser should not fail")

			By("verifying the user is no longer in the group")
			_, err = c.Groups().GetUser(ctx, groupID, userID)
			Expect(err).To(HaveOccurred(), "GetUser should fail after removal")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error should be not-found, got: %v", err)
		})
	})

	Context("Project Management", func() {
		var projectName string
		var projectID int64

		BeforeEach(func(ctx SpecContext) {
			projectName = fmt.Sprintf("group_proj_test_%d", time.Now().UnixNano())

			By("creating a test project: " + projectName)
			err := createProject(ctx, c, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)

			By("resolving the project's numeric ID")
			project, err := c.Projects().Get(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "failed to get project %q", projectName)
			projectID = project.ProjectId
		}, NodeTimeout(60*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, c, projectName)
			}
		}, NodeTimeout(15*time.Second))

		It("adds a project to a group", func(ctx SpecContext) {
			permission := types.AddProjectToGroupByIdRequestPermissionMEMBER
			_, err := c.Groups().AddProject(ctx, groupID, projectID, types.AddProjectToGroupByIdRequest{
				Permission: &permission,
			})
			Expect(err).NotTo(HaveOccurred(), "AddProject should not fail")
		})

		It("retrieves a project in a group", func(ctx SpecContext) {
			By("adding the project to the group")
			permission := types.AddProjectToGroupByIdRequestPermissionMEMBER
			_, err := c.Groups().AddProject(ctx, groupID, projectID, types.AddProjectToGroupByIdRequest{
				Permission: &permission,
			})
			Expect(err).NotTo(HaveOccurred(), "AddProject should not fail")

			By("retrieving the project from the group")
			groupProject, err := c.Groups().GetProject(ctx, groupID, projectID)
			Expect(err).NotTo(HaveOccurred(), "GetProject should not fail")
			Expect(groupProject).NotTo(BeNil())
			Expect(groupProject.Id).NotTo(BeNil())
			Expect(*groupProject.Id).To(Equal(projectID))
		})

		It("lists projects in a group", func(ctx SpecContext) {
			By("adding the project to the group")
			permission := types.AddProjectToGroupByIdRequestPermissionMEMBER
			_, err := c.Groups().AddProject(ctx, groupID, projectID, types.AddProjectToGroupByIdRequest{
				Permission: &permission,
			})
			Expect(err).NotTo(HaveOccurred(), "AddProject should not fail")

			By("listing projects in the group")
			result, err := c.Groups().ListProjects(ctx, groupID)
			Expect(err).NotTo(HaveOccurred(), "ListProjects should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Items).NotTo(BeNil())
			Expect(*result.Items).NotTo(BeEmpty(), "should have at least one project")

			found := false
			for _, p := range *result.Items {
				if p.Id != nil && *p.Id == projectID {
					found = true

					break
				}
			}
			Expect(found).To(BeTrue(), "test project should appear in group project list")
		})

		It("removes a project from a group", func(ctx SpecContext) {
			By("adding the project to the group")
			permission := types.AddProjectToGroupByIdRequestPermissionMEMBER
			_, err := c.Groups().AddProject(ctx, groupID, projectID, types.AddProjectToGroupByIdRequest{
				Permission: &permission,
			})
			Expect(err).NotTo(HaveOccurred(), "AddProject should not fail")

			By("removing the project from the group")
			err = c.Groups().RemoveProject(ctx, groupID, projectID)
			Expect(err).NotTo(HaveOccurred(), "RemoveProject should not fail")

			By("verifying the project is no longer in the group")
			_, err = c.Groups().GetProject(ctx, groupID, projectID)
			Expect(err).To(HaveOccurred(), "GetProject should fail after removal")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error should be not-found, got: %v", err)
		})
	})
})
