//go:build integration

package integration_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Onboarding", Label("client", "onboarding"), func() {
	Context("GetStatus", func() {
		It("returns onboarding status without a page filter", func(ctx SpecContext) {
			c := newClient()

			result, err := c.Onboarding().GetStatus(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
		}, NodeTimeout(60*time.Second))

		It("returns onboarding status with a page filter", func(ctx SpecContext) {
			c := newClient()

			page := "GENERAL"
			result, err := c.Onboarding().GetStatus(ctx, &page)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
		}, NodeTimeout(60*time.Second))
	})
})
