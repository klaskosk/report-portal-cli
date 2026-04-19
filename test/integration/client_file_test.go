//go:build integration

package integration_test

import (
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: File Download", Label("client", "file"), func() {
	var (
		c           *client.Client
		projectName string
		seeded      seededTestItems
	)

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_file_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding one launch with suite and child test items")
		seeded, err = seedTestItems(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to seed test items in %q", projectName)
	}, NodeTimeout(240*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, c, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("DownloadFile", func() {
		It("downloads a known attachment", func(ctx SpecContext) {
			By("seeding a log entry with an attachment")
			attachID, err := seedLogWithAttachment(ctx, c, projectName, seeded)
			Expect(err).NotTo(HaveOccurred(), "seeding attachment should not fail")
			Expect(attachID).NotTo(BeZero(), "expected seeded attachment ID")

			content, err := c.Project(projectName).DownloadFile(ctx, attachID)
			Expect(err).NotTo(HaveOccurred(), "downloading file should not fail")
			Expect(content).NotTo(BeNil())
			Expect(content.Data).NotTo(BeEmpty(), "file content should not be empty")
		})

		It("returns not found for a non-existent data ID", func(ctx SpecContext) {
			_, err := c.Project(projectName).DownloadFile(ctx, 999999)
			Expect(err).To(HaveOccurred(), "downloading non-existent file should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for non-existent file should be not-found, got: %v", err)
		})
	})
})
