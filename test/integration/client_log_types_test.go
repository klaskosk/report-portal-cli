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

var _ = Describe("Client: Log Types", Label("client", "log-types"), func() {
	var c *client.Client
	var projectName string

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_log_types_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, c, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("CRUD Lifecycle", func() {
		It("creates, reads, updates, and deletes a log type", func(ctx SpecContext) {
			logTypes := c.Project(projectName).LogTypes()

			By("creating a log type")
			logTypeName := fmt.Sprintf("T%d", time.Now().UnixNano()%10000000)
			created, err := logTypes.Create(ctx, types.LogTypeRequest{
				Name:         logTypeName,
				Level:        45000,
				IsFilterable: false,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.Id).NotTo(BeNil())
			logTypeID := *created.Id

			By("listing log types")
			list, err := logTypes.List(ctx)
			Expect(err).NotTo(HaveOccurred())

			found := false
			for _, lt := range list {
				if lt.Id != nil && *lt.Id == logTypeID {
					found = true

					break
				}
			}
			Expect(found).To(BeTrue(), "created log type should appear in list")

			By("updating the log type")
			updatedName := fmt.Sprintf("U%d", time.Now().UnixNano()%10000000)
			_, err = logTypes.Update(ctx, logTypeID, types.LogTypeRequest{
				Name:         updatedName,
				Level:        45000,
				IsFilterable: false,
			})
			Expect(err).NotTo(HaveOccurred())

			By("deleting the log type")
			err = logTypes.Delete(ctx, logTypeID)
			Expect(err).NotTo(HaveOccurred())

			By("verifying deletion")
			list, err = logTypes.List(ctx)
			Expect(err).NotTo(HaveOccurred())

			for _, lt := range list {
				if lt.Id != nil {
					Expect(*lt.Id).NotTo(Equal(logTypeID), "deleted log type should not appear in list")
				}
			}
		})
	})

	Context("List", func() {
		It("returns a list for a new project", func(ctx SpecContext) {
			list, err := c.Project(projectName).LogTypes().List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(list).NotTo(BeNil())
		})
	})
})
