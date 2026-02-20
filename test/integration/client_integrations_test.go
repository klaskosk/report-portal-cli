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

		It("gets a global integration by ID when one exists", func(ctx SpecContext) {
			c := newClient()

			integrations, err := c.Integrations().ListGlobal(ctx)
			Expect(err).NotTo(HaveOccurred())

			if len(integrations) == 0 {
				By("verifying empty global integrations are handled")
				Expect(integrations).To(BeEmpty())
				return
			}

			integration, err := c.Integrations().GetGlobal(ctx, *integrations[0].Id)
			Expect(err).NotTo(HaveOccurred())
			Expect(integration).NotTo(BeNil())
			Expect(integration.Id).NotTo(BeNil())
			Expect(*integration.Id).To(Equal(*integrations[0].Id))
		}, NodeTimeout(30*time.Second))

		It("returns not found for a non-existent global integration", func(ctx SpecContext) {
			c := newClient()

			_, err := c.Integrations().GetGlobal(ctx, 999999)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue())
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

		It("returns not found for a non-existent project integration", func(ctx SpecContext) {
			c := newClient()

			_, err := c.Project("superadmin_personal").Integrations().Get(ctx, 999999)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue())
		}, NodeTimeout(30*time.Second))
	})

	Context("Global Integration CRUD", func() {
		It("creates, updates, and deletes a global integration", func(ctx SpecContext) {
			c := newClient()

			By("determining a plugin type to use")
			pluginType := "email"
			allIntegrations, err := c.Integrations().ListGlobal(ctx)
			Expect(err).NotTo(HaveOccurred())
			if len(allIntegrations) > 0 &&
				allIntegrations[0].IntegrationType != nil &&
				allIntegrations[0].IntegrationType.Name != nil &&
				*allIntegrations[0].IntegrationType.Name != "" {
				pluginType = *allIntegrations[0].IntegrationType.Name
			}

			By("creating a global integration for plugin: " + pluginType)
			name := fmt.Sprintf("integration-test-global-%d", time.Now().UnixNano())
			enabled := true
			created, err := c.Integrations().CreateGlobal(ctx, pluginType, types.IntegrationRQ{
				Name:    &name,
				Enabled: &enabled,
			})
			if err != nil {
				By("verifying unsupported plugin configurations fail predictably")
				Expect(
					client.IsBadRequest(err) || client.IsForbidden(err) || client.IsNotFound(err),
				).To(BeTrue(), "unexpected create global integration error: %v", err)
				return
			}
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())
			integrationID := *created.Id
			DeferCleanup(func(ctx SpecContext) {
				_, _ = c.Integrations().DeleteGlobal(ctx, integrationID)
			})

			By("updating the global integration")
			updatedName := name + "-updated"
			_, err = c.Integrations().UpdateGlobal(ctx, integrationID, types.IntegrationRQ{
				Name:    &updatedName,
				Enabled: &enabled,
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the update")
			updated, err := c.Integrations().GetGlobal(ctx, integrationID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).NotTo(BeNil())
			Expect(*updated.Name).To(Equal(updatedName))

			By("deleting the global integration")
			_, err = c.Integrations().DeleteGlobal(ctx, integrationID)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the integration is gone")
			_, err = c.Integrations().GetGlobal(ctx, integrationID)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue())
		}, NodeTimeout(60*time.Second))
	})
})
