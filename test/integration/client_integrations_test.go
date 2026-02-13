//go:build integration

package integration_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Integrations", Label("client", "integrations"), func() {
	Context("Global Integrations", func() {
		It("lists global integrations without error", func(ctx SpecContext) {
			c := newClient()

			integrations, err := c.Integrations().ListGlobal(ctx)
			Expect(err).NotTo(HaveOccurred(),
				"listing global integrations should not fail")
			Expect(integrations).NotTo(BeNil(),
				"global integrations result should not be nil")
		}, NodeTimeout(30*time.Second))
	})

	Context("Project Integrations", func() {
		It("lists project integrations without error", func(ctx SpecContext) {
			c := newClient()

			integrations, err := c.Project("superadmin_personal").Integrations().List(ctx)
			Expect(err).NotTo(HaveOccurred(),
				"listing project integrations should not fail")
			Expect(integrations).NotTo(BeNil(),
				"project integrations result should not be nil")
		}, NodeTimeout(30*time.Second))
	})
})
