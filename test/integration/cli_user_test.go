//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// cliUserResource models the JSON contract used by the user CLI commands.
type cliUserResource struct {
	ID          int64   `json:"id"`
	UserID      string  `json:"userId"`
	Email       string  `json:"email"`
	FullName    *string `json:"fullName"`
	UserRole    *string `json:"userRole"`
	Active      *bool   `json:"active"`
	AccountType *string `json:"accountType"`
}

var _ = Describe("CLI: user", Label("cli", "user"), func() {
	var c *client.Client

	BeforeEach(func() {
		c = newClient()
	})

	It("lists users in table format", func() {
		result := runCLI("user", "list")
		expectCLISuccess(result, "rp user list")
		Expect(result.Stdout).To(ContainSubstring("LOGIN"))
		Expect(result.Stdout).To(ContainSubstring(testAdminUser))
	})

	It("lists users in JSON format", func() {
		result := runCLI("user", "list", "-o", "json")
		expectCLISuccess(result, "rp user list -o json")

		var users []cliUserResource
		Expect(json.Unmarshal([]byte(result.Stdout), &users)).To(Succeed(),
			"stdout should be a valid JSON array, got: %s", result.Stdout)
		Expect(users).NotTo(BeEmpty())
		Expect(users[0].UserID).NotTo(BeEmpty())
		Expect(users[0].Email).NotTo(BeEmpty())
	})

	It("gets a specific user in table format", func() {
		result := runCLI("user", "get", testAdminUser)
		expectCLISuccess(result, "rp user get "+testAdminUser)
		Expect(result.Stdout).To(ContainSubstring(testAdminUser))
		Expect(result.Stdout).To(ContainSubstring("FULL NAME"))
	})

	It("gets a specific user in JSON format", func() {
		result := runCLI("user", "get", testAdminUser, "-o", "json")
		expectCLISuccess(result, "rp user get <login> -o json")
		expectJSONObject(result.Stdout)

		var user cliUserResource
		Expect(json.Unmarshal([]byte(result.Stdout), &user)).To(Succeed())
		Expect(user.UserID).To(Equal(testAdminUser))
		Expect(user.Email).NotTo(BeEmpty())
	})

	It("creates a user and verifies it with get", func(ctx SpecContext) {
		username := uniqueCLIUsername("create")
		email := fmt.Sprintf("%s@example.com", username)

		result := runCLI(
			"user", "create",
			"--login", username,
			"--email", email,
			"--full-name", "CLI Create User",
			"--password", "Test_P@ssw0rd!",
			"--default-project", "superadmin_personal",
			"--account-role", "USER",
			"--project-role", "MEMBER",
		)
		expectCLISuccess(result, "rp user create")
		Expect(result.Stdout).To(ContainSubstring(username))

		createdUser, err := c.Users().GetByLogin(ctx, username)
		Expect(err).NotTo(HaveOccurred(), "created user should be retrievable via API")
		Expect(createdUser.Email).To(Equal(email))
		Expect(createdUser.FullName).NotTo(BeNil())
		Expect(*createdUser.FullName).To(Equal("CLI Create User"))

		DeferCleanup(func(ctx SpecContext) {
			_, _ = c.Users().Delete(ctx, createdUser.Id)
		})

		getResult := runCLI("user", "get", username, "-o", "json")
		expectCLISuccess(getResult, "rp user get created-user -o json")

		var fetched cliUserResource
		Expect(json.Unmarshal([]byte(getResult.Stdout), &fetched)).To(Succeed())
		Expect(fetched.UserID).To(Equal(username))
		Expect(fetched.Email).To(Equal(email))
	})

	It("updates a user while preserving full name when omitted", func(ctx SpecContext) {
		user := createCLIUserFixture(ctx, c, uniqueCLIUsername("update"))
		DeferCleanup(func(ctx SpecContext) {
			_, _ = c.Users().Delete(ctx, user.Id)
		})

		updatedEmail := fmt.Sprintf("%s-updated@example.com", user.UserId)

		result := runCLI(
			"user", "update", user.UserId,
			"--email", updatedEmail,
			"--active=false",
		)
		expectCLISuccess(result, "rp user update <login>")
		Expect(result.Stdout).To(ContainSubstring("updated"))

		updatedUser, err := c.Users().GetByLogin(ctx, user.UserId)
		Expect(err).NotTo(HaveOccurred(), "updated user should still exist")
		Expect(updatedUser.Email).To(Equal(updatedEmail))
		Expect(updatedUser.FullName).NotTo(BeNil())
		Expect(*updatedUser.FullName).To(Equal("CLI Fixture User"))
		Expect(updatedUser.Active).NotTo(BeNil())
		Expect(*updatedUser.Active).To(BeFalse())
	})

	It("deletes a user by login", func(ctx SpecContext) {
		user := createCLIUserFixture(ctx, c, uniqueCLIUsername("delete"))
		DeferCleanup(func(ctx SpecContext) {
			_, _ = c.Users().Delete(ctx, user.Id)
		})

		result := runCLI("user", "delete", user.UserId)
		expectCLISuccess(result, "rp user delete <login>")
		Expect(result.Stdout).To(ContainSubstring("deleted"))

		_, err := c.Users().GetByLogin(ctx, user.UserId)
		Expect(err).To(HaveOccurred(), "deleted user should no longer exist")
		Expect(client.IsNotFound(err)).To(BeTrue(),
			"deleted user lookup should return not found, got: %v", err)
	})

	It("exits non-zero for a non-existent user", func() {
		result := runCLI("user", "get", "nonexistent_user_999999")
		expectCLIError(result, "rp user get nonexistent_user_999999")
	})

	It("prints errors as JSON when -o json is set", func() {
		result := runCLI("user", "get", "nonexistent_user_999999", "-o", "json")
		expectCLIError(result, "rp user get nonexistent_user_999999 -o json")
		expectJSONError(result, "not found")
		Expect(result.Stdout).To(BeEmpty(), "stdout should be empty when JSON error goes to stderr")
	})
})

// uniqueCLIUsername returns a unique username suitable for integration tests.
func uniqueCLIUsername(prefix string) string {
	return fmt.Sprintf("cli_%s_user_%d", prefix, time.Now().UnixNano())
}

// createCLIUserFixture creates a disposable user for CLI integration tests.
func createCLIUserFixture(ctx SpecContext, c *client.Client, username string) *types.UserResource {
	password := "Test_P@ssw0rd!"
	email := fmt.Sprintf("%s@example.com", username)
	accountRole := types.CreateUserRQFullAccountRole("USER")
	projectRole := types.CreateUserRQFullProjectRole("MEMBER")

	_, err := c.Users().Create(ctx, types.CreateUserRQFull{
		Login:          username,
		Password:       &password,
		Email:          email,
		FullName:       "CLI Fixture User",
		DefaultProject: strPtr("superadmin_personal"),
		AccountRole:    &accountRole,
		ProjectRole:    &projectRole,
	})
	Expect(err).NotTo(HaveOccurred(), "failed to create CLI user fixture %q", username)

	user, err := c.Users().GetByLogin(ctx, username)
	Expect(err).NotTo(HaveOccurred(), "failed to look up CLI user fixture %q", username)

	return user
}
