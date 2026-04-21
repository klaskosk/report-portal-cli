//go:build integration

package integration_test

import (
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
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

	Context("UpdateState", func() {
		It("toggles a plugin's enabled state and restores it", func(ctx SpecContext) {
			c := newClient()

			By("listing plugins to find a known plugin")
			plugins, err := c.Plugins().List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(plugins).NotTo(BeEmpty(), "at least one plugin must be installed")

			plugin := plugins[0]
			Expect(plugin.Name).NotTo(BeNil())
			Expect(plugin.Type).NotTo(BeNil())

			originalEnabled := true
			if plugin.Enabled != nil {
				originalEnabled = *plugin.Enabled
			}

			DeferCleanup(func(ctx SpecContext) {
				_, _ = c.Plugins().UpdateState(ctx, *plugin.Type, types.UpdatePluginStateRQ{
					Enabled: &originalEnabled,
				})
			})

			By("updating the plugin state to the opposite")
			toggled := !originalEnabled
			result, err := c.Plugins().UpdateState(ctx, *plugin.Type, types.UpdatePluginStateRQ{
				Enabled: &toggled,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			By("restoring the original state")
			result, err = c.Plugins().UpdateState(ctx, *plugin.Type, types.UpdatePluginStateRQ{
				Enabled: &originalEnabled,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
		}, NodeTimeout(30*time.Second))
	})

	Context("Plugin Commands", func() {
		It("returns an error when executing a public command on a non-existent plugin", func(ctx SpecContext) {
			c := newClient()

			_, err := c.Plugins().ExecutePublicCommand(ctx, "nonexistent-plugin", "test-command", nil)
			Expect(err).To(HaveOccurred())
			Expect(
				client.IsNotFound(err) || client.IsBadRequest(err) || client.IsForbidden(err),
			).To(BeTrue(),
				"non-existent plugin should produce not found, bad request, or forbidden, got: %v", err)
		}, NodeTimeout(30*time.Second))

		It("returns an error when executing a command on a non-existent plugin", func(ctx SpecContext) {
			c := newClient()

			_, err := c.Plugins().ExecuteCommand(
				ctx,
				"superadmin_personal",
				"nonexistent-plugin",
				"test-command",
				nil,
			)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err) || client.IsBadRequest(err)).To(BeTrue(),
				"non-existent plugin should produce not found or bad request, got: %v", err)
		}, NodeTimeout(30*time.Second))
	})

	Context("GetPublicFile", func() {
		It("returns an error for a non-existent plugin file", func(ctx SpecContext) {
			c := newClient()

			_, err := c.Plugins().GetPublicFile(ctx, "nonexistent-plugin", "nonexistent-file.png")
			Expect(err).To(HaveOccurred())
		}, NodeTimeout(30*time.Second))
	})
})
