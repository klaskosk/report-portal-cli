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

var _ = Describe("Client: Widgets", Label("client", "widgets"), func() {
	var c *client.Client
	var projectName string
	var filterID int64

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_widget_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("creating a filter for widget tests")
		created, filterErr := c.Project(projectName).Filters().Create(ctx, types.UpdateUserFilterRQ{
			Name: fmt.Sprintf("widget-test-filter-%d", time.Now().UnixNano()),
			Type: types.UpdateUserFilterRQTypeLaunch,
			Conditions: []types.UserFilterCondition{
				{
					FilteringField: "name",
					Condition:      "cnt",
					Value:          "widget-test",
				},
			},
			Orders: []types.Order{
				{SortingColumn: "startTime", IsAsc: false},
			},
		})
		Expect(filterErr).NotTo(HaveOccurred())
		Expect(created.Id).NotTo(BeNil())
		filterID = *created.Id
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, c, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("Create and Get", func() {
		It("creates a widget and gets it by ID", func(ctx SpecContext) {
			widgets := c.Project(projectName).Widgets()

			By("creating a widget")
			widgetOpts := map[string]interface{}{}
			contentFields := []string{"statistics$executions$total", "statistics$executions$passed"}
			itemsCount := int32(10)
			filterIDs := []int64{filterID}
			widgetName := fmt.Sprintf("test-widget-%d", time.Now().UnixNano())
			created, err := widgets.Create(ctx, types.WidgetRQ{
				Name:       widgetName,
				WidgetType: types.WidgetRQWidgetTypeStatisticTrend,
				FilterIds:  &filterIDs,
				ContentParameters: &types.ContentParameters{
					ContentFields: &contentFields,
					ItemsCount:    &itemsCount,
					WidgetOptions: &widgetOpts,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())
			widgetID := *created.Id

			By("getting the widget by ID")
			widget, err := widgets.GetByID(ctx, widgetID)
			Expect(err).NotTo(HaveOccurred())
			Expect(widget).NotTo(BeNil())
			Expect(widget.Id).To(Equal(widgetID))
			Expect(widget.Name).To(Equal(widgetName))
		})
	})

	Context("Update", func() {
		It("updates a widget name", func(ctx SpecContext) {
			widgets := c.Project(projectName).Widgets()

			By("creating a widget")
			widgetOpts := map[string]interface{}{}
			contentFields := []string{"statistics$executions$total"}
			itemsCount := int32(10)
			filterIDs := []int64{filterID}
			created, err := widgets.Create(ctx, types.WidgetRQ{
				Name:       fmt.Sprintf("update-widget-%d", time.Now().UnixNano()),
				WidgetType: types.WidgetRQWidgetTypeLaunchStatistics,
				FilterIds:  &filterIDs,
				ContentParameters: &types.ContentParameters{
					ContentFields: &contentFields,
					ItemsCount:    &itemsCount,
					WidgetOptions: &widgetOpts,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Id).NotTo(BeNil())
			widgetID := *created.Id

			By("updating the widget")
			updatedName := fmt.Sprintf("updated-widget-%d", time.Now().UnixNano())
			updateFields := []string{"statistics$executions$total"}
			updateItemsCount := int32(10)
			updateFilterIDs := []int64{filterID}
			_, err = widgets.Update(ctx, widgetID, types.WidgetRQ{
				Name:       updatedName,
				WidgetType: types.WidgetRQWidgetTypeLaunchStatistics,
				FilterIds:  &updateFilterIDs,
				ContentParameters: &types.ContentParameters{
					ContentFields: &updateFields,
					ItemsCount:    &updateItemsCount,
					WidgetOptions: &widgetOpts,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the update")
			widget, err := widgets.GetByID(ctx, widgetID)
			Expect(err).NotTo(HaveOccurred())
			Expect(widget.Name).To(Equal(updatedName))
		})
	})

	Context("GetData", func() {
		It("retrieves widget data by ID", func(ctx SpecContext) {
			widgets := c.Project(projectName).Widgets()

			By("creating a widget")
			widgetOpts := map[string]interface{}{}
			contentFields := []string{"statistics$executions$total", "statistics$executions$passed"}
			itemsCount := int32(10)
			filterIDs := []int64{filterID}
			widgetName := fmt.Sprintf("data-widget-%d", time.Now().UnixNano())
			created, err := widgets.Create(ctx, types.WidgetRQ{
				Name:       widgetName,
				WidgetType: types.WidgetRQWidgetTypeCumulative,
				FilterIds:  &filterIDs,
				ContentParameters: &types.ContentParameters{
					ContentFields: &contentFields,
					ItemsCount:    &itemsCount,
					WidgetOptions: &widgetOpts,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())
			widgetID := *created.Id

			By("getting widget data")
			widget, err := widgets.GetData(ctx, widgetID)
			Expect(err).NotTo(HaveOccurred())
			Expect(widget).NotTo(BeNil())
			Expect(widget.Id).To(Equal(widgetID))
			Expect(widget.Name).To(Equal(widgetName))
		})
	})

	Context("GetNames", func() {
		It("returns paged string content via the client wrapper", func(ctx SpecContext) {
			widgets := c.Project(projectName).Widgets()

			By("creating a widget")
			widgetOpts := map[string]interface{}{}
			contentFields := []string{"statistics$executions$total"}
			itemsCount := int32(10)
			filterIDs := []int64{filterID}
			widgetName := fmt.Sprintf("raw-names-widget-%d", time.Now().UnixNano())
			_, err := widgets.Create(ctx, types.WidgetRQ{
				Name:       widgetName,
				WidgetType: types.WidgetRQWidgetTypeLaunchStatistics,
				FilterIds:  &filterIDs,
				ContentParameters: &types.ContentParameters{
					ContentFields: &contentFields,
					ItemsCount:    &itemsCount,
					WidgetOptions: &widgetOpts,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("getting widget names via the client wrapper")
			result, err := widgets.GetNames(ctx, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeEmpty())
		})

		It("returns widget names including the created widget", func(ctx SpecContext) {
			widgets := c.Project(projectName).Widgets()

			By("creating a widget")
			widgetOpts := map[string]interface{}{}
			contentFields := []string{"statistics$executions$total"}
			itemsCount := int32(10)
			filterIDs := []int64{filterID}
			widgetName := fmt.Sprintf("names-widget-%d", time.Now().UnixNano())
			_, err := widgets.Create(ctx, types.WidgetRQ{
				Name:       widgetName,
				WidgetType: types.WidgetRQWidgetTypeLaunchStatistics,
				FilterIds:  &filterIDs,
				ContentParameters: &types.ContentParameters{
					ContentFields: &contentFields,
					ItemsCount:    &itemsCount,
					WidgetOptions: &widgetOpts,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("getting widget names")
			result, err := widgets.GetNames(ctx, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeEmpty())
		})
	})
})
