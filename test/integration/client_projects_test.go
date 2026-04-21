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
			err := createProject(ctx, c, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)
		}, NodeTimeout(30*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, c, projectName)
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
			Expect(result.Content).NotTo(BeNil(), "should have at least one project")
			Expect(*result.Content).NotTo(BeEmpty(), "should have at least one project")
		})

		It("supports pagination options", func(ctx SpecContext) {
			result, err := c.Projects().List(ctx,
				client.WithProjectPageNumber(0),
				client.WithProjectPageSize(1),
			)
			Expect(err).NotTo(HaveOccurred(), "List with pagination should not fail")
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeNil())
			Expect(len(*result.Content)).To(BeNumerically("<=", 1),
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
			Expect(info.ProjectName).To(Equal("superadmin_personal"))
		})

		It("returns not found for an unknown project", func(ctx SpecContext) {
			_, err := c.Projects().GetInfo(ctx, "nonexistent_project_12345")
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error should be not-found, got: %v", err)
		})
	})

	Context("Export", func() {
		It("returns data for projects", func(ctx SpecContext) {
			data, err := c.Projects().Export(ctx)
			Expect(err).NotTo(HaveOccurred(), "Export should not fail")
			Expect(data).NotTo(BeEmpty(), "exported data should not be empty")
		})
	})

	Context("Analyzer Status", func() {
		It("returns analyzer status", func(ctx SpecContext) {
			status, err := c.Projects().GetAnalyzerStatus(ctx)
			Expect(err).NotTo(HaveOccurred(), "GetAnalyzerStatus should not fail")
			Expect(status).NotTo(BeNil(), "analyzer status map should not be nil")
		})
	})

	Context("Assignable Users", func() {
		It("returns assignable users for a project", func(ctx SpecContext) {
			result, err := c.Projects().ListAssignableUsers(ctx, "superadmin_personal")
			Expect(err).NotTo(HaveOccurred(), "ListAssignableUsers should not fail")
			Expect(result).NotTo(BeNil())
		})
	})

	Context("Usernames", func() {
		It("lists usernames for a project", func(ctx SpecContext) {
			usernames, err := c.Projects().ListUsernames(ctx, "superadmin_personal", "super")
			Expect(err).NotTo(HaveOccurred(), "ListUsernames should not fail")
			Expect(usernames).NotTo(BeEmpty(),
				"superadmin_personal should have at least one username matching 'super'")
		})

		It("searches usernames in a project", func(ctx SpecContext) {
			result, err := c.Projects().SearchUsernames(ctx, "superadmin_personal", "super")
			Expect(err).NotTo(HaveOccurred(), "SearchUsernames should not fail")
			Expect(result).NotTo(BeNil())
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
				EntryType:   types.CreateProjectRQEntryTypeINTERNAL,
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
			err := createProject(ctx, c, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)
		}, NodeTimeout(30*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, c, projectName)
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
			Expect(result.Content).NotTo(BeNil(),
				"superadmin_personal should have at least one user")
			Expect(*result.Content).NotTo(BeEmpty(),
				"superadmin_personal should have at least one user")
		})
	})

	Context("User Assignment", func() {
		It("assigns and unassigns a user", func(ctx SpecContext) {
			By("creating a test user")

			password := "Testpassword1!"
			userLogin := fmt.Sprintf("projassign_%d", time.Now().UnixNano())
			accountRole := types.CreateUserRQFullAccountRoleUSER
			createdUser, err := c.Users().Create(ctx, types.CreateUserRQFull{
				Login:       userLogin,
				Email:       userLogin + "@example.com",
				FullName:    "Project Assignment Test User",
				Password:    &password,
				AccountRole: &accountRole,
			})
			Expect(err).NotTo(HaveOccurred(), "user creation should not fail")
			Expect(createdUser).NotTo(BeNil())
			Expect(createdUser.Id).NotTo(BeNil())

			DeferCleanup(func(ctx SpecContext) {
				_, delErr := c.Users().Delete(ctx, *createdUser.Id)
				if delErr != nil {
					GinkgoWriter.Printf("warning: could not delete test user %q: %v\n",
						userLogin, delErr)
				}
			}, NodeTimeout(15*time.Second))

			By("assigning the user to the project")

			assignResult, err := c.Projects().AssignUsers(ctx, "superadmin_personal",
				types.AssignUsersRQ{
					UserNames: map[string]string{
						userLogin: "MEMBER",
					},
				})
			Expect(err).NotTo(HaveOccurred(), "AssignUsers should not fail")
			Expect(assignResult).NotTo(BeNil())

			By("verifying the user appears in project users")

			usersResult, err := c.Projects().ListUsers(ctx, "superadmin_personal")
			Expect(err).NotTo(HaveOccurred(), "ListUsers should not fail")
			Expect(usersResult).NotTo(BeNil())

			users := expectContent(usersResult.Content, "project users")
			found := false
			for _, u := range users {
				if u.UserId == userLogin {
					found = true

					break
				}
			}
			Expect(found).To(BeTrue(), "assigned user %q should appear in project users", userLogin)

			By("unassigning the user")

			unassignResult, err := c.Projects().UnassignUsers(ctx, "superadmin_personal",
				types.UnassignUsersRQ{
					UserNames: []string{userLogin},
				})
			Expect(err).NotTo(HaveOccurred(), "UnassignUsers should not fail")
			Expect(unassignResult).NotTo(BeNil())

			By("verifying the user no longer appears in project users")

			usersAfter, err := c.Projects().ListUsers(ctx, "superadmin_personal")
			Expect(err).NotTo(HaveOccurred(), "ListUsers after unassign should not fail")

			afterUsers := expectContent(usersAfter.Content, "project users after unassign")
			for _, u := range afterUsers {
				Expect(u.UserId).NotTo(Equal(userLogin),
					"unassigned user %q should not appear in project users", userLogin)
			}
		}, NodeTimeout(60*time.Second))
	})

	Context("Preferences", func() {
		var projectName string

		BeforeEach(func(ctx SpecContext) {
			projectName = fmt.Sprintf("client_pref_test_%d", time.Now().UnixNano())
			err := createProject(ctx, c, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)
		}, NodeTimeout(30*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, c, projectName)
			}
		}, NodeTimeout(15*time.Second))

		It("adds, gets, and removes a project preference", func(ctx SpecContext) {
			By("creating a filter to use as a preference")

			filter, err := c.Project(projectName).Filters().Create(ctx, types.UpdateUserFilterRQ{
				Name: "preference-test-filter",
				Type: types.UpdateUserFilterRQTypeLaunch,
				Conditions: []types.UserFilterCondition{
					{
						Condition:      "cnt",
						FilteringField: "name",
						Value:          "test",
					},
				},
				Orders: []types.Order{
					{IsAsc: false, SortingColumn: "startTime"},
				},
			})
			Expect(err).NotTo(HaveOccurred(), "filter creation should not fail")
			Expect(filter).NotTo(BeNil())
			Expect(filter.Id).NotTo(BeNil())

			filterID := *filter.Id

			DeferCleanup(func(ctx SpecContext) {
				_, delErr := c.Project(projectName).Filters().Delete(ctx, filterID)
				if delErr != nil {
					GinkgoWriter.Printf("warning: could not delete test filter %d: %v\n",
						filterID, delErr)
				}
			}, NodeTimeout(15*time.Second))

			By("adding the filter as a preference")

			addResult, err := c.Projects().AddPreference(ctx, projectName, filterID)
			Expect(err).NotTo(HaveOccurred(), "AddPreference should not fail")
			Expect(addResult).NotTo(BeNil())

			By("verifying the filter appears in preferences")

			pref, err := c.Projects().GetPreference(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "GetPreference should not fail")
			Expect(pref).NotTo(BeNil())
			Expect(pref.Filters).NotTo(BeNil(), "preference filters should not be nil after add")

			filterIDs := make([]int64, len(*pref.Filters))
			for i, f := range *pref.Filters {
				filterIDs[i] = f.Id
			}
			Expect(filterIDs).To(ContainElement(filterID),
				"preference filters should contain the added filter %d", filterID)

			By("removing the preference")

			removeResult, err := c.Projects().RemovePreference(ctx, projectName, filterID)
			Expect(err).NotTo(HaveOccurred(), "RemovePreference should not fail")
			Expect(removeResult).NotTo(BeNil())

			By("verifying the filter is gone from preferences")

			prefAfter, err := c.Projects().GetPreference(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "GetPreference after remove should not fail")
			Expect(prefAfter).NotTo(BeNil())

			if prefAfter.Filters != nil {
				for _, f := range *prefAfter.Filters {
					Expect(f.Id).NotTo(Equal(filterID),
						"removed filter %d should not appear in preferences", filterID)
				}
			}
		}, NodeTimeout(60*time.Second))
	})

	Context("Notification Config", func() {
		var projectName string

		BeforeEach(func(ctx SpecContext) {
			projectName = fmt.Sprintf("client_notif_test_%d", time.Now().UnixNano())
			err := createProject(ctx, c, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)
		}, NodeTimeout(30*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, c, projectName)
			}
		}, NodeTimeout(15*time.Second))

		It("updates notification configuration", func(ctx SpecContext) {
			enabled := true
			caseEnabled := true
			recipients := []string{testAdminUser}
			launchNames := []string{"test-launch"}
			ruleType := "email"

			result, err := c.Projects().UpdateNotificationConfig(ctx, projectName,
				types.ProjectNotificationConfigDTO{
					Enabled: &enabled,
					Cases: &[]types.SenderCaseDTO{
						{
							RuleName:           "test-notification-rule",
							SendCase:           types.Always,
							Recipients:         &recipients,
							LaunchNames:        &launchNames,
							Enabled:            &caseEnabled,
							AttributesOperator: types.AND,
							Type:               &ruleType,
						},
					},
				})
			Expect(err).NotTo(HaveOccurred(), "UpdateNotificationConfig should not fail")
			Expect(result).NotTo(BeNil())
		}, NodeTimeout(30*time.Second))
	})

	Context("Index Management", func() {
		var projectName string

		BeforeEach(func(ctx SpecContext) {
			projectName = fmt.Sprintf("client_index_test_%d", time.Now().UnixNano())
			err := createProject(ctx, c, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)
		}, NodeTimeout(30*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, c, projectName)
			}
		}, NodeTimeout(15*time.Second))

		It("deletes project index", func(ctx SpecContext) {
			result, err := c.Projects().DeleteIndex(ctx, projectName)

			// CI environment may not have an analyzer deployed (409 Conflict, error code 40302).
			if err != nil {
				Expect(client.HasErrorCode(err, 40302)).To(BeTrue(),
					"unexpected error from DeleteIndex (expected error code 40302): %v", err)

				return
			}

			Expect(result).NotTo(BeNil())
		})
	})
})
