//go:build integration

package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CLI: download", Label("cli", "download"), func() {
	var (
		c                *client.Client
		projectName      string
		seededTestData   seededTestItems
		attachmentDataID int64
	)

	BeforeEach(func(ctx SpecContext) {
		c = newClient()
		projectName = fmt.Sprintf("cli_download_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, c, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding one launch with suite and child test items")
		seededTestData, err = seedTestItems(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to seed test items into %q", projectName)

		By("seeding a log entry with an attachment")
		attachmentDataID, err = seedLogWithAttachment(
			ctx,
			c,
			projectName,
			seededTestData,
		)
		if err != nil {
			Skip("could not seed attachment: " + err.Error())
		}
		// RP 5.15 async indexing requires longer startup for item/log hierarchy
	}, NodeTimeout(240*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName == "" {
			return
		}

		deleteProject(ctx, c, projectName)
	}, NodeTimeout(15*time.Second))

	It("writes file to stdout", func() {
		Expect(attachmentDataID).NotTo(BeZero(), "expected seeded attachment ID")

		dataIDStr := strconv.FormatInt(attachmentDataID, 10)
		result := runCLI("download", dataIDStr, "-p", projectName)
		expectCLISuccess(result, "rp download <data-id>")
		Expect(result.Stdout).NotTo(BeEmpty(), "stdout should contain file content")
	})

	It("writes file to disk with --output-file", func() {
		Expect(attachmentDataID).NotTo(BeZero(), "expected seeded attachment ID")

		tmpDir, err := os.MkdirTemp("", "rp-download-test-*")
		Expect(err).NotTo(HaveOccurred())

		defer os.RemoveAll(tmpDir)

		dataIDStr := strconv.FormatInt(attachmentDataID, 10)
		outPath := filepath.Join(tmpDir, "downloaded.txt")
		result := runCLI("download", dataIDStr, "-p", projectName, "--output-file", outPath)
		expectCLISuccess(result, "rp download --output-file")

		data, err := os.ReadFile(outPath)
		Expect(err).NotTo(HaveOccurred(), "should be able to read the downloaded file")
		Expect(data).NotTo(BeEmpty(), "downloaded file should not be empty")
	})

	It("exits non-zero for a non-existent data ID", func() {
		result := runCLI("download", "999999", "-p", projectName)
		expectCLIError(result, "rp download 999999")
	})

	It("prints errors as JSON when -o json is set", func() {
		result := runCLI("download", "999999", "-p", projectName, "-o", "json")
		expectCLIError(result, "rp download 999999 -o json")
		expectJSONError(result, "")
		Expect(result.Stdout).To(BeEmpty(), "stdout should be empty when JSON error goes to stderr")
	})
})
