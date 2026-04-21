//go:build integration

package integration_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// createTestPNG generates a minimal 1x1 PNG image for upload tests.
func createTestPNG() *bytes.Reader {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	var buf bytes.Buffer
	Expect(png.Encode(&buf, img)).To(Succeed())

	return bytes.NewReader(buf.Bytes())
}

var _ = Describe("Client: Files", Label("client", "files"), func() {
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

	Context("User Photo Lifecycle", func() {
		It("uploads, retrieves, and deletes a user photo", func(ctx SpecContext) {
			DeferCleanup(func(ctx SpecContext) {
				_, _ = c.Files().DeletePhoto(ctx)
			})

			By("uploading a photo")
			result, err := c.Files().UploadPhoto(ctx, "test.png", createTestPNG())
			Expect(err).NotTo(HaveOccurred(), "uploading photo should not fail")
			Expect(result).NotTo(BeNil())

			By("retrieving the photo")
			content, err := c.Files().GetMyPhoto(ctx)
			Expect(err).NotTo(HaveOccurred(), "retrieving photo should not fail")
			Expect(content).NotTo(BeNil())
			Expect(content.ContentType).NotTo(BeEmpty(), "content type should be set")
			Expect(content.Data).NotTo(BeEmpty(), "photo data should not be empty")

			By("retrieving the photo thumbnail")
			thumbnail, err := c.Files().GetMyPhoto(ctx, client.WithPhotoThumbnail(true))
			Expect(err).NotTo(HaveOccurred(), "retrieving photo thumbnail should not fail")
			Expect(thumbnail).NotTo(BeNil())

			By("deleting the photo")
			deleteResult, err := c.Files().DeletePhoto(ctx)
			Expect(err).NotTo(HaveOccurred(), "deleting photo should not fail")
			Expect(deleteResult).NotTo(BeNil())
		})
	})

	Context("GetUserPhoto", func() {
		It("retrieves a user photo by login after uploading", func(ctx SpecContext) {
			DeferCleanup(func(ctx SpecContext) {
				_, _ = c.Files().DeletePhoto(ctx)
			})

			By("uploading a photo for the current user")
			_, err := c.Files().UploadPhoto(ctx, "user-photo.png", createTestPNG())
			Expect(err).NotTo(HaveOccurred(), "uploading photo should not fail")

			By("retrieving the photo via GetUserPhoto")
			content, err := c.Project(projectName).GetUserPhoto(ctx, testAdminUser)
			Expect(err).NotTo(HaveOccurred(), "GetUserPhoto should not fail")
			Expect(content).NotTo(BeNil())
			Expect(content.Data).NotTo(BeEmpty(), "user photo data should not be empty")
		})

		It("returns a default photo for a user without a custom photo", func(ctx SpecContext) {
			content, err := c.Project(projectName).GetUserPhoto(ctx, testAdminUser)
			Expect(err).NotTo(HaveOccurred(),
				"GetUserPhoto should succeed even without a custom photo")
			Expect(content).NotTo(BeNil())
			Expect(content.Data).NotTo(BeEmpty(),
				"default photo data should not be empty")
		})
	})
})
