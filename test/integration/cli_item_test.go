//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type itemListEntry struct {
	ID     *int64  `json:"id"`
	UUID   *string `json:"uuid"`
	Name   *string `json:"name"`
	Type   *string `json:"type"`
	Parent *int64  `json:"parent"`
}

var _ = Describe("CLI: get item", Label("cli", "items"), func() {
	var (
		projectName string
		seeded      seededTestItems
	)

	BeforeEach(func(ctx SpecContext) {
		projectName = fmt.Sprintf("cli_item_test_%d", time.Now().UnixNano())

		By("creating isolated project " + projectName)
		err := createProject(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to create test project %q", projectName)

		By("seeding one launch with suite and child test items")
		seeded, err = seedTestItems(ctx, baseURL, bearerToken, projectName)
		Expect(err).To(Succeed(), "failed to seed test items into %q", projectName)
	}, NodeTimeout(90*time.Second))

	AfterEach(func(ctx SpecContext) {
		if projectName == "" {
			return
		}

		deleteProject(ctx, baseURL, bearerToken, projectName)
	}, NodeTimeout(15*time.Second))

	Context("listing items", func() {
		It("renders table output", func() {
			result := runProjectItem(
				projectName,
				"--launch",
				strconv.FormatInt(seeded.LaunchID, 10),
			)
			expectCLISuccess(result, "rp get item --launch")
			Expect(result.Stdout).To(ContainSubstring("ID"))
			Expect(result.Stdout).To(ContainSubstring("UUID"))
			Expect(result.Stdout).To(ContainSubstring("TYPE"))
		})

		It("renders json output", func() {
			result := runProjectItem(projectName,
				"-o", "json",
				"--launch", strconv.FormatInt(seeded.LaunchID, 10),
			)
			expectCLISuccess(result, "rp get item -o json --launch")
			requireItemList(result)
		})

		It("renders go-template output", func() {
			result := runProjectItem(projectName,
				"-o", `go-template={{len .}}`,
				"--launch", strconv.FormatInt(seeded.LaunchID, 10),
			)
			expectCLISuccess(result, "rp get item -o go-template=... --launch")
			Expect(result.Stdout).NotTo(BeEmpty(),
				"go-template output should include rendered output")
		})
	})

	Context("getting a single item", func() {
		It("returns an item by id", func() {
			result := runProjectItem(projectName, strconv.FormatInt(seeded.SuiteID, 10))
			expectCLISuccess(result, fmt.Sprintf("rp get item %d", seeded.SuiteID))
			Expect(result.Stdout).To(ContainSubstring(strconv.FormatInt(seeded.SuiteID, 10)))
		})

		It("returns an item by uuid", func() {
			result := runProjectItem(projectName, seeded.SuiteUUID)
			expectCLISuccess(result, fmt.Sprintf("rp get item %s", seeded.SuiteUUID))
			Expect(result.Stdout).To(ContainSubstring(seeded.SuiteUUID))
		})

		It("exits non-zero for a missing item", func() {
			result := runProjectItem(projectName, "999999")
			expectCLIError(result, "rp get item 999999")
			Expect(result.Stderr).To(ContainSubstring("Error"))
		})
	})

	Context("filter flags", func() {
		It("filters items by launch id", func() {
			result := runProjectItem(projectName,
				"-o", "json",
				"--launch", strconv.FormatInt(seeded.LaunchID, 10),
			)
			expectCLISuccess(result, "rp get item --launch")
			items := requireItemList(result)
			Expect(items).NotTo(BeEmpty())
		})

		It("filters suites with --suites shortcut", func() {
			result := runProjectItem(projectName,
				"-o", "json",
				"--launch", strconv.FormatInt(seeded.LaunchID, 10),
				"--suites",
			)
			expectCLISuccess(result, "rp get item --launch --suites")
			items := requireItemList(result)

			for _, item := range items {
				Expect(item.Type).NotTo(BeNil())
				Expect(*item.Type).To(Equal("SUITE"))
			}
		})

		It("filters by explicit type", func() {
			result := runProjectItem(projectName,
				"-o", "json",
				"--launch", strconv.FormatInt(seeded.LaunchID, 10),
				"--type", "TEST",
			)
			expectCLISuccess(result, "rp get item --type TEST")
			items := requireItemList(result)
			Expect(items).NotTo(BeEmpty())

			for _, item := range items {
				Expect(item.Type).NotTo(BeNil())
				Expect(*item.Type).To(Equal("TEST"))
			}
		})

		It("filters children by parent suite id", func() {
			result := runProjectItem(projectName,
				"-o", "json",
				"--launch", strconv.FormatInt(seeded.LaunchID, 10),
				"--parent", strconv.FormatInt(seeded.SuiteID, 10),
			)
			expectCLISuccess(result, "rp get item --parent")
			items := requireItemList(result)
			Expect(items).NotTo(BeEmpty())

			for _, item := range items {
				Expect(item.Parent).NotTo(BeNil())
				Expect(*item.Parent).To(Equal(seeded.SuiteID))
			}
		})
	})

	It("prints errors as JSON when -o json is set", func() {
		result := runProjectItem(projectName, "-o", "json", "999999")
		expectCLIError(result, "rp get item 999999 -o json")
		expectJSONError(result, "not found")
		Expect(result.Stderr).To(BeEmpty(), "stderr should be empty in JSON mode")
	})
})

func runProjectItem(projectName string, args ...string) cliResult {
	GinkgoHelper()

	fullArgs := append([]string{"get", "item"}, args...)
	fullArgs = append(fullArgs, "-p", projectName)

	return runCLI(fullArgs...)
}

func requireItemList(result cliResult) []itemListEntry {
	GinkgoHelper()

	expectCLISuccess(result, "rp get item -o json")

	var items []itemListEntry
	Expect(json.Unmarshal([]byte(result.Stdout), &items)).To(Succeed(),
		"stdout should be valid JSON array, got: %s", result.Stdout)
	Expect(items).NotTo(BeEmpty(),
		"JSON output should contain at least one item")

	return items
}
