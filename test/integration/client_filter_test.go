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

var _ = Describe("Client: Filters", Label("client", "filters"), func() {
	var c *client.Client
	var projectName string

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_filter_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, baseURL, bearerToken, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("CRUD lifecycle", func() {
		It("creates, gets, updates, and deletes a filter", func(ctx SpecContext) {
			filters := c.Project(projectName).Filters()

			By("creating a filter")
			filterName := fmt.Sprintf("test-filter-%d", time.Now().UnixNano())
			created, err := filters.Create(ctx, types.UpdateUserFilterRQ{
				Name: filterName,
				Type: types.UpdateUserFilterRQTypeLaunch,
				Conditions: []types.UserFilterCondition{
					{
						FilteringField: "name",
						Condition:      "cnt",
						Value:          "test",
					},
				},
				Orders: []types.Order{
					{SortingColumn: "startTime", IsAsc: false},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())
			filterID := *created.Id

			By("getting the filter by ID")
			filter, err := filters.GetByID(ctx, filterID)
			Expect(err).NotTo(HaveOccurred())
			Expect(filter).NotTo(BeNil())
			Expect(filter.Id).To(Equal(filterID))
			Expect(filter.Name).To(Equal(filterName))
			Expect(filter.Conditions).To(HaveLen(1))

			By("updating the filter")
			updatedName := fmt.Sprintf("updated-filter-%d", time.Now().UnixNano())
			_, err = filters.Update(ctx, filterID, types.UpdateUserFilterRQ{
				Name: updatedName,
				Type: types.UpdateUserFilterRQTypeLaunch,
				Conditions: []types.UserFilterCondition{
					{
						FilteringField: "name",
						Condition:      "cnt",
						Value:          "updated",
					},
				},
				Orders: []types.Order{
					{SortingColumn: "startTime", IsAsc: false},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the update")
			updated, err := filters.GetByID(ctx, filterID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal(updatedName))

			By("deleting the filter")
			_, err = filters.Delete(ctx, filterID)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the filter is gone")
			_, err = filters.GetByID(ctx, filterID)
			Expect(err).To(HaveOccurred())
			Expect(client.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("GetByIds", func() {
		It("returns multiple filters by their IDs", func(ctx SpecContext) {
			filters := c.Project(projectName).Filters()

			By("creating two filters")
			filter1Name := fmt.Sprintf("ids-filter-1-%d", time.Now().UnixNano())
			created1, err := filters.Create(ctx, types.UpdateUserFilterRQ{
				Name: filter1Name,
				Type: types.UpdateUserFilterRQTypeLaunch,
				Conditions: []types.UserFilterCondition{
					{FilteringField: "name", Condition: "cnt", Value: "test1"},
				},
				Orders: []types.Order{
					{SortingColumn: "startTime", IsAsc: false},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created1.Id).NotTo(BeNil())

			filter2Name := fmt.Sprintf("ids-filter-2-%d", time.Now().UnixNano())
			created2, err := filters.Create(ctx, types.UpdateUserFilterRQ{
				Name: filter2Name,
				Type: types.UpdateUserFilterRQTypeLaunch,
				Conditions: []types.UserFilterCondition{
					{FilteringField: "name", Condition: "cnt", Value: "test2"},
				},
				Orders: []types.Order{
					{SortingColumn: "startTime", IsAsc: false},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created2.Id).NotTo(BeNil())

			By("getting both filters by IDs")
			result, err := filters.GetByIds(ctx, []int64{*created1.Id, *created2.Id})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))

			names := []string{result[0].Name, result[1].Name}
			Expect(names).To(ContainElement(filter1Name))
			Expect(names).To(ContainElement(filter2Name))
		})
	})

	Context("List", func() {
		It("lists filters including the created one", func(ctx SpecContext) {
			filters := c.Project(projectName).Filters()

			filterName := fmt.Sprintf("listed-filter-%d", time.Now().UnixNano())
			created, err := filters.Create(ctx, types.UpdateUserFilterRQ{
				Name:       filterName,
				Type:       types.UpdateUserFilterRQTypeLaunch,
				Conditions: []types.UserFilterCondition{
					{
						FilteringField: "name",
						Condition:      "cnt",
						Value:          "listed",
					},
				},
				Orders: []types.Order{
					{SortingColumn: "startTime", IsAsc: false},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Id).NotTo(BeNil())

			result, err := filters.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Content).NotTo(BeNil())

			found := false
			for _, f := range *result.Content {
				if f.Id == *created.Id {
					found = true
					Expect(f.Name).To(Equal(filterName))

					break
				}
			}
			Expect(found).To(BeTrue(), "created filter should appear in list")
		})
	})
})

