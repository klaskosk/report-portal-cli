//go:build integration

package integration_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Plugins", Label("client", "plugins"), func() {
	Context("List", func() {
		It("returns installed plugins without error", func(ctx SpecContext) {
			c := newClient()

			plugins, err := c.Plugins().List(ctx)
			Expect(err).NotTo(HaveOccurred(),
				"listing plugins should not fail")
			Expect(plugins).NotTo(BeNil(),
				"plugin list result should not be nil")
		}, NodeTimeout(30*time.Second))
	})

	Context("ListPublic", func() {
		It("returns public plugins without error", func(ctx SpecContext) {
			c := newClient()

			plugins, err := c.Plugins().ListPublic(ctx)
			Expect(err).NotTo(HaveOccurred(),
				"listing public plugins should not fail")
			Expect(plugins).NotTo(BeNil(),
				"public plugin list result should not be nil")
		}, NodeTimeout(30*time.Second))
	})
})
