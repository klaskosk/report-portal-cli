//go:build integration

package integration_test

import (
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
				// Skip("no global integrations available to get")
				Fail("no global integrations available to get")
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

			By("listing global integrations by plugin to find an available plugin type")
			allIntegrations, err := c.Integrations().ListGlobal(ctx)
			Expect(err).NotTo(HaveOccurred())

			if len(allIntegrations) == 0 {
				// Skip("no global integrations available to determine plugin type")
				Fail("no global integrations available to determine plugin type")
			}

			// Use the first integration's type to determine what plugins are available
			pluginType := ""
			if allIntegrations[0].IntegrationType != nil && allIntegrations[0].IntegrationType.Name != nil {
				pluginType = *allIntegrations[0].IntegrationType.Name
			}
			if pluginType == "" {
				// Skip("cannot determine plugin type from existing integrations")
				Fail("cannot determine plugin type from existing integrations")
			}

			By("creating a global integration for plugin: " + pluginType)
			name := "integration-test-global"
			enabled := true
			params := &map[string]interface{}{
				"url": "https://test.example.com",
			}
			created, err := c.Integrations().CreateGlobal(ctx, pluginType, types.IntegrationRQ{
				Name:                  &name,
				Enabled:               &enabled,
				IntegrationParameters: params,
			})
			if err != nil {
				// Some plugin types may not support creation; skip gracefully
				// Skip("plugin type " + pluginType + " does not support integration creation: " + err.Error())
				Fail("plugin type " + pluginType + " does not support integration creation: " + err.Error())
			}
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())
			integrationID := *created.Id

			By("updating the global integration")
			updatedName := "integration-test-global-updated"
			_, err = c.Integrations().UpdateGlobal(ctx, integrationID, types.IntegrationRQ{
				Name:                  &updatedName,
				Enabled:               &enabled,
				IntegrationParameters: params,
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
