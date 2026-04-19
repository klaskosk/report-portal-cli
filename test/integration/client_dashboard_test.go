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

var _ = Describe("Client: Dashboards", Label("client", "dashboards"), func() {
	var c *client.Client
	var projectName string

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_dashboard_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, c, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("CRUD lifecycle", func() {
		It("creates, gets, updates, and deletes a dashboard", func(ctx SpecContext) {
			dashboards := c.Project(projectName).Dashboards()

			By("creating a dashboard")
			description := "integration test dashboard"
			created, err := dashboards.Create(ctx, types.CreateDashboardRQ{
				Name:        "Test Dashboard",
				Description: &description,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())
			dashboardID := *created.Id

			By("getting the dashboard by ID")
			dashboard, err := dashboards.GetByID(ctx, dashboardID)
			Expect(err).NotTo(HaveOccurred())
			Expect(dashboard).NotTo(BeNil())
			Expect(dashboard.Id).To(Equal(dashboardID))
			Expect(dashboard.Name).To(Equal("Test Dashboard"))
			Expect(dashboard.Description).NotTo(BeNil())
			Expect(*dashboard.Description).To(Equal("integration test dashboard"))

			By("updating the dashboard")
			newDescription := "updated description"
			_, err = dashboards.Update(ctx, dashboardID, types.UpdateDashboardRQ{
				Name:        "Updated Dashboard",
				Description: &newDescription,
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the update")
			updated, err := dashboards.GetByID(ctx, dashboardID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("Updated Dashboard"))
			Expect(updated.Description).NotTo(BeNil())
			Expect(*updated.Description).To(Equal("updated description"))

			By("deleting the dashboard")
			_, err = dashboards.Delete(ctx, dashboardID)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the dashboard is gone")
			_, err = dashboards.GetByID(ctx, dashboardID)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("List", func() {
		It("lists dashboards including the created one", func(ctx SpecContext) {
			dashboards := c.Project(projectName).Dashboards()

			created, err := dashboards.Create(ctx, types.CreateDashboardRQ{
				Name: "Listed Dashboard",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Id).NotTo(BeNil())

			result, err := dashboards.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeNil())

			found := false
			for _, d := range *result.Content {
				if d.Id == *created.Id {
					found = true
					Expect(d.Name).To(Equal("Listed Dashboard"))

					break
				}
			}
			Expect(found).To(BeTrue(), "created dashboard should appear in list")
		})
	})

	Context("Widget management", func() {
		It("adds and removes a widget from a dashboard", func(ctx SpecContext) {
			dashboards := c.Project(projectName).Dashboards()
			filters := c.Project(projectName).Filters()
			widgets := c.Project(projectName).Widgets()

			By("creating a filter for the widget")
			filterCreated, err := filters.Create(ctx, types.UpdateUserFilterRQ{
				Name:       fmt.Sprintf("widget-filter-%d", time.Now().UnixNano()),
				Type:       types.UpdateUserFilterRQTypeLaunch,
				Conditions: []types.UserFilterCondition{
					{
						FilteringField: "name",
						Condition:      "cnt",
						Value:          "dashboard",
					},
				},
				Orders: []types.Order{
					{SortingColumn: "startTime", IsAsc: false},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(filterCreated.Id).NotTo(BeNil())
			filterID := *filterCreated.Id

			By("creating a widget")
			widgetOpts := map[string]interface{}{}
			contentFields := []string{"statistics$executions$total", "statistics$executions$passed", "statistics$executions$failed"}
			itemsCount := int32(10)
			filterIDs := []int64{filterID}
			widgetCreated, err := widgets.Create(ctx, types.WidgetRQ{
				Name:       fmt.Sprintf("dashboard-widget-%d", time.Now().UnixNano()),
				WidgetType: types.WidgetRQWidgetTypeLaunchStatistics,
				FilterIds:  &filterIDs,
				ContentParameters: &types.ContentParameters{
					ContentFields: &contentFields,
					ItemsCount:    &itemsCount,
					WidgetOptions: &widgetOpts,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetCreated.Id).NotTo(BeNil())
			widgetID := *widgetCreated.Id

			By("creating a dashboard")
			dashCreated, err := dashboards.Create(ctx, types.CreateDashboardRQ{
				Name: "Widget Dashboard",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(dashCreated.Id).NotTo(BeNil())
			dashboardID := *dashCreated.Id

			By("adding the widget to the dashboard")
			widgetName := fmt.Sprintf("dashboard-widget-%d", time.Now().UnixNano())
			_, err = dashboards.AddWidget(ctx, dashboardID, types.AddWidgetRq{
				AddWidget: types.WidgetObjectModel{
					WidgetId:   widgetID,
					WidgetName: &widgetName,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the widget appears on the dashboard")
			dash, err := dashboards.GetByID(ctx, dashboardID)
			Expect(err).NotTo(HaveOccurred())
			Expect(dash.Widgets).NotTo(BeNil())
			Expect(*dash.Widgets).NotTo(BeEmpty())

			found := false
			for _, w := range *dash.Widgets {
				if w.WidgetId == widgetID {
					found = true

					break
				}
			}
			Expect(found).To(BeTrue(), "widget should appear on the dashboard")

			By("removing the widget from the dashboard")
			_, err = dashboards.RemoveWidget(ctx, dashboardID, widgetID)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the widget is removed")
			dashAfter, err := dashboards.GetByID(ctx, dashboardID)
			Expect(err).NotTo(HaveOccurred())
			if dashAfter.Widgets != nil {
				for _, w := range *dashAfter.Widgets {
					Expect(w.WidgetId).NotTo(Equal(widgetID),
						"widget should no longer appear on the dashboard")
				}
			}
		})
	})
})
