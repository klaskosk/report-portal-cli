//go:build integration

package integration_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CLI: whoami", Label("cli", "whoami"), func() {
	It("shows user info in table format", func() {
		result := runCLI("whoami")
		expectCLISuccess(result, "rp whoami")
		Expect(result.Stdout).To(ContainSubstring(testAdminUser))
		Expect(result.Stdout).To(ContainSubstring("Login"))
		Expect(result.Stdout).To(ContainSubstring("Email"))
	})

	It("shows user info in JSON format", func() {
		result := runCLI("whoami", "-o", "json")
		expectCLISuccess(result, "rp whoami -o json")

		var user map[string]any
		Expect(json.Unmarshal([]byte(result.Stdout), &user)).To(Succeed(),
			"stdout should be valid JSON, got: %s", result.Stdout)
		Expect(user).To(HaveKey("userId"))
		Expect(user["userId"]).To(Equal(testAdminUser))
		Expect(user).To(HaveKey("email"))
	})

	It("errors with invalid credentials", func() {
		result := runCLIWithArgs("--url", baseURL, "--token", "invalid-token", "whoami")
		expectCLIError(result, "rp whoami with invalid token")
	})

	It("prints error as JSON when -o json is set and auth fails", func() {
		result := runCLIWithArgs("--url", baseURL, "--token", "invalid-token", "-o", "json", "whoami")
		expectCLIError(result, "rp whoami -o json with invalid token")
		expectJSONError(result, "")
	})
})
