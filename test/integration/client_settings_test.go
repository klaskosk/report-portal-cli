//go:build integration

package integration_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Settings", Label("client", "settings"), func() {
	It("returns server settings without error", func(ctx SpecContext) {
		c := newClient()

		settings, err := c.Settings().GetServerSettings(ctx)
		Expect(err).NotTo(HaveOccurred(),
			"getting server settings should not fail")
		Expect(settings).NotTo(BeNil(),
			"server settings result should not be nil")
	}, NodeTimeout(30*time.Second))
})
