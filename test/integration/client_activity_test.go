//go:build integration

package integration_test

import (
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Activities", Label("client", "activity"), func() {
	var c *client.Client
	var projectName string

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_activity_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, baseURL, bearerToken, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("ListForItem", func() {
		It("returns an activity response for a seeded test item", func(ctx SpecContext) {
			By("seeding test items with a launch and hierarchy")
			seeded, err := seedTestItems(ctx, baseURL, bearerToken, projectName)
			Expect(err).NotTo(HaveOccurred(), "seeding test items should succeed")

			By("seeding item logs to generate activity records")
			_, err = seedTestLogEntries(ctx, baseURL, bearerToken, projectName, seeded, 1)
			Expect(err).NotTo(HaveOccurred(), "seeding test logs should succeed")

			By("querying activities for the seeded test hierarchy")
			activities := c.Project(projectName).Activities()
			candidateItemIDs := []int64{seeded.SuiteID, seeded.TestID}

			activityQuerySucceeded := false
			foundItemID := int64(0)
			var lastListErr error
			waitForSeedResolution(ctx, activityResolutionAttempts, activityResolutionInterval, func() bool {
				for _, itemID := range candidateItemIDs {
					result, listErr := activities.ListForItem(
						ctx,
						itemID,
						client.WithActivityPageSize(50),
					)
					if listErr != nil {
						lastListErr = listErr
						continue
					}
					lastListErr = nil
					activityQuerySucceeded = true
					foundItemID = itemID
					_ = result
					return true
				}

				return false
			})

			Expect(activityQuerySucceeded).To(BeTrue(),
				"activity query for suite or test item should eventually succeed (suiteID=%d testID=%d foundItemID=%d lastError=%v)",
				seeded.SuiteID,
				seeded.TestID,
				foundItemID,
				lastListErr,
			)
		}, NodeTimeout(120*time.Second))
	})
})
