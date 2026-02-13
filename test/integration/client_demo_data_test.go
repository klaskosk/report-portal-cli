//go:build integration

package integration_test

import (
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Demo Data", Label("client", "demo-data"), func() {
	var projectName string

	BeforeEach(func(ctx SpecContext) {
		projectName = fmt.Sprintf("client_demo_data_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, baseURL, bearerToken, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("Generate", func() {
		It("generates demo data and returns resource IDs", func(ctx SpecContext) {
			c := newClient()
			shouldCreateDashboard := true

			result, err := c.Project(projectName).DemoData().Generate(ctx, types.DemoDataRq{
				CreateDashboard: &shouldCreateDashboard,
			})
			Expect(err).NotTo(HaveOccurred(),
				"generating demo data should not fail")
			Expect(result).NotTo(BeNil(),
				"demo data result should not be nil")
			Expect(result.LaunchIds).NotTo(BeNil(),
				"demo data should contain launch IDs")
			Expect(*result.LaunchIds).NotTo(BeEmpty(),
				"demo data should have created at least one launch")
		}, NodeTimeout(60*time.Second))
	})
})
