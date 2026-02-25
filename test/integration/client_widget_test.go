//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		err := createProject(ctx, baseURL, bearerToken, projectName)
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
			deleteProject(ctx, baseURL, bearerToken, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("Create and Get", func() {
		It("creates a widget and gets it by ID", func(ctx SpecContext) {
			widgets := c.Project(projectName).Widgets()

			By("creating a widget")
			widgetOpts := map[string]any{}
			widgetName := fmt.Sprintf("test-widget-%d", time.Now().UnixNano())
			created, err := widgets.Create(ctx, types.WidgetRQ{
				Name:       widgetName,
				WidgetType: "statisticTrend",
				FilterIds:  &[]int64{filterID},
				ContentParameters: &types.ContentParameters{
					ContentFields: []string{"statistics$executions$total", "statistics$executions$passed"},
					ItemsCount:    10,
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
			widgetOpts := map[string]any{}
			created, err := widgets.Create(ctx, types.WidgetRQ{
				Name:       fmt.Sprintf("update-widget-%d", time.Now().UnixNano()),
				WidgetType: "launchStatistics",
				FilterIds:  &[]int64{filterID},
				ContentParameters: &types.ContentParameters{
					ContentFields: []string{"statistics$executions$total"},
					ItemsCount:    10,
					WidgetOptions: &widgetOpts,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Id).NotTo(BeNil())
			widgetID := *created.Id

			By("updating the widget")
			updatedName := fmt.Sprintf("updated-widget-%d", time.Now().UnixNano())
			_, err = widgets.Update(ctx, widgetID, types.WidgetRQ{
				Name:       updatedName,
				WidgetType: "launchStatistics",
				FilterIds:  &[]int64{filterID},
				ContentParameters: &types.ContentParameters{
					ContentFields: []string{"statistics$executions$total"},
					ItemsCount:    10,
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
			widgetOpts := map[string]any{}
			widgetName := fmt.Sprintf("data-widget-%d", time.Now().UnixNano())
			created, err := widgets.Create(ctx, types.WidgetRQ{
				Name:       widgetName,
				WidgetType: "cumulative",
				FilterIds:  &[]int64{filterID},
				ContentParameters: &types.ContentParameters{
					ContentFields: []string{"statistics$executions$total", "statistics$executions$passed"},
					ItemsCount:    10,
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
		It("returns paged string content from the raw widget names endpoint", func(ctx SpecContext) {
			widgets := c.Project(projectName).Widgets()

			By("creating a widget")
			widgetOpts := map[string]any{}
			widgetName := fmt.Sprintf("raw-names-widget-%d", time.Now().UnixNano())
			_, err := widgets.Create(ctx, types.WidgetRQ{
				Name:       widgetName,
				WidgetType: "launchStatistics",
				FilterIds:  &[]int64{filterID},
				ContentParameters: &types.ContentParameters{
					ContentFields: []string{"statistics$executions$total"},
					ItemsCount:    10,
					WidgetOptions: &widgetOpts,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("calling the raw widget names endpoint")
			requestURL := fmt.Sprintf("%s/api/v1/%s/widget/names/all", baseURL, projectName)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Authorization", "Bearer "+bearerToken)

			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			var payload struct {
				Content []json.RawMessage          `json:"content"`
				Page    map[string]json.RawMessage `json:"page"`
			}
			err = json.Unmarshal(body, &payload)
			Expect(err).NotTo(HaveOccurred(), "response body: %s", string(body))

			Expect(payload.Content).NotTo(BeEmpty())
			Expect(payload.Page).NotTo(BeEmpty())

			for _, item := range payload.Content {
				var name string
				Expect(json.Unmarshal(item, &name)).To(
					Succeed(),
					"expected string content item, got: %s",
					string(item),
				)
			}
		})

		It("returns widget names including the created widget", func(ctx SpecContext) {
			widgets := c.Project(projectName).Widgets()

			By("creating a widget")
			widgetOpts := map[string]any{}
			widgetName := fmt.Sprintf("names-widget-%d", time.Now().UnixNano())
			_, err := widgets.Create(ctx, types.WidgetRQ{
				Name:       widgetName,
				WidgetType: "launchStatistics",
				FilterIds:  &[]int64{filterID},
				ContentParameters: &types.ContentParameters{
					ContentFields: []string{"statistics$executions$total"},
					ItemsCount:    10,
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
