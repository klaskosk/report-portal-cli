//go:build integration

package integration_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Users", Label("client", "users"), func() {
	var c *client.Client

	BeforeEach(func() {
		c = newClient()
	})

	Context("GetMyself", func() {
		It("returns the authenticated user", func(ctx SpecContext) {
			user, err := c.Users().GetMyself(ctx)
			Expect(err).NotTo(HaveOccurred(), "GetMyself should not fail")
			Expect(user).NotTo(BeNil())
			Expect(user.UserId).To(Equal(testAdminUser))
			Expect(user.Email).NotTo(BeEmpty())
		})
	})

	Context("GetByLogin", func() {
		It("returns the admin user", func(ctx SpecContext) {
			user, err := c.Users().GetByLogin(ctx, testAdminUser)
			Expect(err).NotTo(HaveOccurred(), "GetByLogin should not fail")
			Expect(user).NotTo(BeNil())
			Expect(user.UserId).To(Equal(testAdminUser))
		})

		It("returns not found for an unknown login", func(ctx SpecContext) {
			_, err := c.Users().GetByLogin(ctx, "nonexistent_user_12345")
			Expect(err).To(HaveOccurred(), "getting unknown user should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for unknown user should be not-found, got: %v", err)
		})
	})

	Context("ListProjects", func() {
		It("returns at least one project for the admin user", func(ctx SpecContext) {
			projects, err := c.Users().ListProjects(ctx, testAdminUser)
			Expect(err).NotTo(HaveOccurred(), "ListProjects should not fail")
			Expect(projects).NotTo(BeEmpty(),
				"admin user should have at least one project")
		})
	})

	Context("List", func() {
		It("returns at least one user", func(ctx SpecContext) {
			page, err := c.Users().List(ctx)
			Expect(err).NotTo(HaveOccurred(), "List should not fail")
			Expect(page).NotTo(BeNil())
			Expect(page.Content).NotTo(BeNil())
			Expect(*page.Content).NotTo(BeEmpty())
		})

		It("supports pagination options", func(ctx SpecContext) {
			page, err := c.Users().List(ctx, client.WithUsersPageSize(1))
			Expect(err).NotTo(HaveOccurred(), "List with page size should not fail")
			Expect(page).NotTo(BeNil())
			Expect(page.Page).NotTo(BeNil())
			Expect(*page.Page.Size).To(Equal(int64(1)))
		})

		It("filters by role", func(ctx SpecContext) {
			page, err := c.Users().List(ctx, client.WithUsersRole("ADMINISTRATOR"))
			Expect(err).NotTo(HaveOccurred(), "List with role filter should not fail")
			Expect(page).NotTo(BeNil())
			Expect(page.Content).NotTo(BeNil())

			for _, u := range *page.Content {
				Expect(u.UserRole).NotTo(BeNil())
				Expect(*u.UserRole).To(Equal("ADMINISTRATOR"))
			}
		})
	})

	Context("Search", func() {
		It("finds users matching a term", func(ctx SpecContext) {
			page, err := c.Users().Search(ctx, "admin")
			Expect(err).NotTo(HaveOccurred(), "Search should not fail")
			Expect(page).NotTo(BeNil())
			Expect(page.Content).NotTo(BeNil())

			logins := make([]string, 0, len(*page.Content))
			for _, u := range *page.Content {
				logins = append(logins, u.UserId)
			}

			Expect(logins).To(ContainElement(testAdminUser))
		})

		It("returns empty for a non-matching term", func(ctx SpecContext) {
			page, err := c.Users().Search(ctx, "nonexistent_user_xyz_99999")
			Expect(err).NotTo(HaveOccurred(), "Search should not fail even with no results")
			Expect(page).NotTo(BeNil())

			if page.Content != nil {
				Expect(*page.Content).To(BeEmpty())
			}
		})
	})

	Context("Export", func() {
		It("returns CSV data", func(ctx SpecContext) {
			data, err := c.Users().Export(ctx)
			Expect(err).NotTo(HaveOccurred(), "Export should not fail")
			Expect(data).NotTo(BeEmpty(), "exported data should not be empty")

			firstLine := strings.SplitN(string(data), "\n", 2)[0]
			Expect(firstLine).To(ContainSubstring("Login"),
				"CSV header should contain column names")
		})
	})

	Context("User Lifecycle", func() {
		It("creates, edits, and deletes a user", func(ctx SpecContext) {
			username := fmt.Sprintf("test_user_%d", time.Now().UnixNano())
			password := "Test_P@ssw0rd!"
			email := fmt.Sprintf("%s@example.com", username)
			accountRole := types.CreateUserRQFullAccountRole("USER")
			projectRole := types.CreateUserRQFullProjectRole("MEMBER")

			By("creating a test user")
			created, err := c.Users().Create(ctx, types.CreateUserRQFull{
				Login:          username,
				Password:       &password,
				Email:          email,
				FullName:       "Integration Test User",
				DefaultProject: strPtr("superadmin_personal"),
				AccountRole:    &accountRole,
				ProjectRole:    &projectRole,
			})
			Expect(err).NotTo(HaveOccurred(), "Create should not fail")
			Expect(created).NotTo(BeNil())

			By("looking up the user to get their numeric ID")
			user, err := c.Users().GetByLogin(ctx, username)
			Expect(err).NotTo(HaveOccurred(), "GetByLogin for new user should not fail")
			Expect(user.UserId).To(Equal(username))
			userID := user.Id

			DeferCleanup(func(ctx SpecContext) {
				_, _ = c.Users().Delete(ctx, userID)
			})

			By("editing the user")
			editResult, err := c.Users().Edit(ctx, username, types.EditUserRQ{
				FullName: "Updated Test User",
			})
			Expect(err).NotTo(HaveOccurred(), "Edit should not fail")
			Expect(editResult).NotTo(BeNil())

			By("verifying the edit took effect")
			updated, err := c.Users().GetByLogin(ctx, username)
			Expect(err).NotTo(HaveOccurred(), "GetByLogin after edit should not fail")
			Expect(updated.FullName).NotTo(BeNil())
			Expect(*updated.FullName).To(Equal("Updated Test User"))

			By("deleting the user")
			deleteResult, err := c.Users().Delete(ctx, userID)
			Expect(err).NotTo(HaveOccurred(), "Delete should not fail")
			Expect(deleteResult).NotTo(BeNil())

			By("verifying the user is deleted")
			_, err = c.Users().GetByLogin(ctx, username)
			Expect(err).To(HaveOccurred(), "GetByLogin after delete should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for deleted user should be not-found, got: %v", err)
		})
	})

	Context("API Keys", func() {
		It("creates, lists, and deletes an API key", func(ctx SpecContext) {
			By("getting the current user's ID")
			myself, err := c.Users().GetMyself(ctx)
			Expect(err).NotTo(HaveOccurred(), "GetMyself should not fail")
			myID := myself.Id

			keyName := fmt.Sprintf("test-key-%d", time.Now().UnixNano())

			By("creating an API key")
			created, err := c.Users().CreateAPIKey(ctx, myID, types.ApiKeyRQ{
				Name: keyName,
			})
			Expect(err).NotTo(HaveOccurred(), "CreateAPIKey should not fail")
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeZero())
			keyID := created.Id

			DeferCleanup(func(ctx SpecContext) {
				_, _ = c.Users().DeleteAPIKey(ctx, myID, keyID)
			})

			By("listing API keys")
			keys, err := c.Users().ListAPIKeys(ctx, myID)
			Expect(err).NotTo(HaveOccurred(), "ListAPIKeys should not fail")

			found := false
			for _, k := range keys {
				if k.Id == keyID {
					found = true
					Expect(k.Name).To(Equal(keyName))

					break
				}
			}
			Expect(found).To(BeTrue(), "created key should appear in list")

			By("deleting the API key")
			deleteResult, err := c.Users().DeleteAPIKey(ctx, myID, keyID)
			Expect(err).NotTo(HaveOccurred(), "DeleteAPIKey should not fail")
			Expect(deleteResult).NotTo(BeNil())

			By("verifying the key is deleted")
			keysAfter, err := c.Users().ListAPIKeys(ctx, myID)
			Expect(err).NotTo(HaveOccurred(), "ListAPIKeys after delete should not fail")

			for _, k := range keysAfter {
				Expect(k.Id).NotTo(Equal(keyID),
					"deleted key should not appear in list")
			}
		})
	})
})

func strPtr(s string) *string {
	return &s
}
