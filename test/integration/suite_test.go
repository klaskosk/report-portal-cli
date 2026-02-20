//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/klaskosk/report-portal-cli/test/integration/internal/runid"
)

const (
	testAdminUser     = "superadmin"
	testAdminPassword = "test-password"
)

var (
	// baseURL is the Report Portal gateway URL discovered after containers start.
	baseURL string

	// bearerToken is the API token obtained from the UAT service during setup.
	bearerToken string

	// rpBinaryPath is the path to the compiled rp CLI binary.
	rpBinaryPath string

	// coverDir is the directory where the coverage-instrumented binary writes
	// its profiling data. Each invocation of the binary appends a new counter
	// file, and after the suite the data can be merged with go tool covdata.
	coverDir string
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	By("generating a unique stack identifier for this run")

	stackID := runid.Generate()
	GinkgoWriter.Printf("Stack identifier: %s\n", stackID)

	Expect(os.Setenv("RP_TEST_STACK_ID", stackID)).To(Succeed(),
		"failed to set RP_TEST_STACK_ID")

	By("starting the Report Portal compose stack")

	composeFile := filepath.Join(".", "docker-compose.test.yml")
	stack, err := compose.NewDockerComposeWith(
		compose.WithStackFiles(composeFile),
		compose.StackIdentifier(stackID),
	)
	Expect(err).NotTo(HaveOccurred(),
		"failed to create docker compose stack from %q", composeFile)

	DeferCleanup(func(ctx SpecContext) {
		By("tearing down the compose stack")

		err := stack.Down(ctx,
			compose.RemoveOrphans(true),
			compose.RemoveVolumes(true),
		)
		Expect(err).NotTo(HaveOccurred(),
			"failed to tear down the compose stack")
	}, NodeTimeout(30*time.Second))

	err = stack.
		WithOsEnv().
		WaitForService(
			"api",
			wait.ForLog("Started ReportPortalApp").WithStartupTimeout(5*time.Minute),
		).
		WaitForService("uat", wait.ForLog("Started").WithStartupTimeout(5*time.Minute)).
		Up(ctx)
	if err != nil {
		dumpServiceLogs(ctx, stack, "api", "migrations", "gateway")
	}
	Expect(err).To(Succeed(),
		"failed to start compose stack and wait for services to become healthy: %v", err)

	By("discovering the gateway port")

	gatewayContainer, err := stack.ServiceContainer(ctx, "gateway")
	Expect(err).NotTo(HaveOccurred(),
		"failed to get gateway service container")

	mappedPort, err := gatewayContainer.MappedPort(ctx, "8080/tcp")
	Expect(err).NotTo(HaveOccurred(),
		"failed to get mapped port for gateway 8080/tcp")

	host, err := gatewayContainer.Host(ctx)
	Expect(err).NotTo(HaveOccurred(),
		"failed to get host for gateway container")

	baseURL = fmt.Sprintf("http://%s:%s", host, mappedPort.Port())
	GinkgoWriter.Printf("Report Portal base URL: %s\n", baseURL)

	By("obtaining an API token from UAT")

	// Traefik may take a moment to register the UAT route after the service starts.
	Eventually(func() error {
		bearerToken, err = obtainAPIToken(ctx, baseURL, testAdminUser, testAdminPassword)

		return err
	}).WithTimeout(60*time.Second).WithPolling(2*time.Second).Should(Succeed(),
		"failed to obtain API token for user %q", testAdminUser)
	GinkgoWriter.Println("API token obtained successfully")

	By("building the rp CLI binary")

	rpBinaryPath, err = buildCLIBinary()
	Expect(err).NotTo(HaveOccurred(),
		"failed to build the rp CLI binary")
	GinkgoWriter.Printf("CLI binary built at: %s\n", rpBinaryPath)

	DeferCleanup(func() {
		if coverDir != "" {
			mergeCoverage(coverDir, stackID)
		}

		os.RemoveAll(filepath.Dir(rpBinaryPath))
	})
})

// dumpServiceLogs writes the container logs for the named services to GinkgoWriter
// to aid debugging when the compose stack fails to start.
func dumpServiceLogs(ctx context.Context, stack *compose.DockerCompose, services ...string) {
	for _, svc := range services {
		container, err := stack.ServiceContainer(ctx, svc)
		if err != nil {
			GinkgoWriter.Printf("[%s] could not get container: %v\n", svc, err)

			continue
		}

		logs, err := container.Logs(ctx)
		if err != nil {
			GinkgoWriter.Printf("[%s] could not get logs: %v\n", svc, err)

			continue
		}

		body, _ := io.ReadAll(logs)
		logs.Close()

		GinkgoWriter.Printf("=== Logs for %s ===\n%s\n=== End %s ===\n", svc, string(body), svc)
	}
}

// buildCLIBinary compiles a coverage-instrumented rp binary into a temporary
// directory. It also creates the GOCOVERDIR directory where the binary will
// write its coverage counter files at exit.
func buildCLIBinary() (string, error) {
	tmpDir, err := os.MkdirTemp("", "rp-integration-test-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}

	coverDir = filepath.Join(tmpDir, "covdata")
	if err := os.MkdirAll(coverDir, 0o755); err != nil {
		os.RemoveAll(tmpDir)

		return "", fmt.Errorf("creating coverage data dir: %w", err)
	}

	if err := os.Setenv("GOCOVERDIR", coverDir); err != nil {
		os.RemoveAll(tmpDir)

		return "", fmt.Errorf("setting GOCOVERDIR: %w", err)
	}

	binaryPath := filepath.Join(tmpDir, "rp")

	cmd := exec.Command("go", "build",
		"-cover",
		"-coverpkg=github.com/klaskosk/report-portal-cli/...,github.com/klaskosk/report-portal-cli/pkg/...",
		"-o", binaryPath, "./cmd/rp",
	)
	cmd.Dir = filepath.Join("..", "..")
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)

		return "", fmt.Errorf("building CLI binary: %w", err)
	}

	GinkgoWriter.Printf("Coverage data dir: %s\n", coverDir)

	return binaryPath, nil
}

// mergeCoverage converts the raw coverage counter files into a Go coverage
// profile in the current directory and prints a per-function summary. The
// output filename includes the stack ID so that concurrent runs do not
// overwrite each other. The output file can be loaded with go tool cover.
func mergeCoverage(dataDir, stackID string) {
	outFile := fmt.Sprintf("integration-%s.coverprofile", stackID)

	textfmt := exec.Command("go", "tool", "covdata", "textfmt", "-i="+dataDir, "-o="+outFile)
	textfmt.Stdout = GinkgoWriter
	textfmt.Stderr = GinkgoWriter

	if err := textfmt.Run(); err != nil {
		GinkgoWriter.Printf("warning: could not convert coverage data: %v\n", err)

		return
	}

	GinkgoWriter.Printf("Coverage profile written to %s\n", outFile)

	funcCmd := exec.Command("go", "tool", "cover", "-func="+outFile)
	funcCmd.Stdout = GinkgoWriter
	funcCmd.Stderr = GinkgoWriter

	if err := funcCmd.Run(); err != nil {
		GinkgoWriter.Printf("warning: could not print coverage summary: %v\n", err)
	}
}
