//go:build integration

package integration_test

import (
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Error predicates", Label("client", "errors"), func() {
	Context("IsNotFound against live server", func() {
		It("matches for a non-existent launch ID", func(ctx SpecContext) {
			c := newClient()
			_, err := c.Project("default_personal").Launches().GetByID(ctx, 999999)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"non-existent launch should produce IsNotFound, got: %v", err)
		})

		It("matches for a non-existent launch UUID", func(ctx SpecContext) {
			c := newClient()
			_, err := c.Project("default_personal").Launches().GetByUUID(
				ctx, "00000000-0000-0000-0000-000000000000",
			)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"non-existent launch UUID should produce IsNotFound, got: %v", err)
		})

		It("matches for a non-existent user login", func(ctx SpecContext) {
			c := newClient()
			_, err := c.Users().GetByLogin(ctx, "nonexistent_user_xyz_999")
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"non-existent user should produce IsNotFound, got: %v", err)
		})
	})

	Context("IsUnauthorized against live server", func() {
		It("matches for an invalid bearer token", func(ctx SpecContext) {
			c, err := client.New(baseURL, "invalid-token-value")
			Expect(err).NotTo(HaveOccurred())

			_, err = c.Users().GetMyself(ctx)
			Expect(err).To(HaveOccurred())
			Expect(client.IsUnauthorized(err)).To(BeTrue(),
				"invalid token should produce IsUnauthorized, got: %v", err)
		})
	})

	Context("predicate exclusivity against live server", func() {
		It("IsNotFound does not match auth errors", func(ctx SpecContext) {
			c, err := client.New(baseURL, "invalid-token-value")
			Expect(err).NotTo(HaveOccurred())

			_, err = c.Users().GetMyself(ctx)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeFalse(),
				"auth error should not match IsNotFound")
			Expect(client.IsBadRequest(err)).To(BeFalse(),
				"auth error should not match IsBadRequest")
			Expect(client.IsConflict(err)).To(BeFalse(),
				"auth error should not match IsConflict")
		})

		It("IsUnauthorized does not match not-found errors", func(ctx SpecContext) {
			c := newClient()
			_, err := c.Project("default_personal").Launches().GetByID(ctx, 999999)
			Expect(err).To(HaveOccurred())
			Expect(client.IsUnauthorized(err)).To(BeFalse(),
				"not-found error should not match IsUnauthorized")
			Expect(client.IsForbidden(err)).To(BeFalse(),
				"not-found error should not match IsForbidden")
		})
	})

	Context("HasStatusCode against live server", func() {
		It("matches the correct status code for not-found", func(ctx SpecContext) {
			c := newClient()
			_, err := c.Project("default_personal").Launches().GetByID(ctx, 999999)
			Expect(err).To(HaveOccurred())
			Expect(client.HasStatusCode(err, 404)).To(BeTrue(),
				"not-found error should match HasStatusCode(404)")
			Expect(client.HasStatusCode(err, 401)).To(BeFalse(),
				"not-found error should not match HasStatusCode(401)")
		})

		It("matches the correct status code for unauthorized", func(ctx SpecContext) {
			c, err := client.New(baseURL, "invalid-token-value")
			Expect(err).NotTo(HaveOccurred())

			_, err = c.Users().GetMyself(ctx)
			Expect(err).To(HaveOccurred())
			Expect(client.HasStatusCode(err, 401)).To(BeTrue(),
				"auth error should match HasStatusCode(401)")
			Expect(client.HasStatusCode(err, 404)).To(BeFalse(),
				"auth error should not match HasStatusCode(404)")
		})
	})

	Context("error wrapping preservation", func() {
		It("predicates work through fmt.Errorf wrapping", func(ctx SpecContext) {
			c := newClient()
			_, originalErr := c.Project("default_personal").Launches().GetByID(ctx, 999999)
			Expect(originalErr).To(HaveOccurred())

			wrapped := fmt.Errorf("context: %w", originalErr)
			Expect(client.IsNotFound(wrapped)).To(BeTrue(),
				"IsNotFound should match through wrapping")
			Expect(client.HasStatusCode(wrapped, 404)).To(BeTrue(),
				"HasStatusCode should match through wrapping")
		})
	})

	Context("stable error response handling", func() {
		var projectName string

		BeforeEach(func(ctx SpecContext) {
			projectName = fmt.Sprintf("client_errors_test_%d", time.Now().UnixNano())
			err := createProject(ctx, baseURL, bearerToken, projectName)
			Expect(err).To(Succeed())
		}, NodeTimeout(30*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, baseURL, bearerToken, projectName)
			}
		}, NodeTimeout(15*time.Second))

		It("returns a non-nil APIError for not-found with accessible fields", func(ctx SpecContext) {
			c := newClient()
			_, err := c.Project(projectName).Launches().GetByID(ctx, 999999)
			Expect(err).To(HaveOccurred())

			var apiErr *client.APIError
			Expect(err).To(BeAssignableToTypeOf(apiErr),
				"not-found error should be assignable to *APIError")

			Expect(client.IsNotFound(err)).To(BeTrue())
			Expect(client.HasStatusCode(err, 404)).To(BeTrue())
		})

		It("returns a non-nil error for not-found item", func(ctx SpecContext) {
			c := newClient()
			_, err := c.Project(projectName).Items().GetByID(ctx, 999999)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"non-existent item should produce IsNotFound, got: %v", err)
		})
	})
})
