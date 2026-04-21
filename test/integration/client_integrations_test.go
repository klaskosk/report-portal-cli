//go:build integration

package integration_test

import (
	"fmt"
	"strings"
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

			By("finding a plugin that accepts a minimal create request")
			plugins, err := c.Plugins().List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(plugins).NotTo(BeEmpty(), "at least one plugin must be installed")

			var integrationID int64
			var tried []string
			name := fmt.Sprintf("integration-test-global-%d", time.Now().UnixNano())
			enabled := true

			for _, plugin := range plugins {
				if plugin.Name == nil || *plugin.Name == "" {
					continue
				}

				created, createErr := c.Integrations().CreateGlobal(ctx, *plugin.Name, types.IntegrationRQ{
					Name:    &name,
					Enabled: &enabled,
				})
				if createErr != nil {
					tried = append(tried, fmt.Sprintf("%s: %v", *plugin.Name, createErr))
					continue
				}

				Expect(created).NotTo(BeNil())
				Expect(created.Id).NotTo(BeNil())
				integrationID = *created.Id
				By("using plugin: " + *plugin.Name)

				break
			}

			Expect(integrationID).NotTo(BeZero(),
				"no plugin accepted a minimal create request; tried: %s",
				strings.Join(tried, "; "))

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

	Context("Project Integration CRUD", func() {
		var (
			c           *client.Client
			projectName string
		)

		BeforeEach(func(ctx SpecContext) {
			c = newClient()
			projectName = fmt.Sprintf("integ_crud_test_%d", time.Now().UnixNano())

			By("creating isolated project " + projectName)
			err := createProject(ctx, c, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)
		}, NodeTimeout(30*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, c, projectName)
			}
		}, NodeTimeout(15*time.Second))

		It("creates, gets, updates, and deletes a project integration", func(ctx SpecContext) {
			By("finding a plugin that accepts a minimal create request")
			plugins, err := c.Plugins().List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(plugins).NotTo(BeEmpty(), "at least one plugin must be installed")

			var integrationID int64
			var tried []string
			name := fmt.Sprintf("integ-test-proj-%d", time.Now().UnixNano())
			enabled := true

			for _, plugin := range plugins {
				if plugin.Name == nil || *plugin.Name == "" {
					continue
				}

				created, createErr := c.Project(projectName).Integrations().Create(ctx, *plugin.Name, types.IntegrationRQ{
					Name:    &name,
					Enabled: &enabled,
				})
				if createErr != nil {
					tried = append(tried, fmt.Sprintf("%s: %v", *plugin.Name, createErr))
					continue
				}

				Expect(created).NotTo(BeNil())
				Expect(created.Id).NotTo(BeNil())
				integrationID = *created.Id
				By("using plugin: " + *plugin.Name)

				break
			}

			Expect(integrationID).NotTo(BeZero(),
				"no plugin accepted a minimal create request; tried: %s",
				strings.Join(tried, "; "))

			By("getting the integration by ID")
			fetched, err := c.Project(projectName).Integrations().Get(ctx, integrationID)
			Expect(err).NotTo(HaveOccurred())
			Expect(fetched).NotTo(BeNil())
			Expect(fetched.Id).NotTo(BeNil())
			Expect(*fetched.Id).To(Equal(integrationID))
			Expect(fetched.Name).NotTo(BeNil())
			Expect(*fetched.Name).To(Equal(name))

			By("updating the integration")
			updatedName := name + "-updated"
			_, err = c.Project(projectName).Integrations().Update(ctx, integrationID, types.IntegrationRQ{
				Name:    &updatedName,
				Enabled: &enabled,
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the update")
			updated, err := c.Project(projectName).Integrations().Get(ctx, integrationID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).NotTo(BeNil())
			Expect(*updated.Name).To(Equal(updatedName))

			By("deleting the integration")
			_, err = c.Project(projectName).Integrations().Delete(ctx, integrationID)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the integration is gone")
			_, err = c.Project(projectName).Integrations().Get(ctx, integrationID)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue())
		}, NodeTimeout(60*time.Second))
	})

	Context("Connection Testing", func() {
		It("returns an error when testing connection for a non-existent integration", func(ctx SpecContext) {
			c := newClient()

			_, err := c.Project("superadmin_personal").Integrations().TestConnection(ctx, 999999)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"non-existent integration should produce IsNotFound, got: %v", err)
		}, NodeTimeout(30*time.Second))

		It("returns an error when testing global connection for a non-existent integration", func(ctx SpecContext) {
			c := newClient()

			_, err := c.Integrations().TestGlobalConnection(ctx, 999999)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"non-existent global integration should produce IsNotFound, got: %v", err)
		}, NodeTimeout(30*time.Second))
	})

	Context("Delete All By Type", func() {
		var (
			c           *client.Client
			projectName string
		)

		BeforeEach(func(ctx SpecContext) {
			c = newClient()
			projectName = fmt.Sprintf("integ_delall_test_%d", time.Now().UnixNano())

			By("creating isolated project " + projectName)
			err := createProject(ctx, c, projectName)
			Expect(err).To(Succeed(), "failed to create test project %q", projectName)
		}, NodeTimeout(30*time.Second))

		AfterEach(func(ctx SpecContext) {
			if projectName != "" {
				deleteProject(ctx, c, projectName)
			}
		}, NodeTimeout(15*time.Second))

		It("deletes all project integrations of a given type", func(ctx SpecContext) {
			By("finding a plugin that accepts minimal create requests")
			plugins, err := c.Plugins().List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(plugins).NotTo(BeEmpty(), "at least one plugin must be installed")

			var pluginType string
			var tried []string
			enabled := true

			for _, plugin := range plugins {
				if plugin.Name == nil || *plugin.Name == "" {
					continue
				}

				name := fmt.Sprintf("integ-delall-probe-%d", time.Now().UnixNano())
				created, createErr := c.Project(projectName).Integrations().Create(ctx, *plugin.Name, types.IntegrationRQ{
					Name:    &name,
					Enabled: &enabled,
				})
				if createErr != nil {
					tried = append(tried, fmt.Sprintf("%s: %v", *plugin.Name, createErr))
					continue
				}

				Expect(created).NotTo(BeNil())
				Expect(created.Id).NotTo(BeNil())
				_, _ = c.Project(projectName).Integrations().Delete(ctx, *created.Id)

				pluginType = *plugin.Name
				By("using plugin: " + pluginType)

				break
			}

			Expect(pluginType).NotTo(BeEmpty(),
				"no plugin accepted a minimal create request; tried: %s",
				strings.Join(tried, "; "))

			By("creating two project integrations of the same type")
			for i := range 2 {
				name := fmt.Sprintf("integ-delall-%d-%d", i, time.Now().UnixNano())
				_, err := c.Project(projectName).Integrations().Create(ctx, pluginType, types.IntegrationRQ{
					Name:    &name,
					Enabled: &enabled,
				})
				Expect(err).NotTo(HaveOccurred(),
					"creating integration %d of type %s", i, pluginType)
			}

			By("verifying integrations were created")
			before, err := c.Project(projectName).Integrations().ListByPlugin(ctx, pluginType)
			Expect(err).NotTo(HaveOccurred())
			Expect(before).To(HaveLen(2))

			By("calling DeleteAllByType")
			_, err = c.Project(projectName).Integrations().DeleteAllByType(ctx, pluginType)
			Expect(err).NotTo(HaveOccurred())

			By("verifying none remain")
			after, err := c.Project(projectName).Integrations().ListByPlugin(ctx, pluginType)
			Expect(err).NotTo(HaveOccurred())
			Expect(after).To(BeEmpty())
		}, NodeTimeout(60*time.Second))
	})

	Context("Execute Command", func() {
		It("returns an error for a non-existent project integration command", func(ctx SpecContext) {
			c := newClient()

			_, err := c.Project("superadmin_personal").Integrations().ExecuteCommand(
				ctx,
				999999,
				"test-command",
				map[string]any{},
			)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err) || client.IsBadRequest(err)).To(BeTrue(),
				"non-existent integration command should produce not found or bad request, got: %v", err)
		}, NodeTimeout(30*time.Second))
	})
})
