//go:build integration

package integration_test

import (
	"fmt"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client: Launches", Label("client", "launches"), func() {
	var c *client.Client
	var projectName string

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("client_launch_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding test launches into " + projectName)
		err = seedTestLaunches(ctx, baseURL, bearerToken, projectName, 3)
		Expect(err).To(Succeed(), "failed to seed test launches into %q", projectName)
	}, NodeTimeout(60*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName != "" {
			deleteProject(ctx, baseURL, bearerToken, projectName)
		}
	}, NodeTimeout(15*time.Second))

	Context("List", func() {
		It("returns launches for the project", func(ctx SpecContext) {
			result, err := c.Project(projectName).Launches().List(ctx)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches for project %q should not fail", projectName)
			Expect(result).NotTo(BeNil(),
				"launch list result should not be nil")
			Expect(result.Content).NotTo(BeEmpty(),
				"launch list should contain seeded demo launches")
		})

		It("filters by name", func(ctx SpecContext) {
			// First, list all to discover a known name.
			all, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches to discover a known name should not fail")
			Expect(all.Content).NotTo(BeEmpty(),
				"launch list should contain at least one launch for name discovery")

			knownName := all.Content[0].Name

			filtered, err := c.Project(projectName).Launches().List(ctx,
				client.WithName(knownName),
			)
			Expect(err).NotTo(HaveOccurred(),
				"filtering launches by name %q should not fail", knownName)

			for _, launch := range filtered.Content {
				Expect(launch.Name).To(Equal(knownName),
					"filtered launch should have name %q but got %q", knownName, launch.Name)
			}
		})

		It("paginates results", func(ctx SpecContext) {
			page1, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
				client.WithPageNumber(1),
				client.WithSort("id,asc"),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches page 1 should not fail")
			Expect(page1.Content).To(HaveLen(1),
				"page 1 with page size 1 should return exactly 1 launch")

			page2, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
				client.WithPageNumber(2),
				client.WithSort("id,asc"),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches page 2 should not fail")

			if page2.Page.TotalPages > 1 {
				Expect(page2.Content).To(HaveLen(1),
					"page 2 with page size 1 should return exactly 1 launch")
				Expect(page2.Content[0].Id).NotTo(Equal(page1.Content[0].Id),
					"page 2 launch should differ from page 1 launch")
			}
		})

		It("sorts by start time descending", func(ctx SpecContext) {
			result, err := c.Project(projectName).Launches().List(ctx,
				client.WithSort("startTime,desc"),
				client.WithPageSize(10),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches sorted by startTime desc should not fail")

			if len(result.Content) >= 2 {
				for i := 1; i < len(result.Content); i++ {
					prev := time.Time(result.Content[i-1].StartTime)
					curr := time.Time(result.Content[i].StartTime)
					Expect(prev).To(
						BeTemporally(">=", curr),
						"launch[%d] startTime %v should be >= launch[%d] startTime %v (descending order)",
						i-1, prev, i, curr,
					)
				}
			}
		})
	})

	Context("GetByID", func() {
		var knownID int64

		BeforeEach(func(ctx SpecContext) {
			list, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches to discover a known ID should not fail")
			Expect(list.Content).NotTo(BeEmpty(),
				"launch list should contain at least one launch for ID lookup")

			knownID = list.Content[0].Id
		})

		It("returns the launch", func(ctx SpecContext) {
			launch, err := c.Project(projectName).Launches().GetByID(
				ctx,
				strconv.FormatInt(knownID, 10),
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting launch by ID %d should not fail", knownID)
			Expect(launch).NotTo(BeNil(),
				"launch returned for ID %d should not be nil", knownID)
			Expect(launch.Id).To(Equal(knownID),
				"returned launch ID should match requested ID %d", knownID)
		})

		It("returns not found for an invalid ID", func(ctx SpecContext) {
			_, err := c.Project(projectName).Launches().GetByID(
				ctx,
				"999999",
			)
			Expect(err).To(HaveOccurred(),
				"getting launch by non-existent ID 999999 should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for non-existent launch ID should be a not-found error, got: %v", err)
		})
	})

	Context("GetByUUID", func() {
		var knownUUID string

		BeforeEach(func(ctx SpecContext) {
			list, err := c.Project(projectName).Launches().List(ctx,
				client.WithPageSize(1),
			)
			Expect(err).NotTo(HaveOccurred(),
				"listing launches to discover a known UUID should not fail")
			Expect(list.Content).NotTo(BeEmpty(),
				"launch list should contain at least one launch for UUID lookup")

			knownUUID = list.Content[0].Uuid
		})

		It("returns the launch", func(ctx SpecContext) {
			launch, err := c.Project(projectName).Launches().GetByUUID(
				ctx,
				knownUUID,
			)
			Expect(err).NotTo(HaveOccurred(),
				"getting launch by UUID %q should not fail", knownUUID)
			Expect(launch).NotTo(BeNil(),
				"launch returned for UUID %q should not be nil", knownUUID)
			Expect(launch.Uuid).To(Equal(knownUUID),
				"returned launch UUID should match requested UUID %q", knownUUID)
		})

		It("returns not found for an invalid UUID", func(ctx SpecContext) {
			_, err := c.Project(projectName).Launches().GetByUUID(
				ctx,
				"00000000-0000-0000-0000-000000000000",
			)
			Expect(err).To(HaveOccurred(),
				"getting launch by non-existent UUID should fail")
			Expect(client.IsNotFound(err)).To(BeTrue(),
				"error for non-existent launch UUID should be a not-found error, got: %v", err)
		})
	})
})
