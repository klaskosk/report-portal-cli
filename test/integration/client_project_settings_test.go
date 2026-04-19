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

var _ = Describe("Client: ProjectSettings", Label("client", "project-settings"), func() {
	var c *client.Client

	BeforeEach(func() {
		c = newClient()
	})

	Context("Get", func() {
		It("returns settings with standard issue sub-types", func(ctx SpecContext) {
			settings, err := c.Project("superadmin_personal").Settings().Get(ctx)
			Expect(err).NotTo(HaveOccurred(), "Get should not fail")
			Expect(settings).NotTo(BeNil())
			Expect(settings.SubTypes).NotTo(BeEmpty(), "should have issue sub-types")

			expectedGroups := []string{
				"TO_INVESTIGATE",
				"PRODUCT_BUG",
				"AUTOMATION_BUG",
				"SYSTEM_ISSUE",
				"NO_DEFECT",
			}
			for _, group := range expectedGroups {
				Expect(settings.SubTypes).To(HaveKey(group),
					"should have %q issue sub-type group", group)
				Expect(settings.SubTypes[group]).NotTo(BeEmpty(),
					"%q group should have at least one sub-type", group)
			}
		})

		It("returns not found for a non-existent project", func(ctx SpecContext) {
			_, err := c.Project("nonexistent_project_12345").Settings().Get(ctx)
			Expect(err).To(HaveOccurred(), "Get should fail for unknown project")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error should be not-found, got: %v", err)
		})
	})

	Context("GetNotifications", func() {
		It("returns without error", func(ctx SpecContext) {
			notifications, err := c.Project("superadmin_personal").Settings().GetNotifications(ctx)
			Expect(err).NotTo(HaveOccurred(), "GetNotifications should not fail")
			Expect(notifications).NotTo(BeNil(), "notifications should not be nil")
		})
	})
})

var _ = Describe("Client: ProjectSettings Write", Label("client", "project-settings", "write"), func() {
	var (
		c           *client.Client
		projectName string
	)

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("settings_write_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, c, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("Notification Lifecycle", func() {
		It("creates, updates, and deletes a notification rule", func(ctx SpecContext) {
			settings := c.Project(projectName).Settings()

			// Create.
			ruleName := fmt.Sprintf("test-rule-%d", time.Now().UnixNano())
			enabled := true
			sendCase := types.Always
			attributesOperator := types.AND
			attributes := []types.ItemAttributeResource{}
			recipients := []string{"test@example.com"}

			notificationType := "email"
			created, err := settings.CreateNotification(ctx, types.SenderCaseDTO{
				Attributes:         &attributes,
				RuleName:           ruleName,
				Enabled:            &enabled,
				SendCase:           sendCase,
				AttributesOperator: attributesOperator,
				Recipients:         &recipients,
				Type:               &notificationType,
			})
			Expect(err).NotTo(HaveOccurred(), "creating notification should not fail")
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())

			createdID := *created.Id

			// Update.
			updatedName := ruleName + "-updated"
			notifications, err := settings.GetNotifications(ctx)
			Expect(err).NotTo(HaveOccurred(), "listing notifications should not fail")

			var updateRequest *types.SenderCaseDTO
			for i := range notifications {
				if notifications[i].Id != nil && *notifications[i].Id == createdID {
					updateRequest = &notifications[i]
					break
				}
			}
			Expect(updateRequest).NotTo(BeNil(), "created notification should be listed")

			updateRequest.RuleName = updatedName
			if updateRequest.Attributes == nil {
				updateRequest.Attributes = &attributes
			}
			if updateRequest.AttributesOperator == "" {
				updateRequest.AttributesOperator = attributesOperator
			}
			if updateRequest.Enabled == nil {
				updateRequest.Enabled = &enabled
			}
			if updateRequest.SendCase == "" {
				updateRequest.SendCase = sendCase
			}
			if updateRequest.Recipients == nil {
				updateRequest.Recipients = &recipients
			}

			_, err = settings.UpdateNotification(ctx, *updateRequest)
			Expect(err).NotTo(HaveOccurred(), "updating notification should not fail")

			// Delete.
			_, err = settings.DeleteNotification(ctx, createdID)
			Expect(err).NotTo(HaveOccurred(), "deleting notification should not fail")
		})
	})

	Context("Pattern Template Lifecycle", func() {
		It("creates, updates, and deletes a pattern template", func(ctx SpecContext) {
			settings := c.Project(projectName).Settings()

			// Create.
			name := fmt.Sprintf("test-pattern-%d", time.Now().UnixNano())
			patternType := "string"
			enabled := true
			value := "timeout"

			created, err := settings.CreatePatternTemplate(ctx, types.CreatePatternTemplateRQ{
				Name:    name,
				Type:    patternType,
				Enabled: enabled,
				Value:   value,
			})
			Expect(err).NotTo(HaveOccurred(), "creating pattern template should not fail")
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())

			createdID := *created.Id

			// Update.
			updatedName := name + "-updated"
			_, err = settings.UpdatePatternTemplate(ctx, createdID, types.UpdatePatternTemplateRQ{
				Name:    updatedName,
				Enabled: enabled,
			})
			Expect(err).NotTo(HaveOccurred(), "updating pattern template should not fail")

			// Delete.
			_, err = settings.DeletePatternTemplate(ctx, createdID)
			Expect(err).NotTo(HaveOccurred(), "deleting pattern template should not fail")
		})
	})

	Context("Issue Sub-Type Lifecycle", func() {
		It("creates, updates, and deletes an issue sub-type", func(ctx SpecContext) {
			settings := c.Project(projectName).Settings()

			// Create.
			created, err := settings.CreateIssueSubType(ctx, types.CreateIssueSubTypeRQ{
				LongName:  fmt.Sprintf("Custom Bug %d", time.Now().UnixNano()),
				ShortName: "CB",
				Color:     "#ff0000",
				TypeRef:   "PRODUCT_BUG",
			})
			Expect(err).NotTo(HaveOccurred(), "creating issue sub-type should not fail")
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())
			Expect(created.Locator).NotTo(BeNil())

			createdID := *created.Id
			locator := *created.Locator

			// Update.
			updatedName := "Updated Bug"
			updatedShortName := "UB"
			updatedColor := "#0000ff"
			_, err = settings.UpdateIssueSubType(ctx, types.UpdateOneIssueSubTypeRQ{
				Locator:   locator,
				TypeRef:   "PRODUCT_BUG",
				LongName:  updatedName,
				ShortName: updatedShortName,
				Color:     updatedColor,
			})
			Expect(err).NotTo(HaveOccurred(), "updating issue sub-type should not fail")

			// Delete.
			_, err = settings.DeleteIssueSubType(ctx, createdID)
			Expect(err).NotTo(HaveOccurred(), "deleting issue sub-type should not fail")
		})
	})
})
