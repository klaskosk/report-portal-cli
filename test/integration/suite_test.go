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
)

const (
	testAdminUser     = "superadmin"
	testAdminPassword = "test-password"
	testProject       = "superadmin_personal"
)

var (
	// baseURL is the Report Portal gateway URL discovered after containers start.
	baseURL string

	// bearerToken is the API token obtained from the UAT service during setup.
	bearerToken string

	// rpBinaryPath is the path to the compiled rp CLI binary.
	rpBinaryPath string
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	ctx := context.Background()

	By("starting the Report Portal compose stack")

	composeFile := filepath.Join(".", "docker-compose.test.yml")
	stack, err := compose.NewDockerComposeWith(
		compose.WithStackFiles(composeFile),
		compose.StackIdentifier("rptest"),
	)
	Expect(err).NotTo(HaveOccurred(),
		"failed to create docker compose stack from %q", composeFile)

	DeferCleanup(func() {
		By("tearing down the compose stack")

		downCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := stack.Down(downCtx,
			compose.RemoveOrphans(true),
			compose.RemoveVolumes(true),
		)
		Expect(err).NotTo(HaveOccurred(),
			"failed to tear down the compose stack")
	})

	err = stack.
		WaitForService("api", wait.ForLog("Started ReportPortalApp").WithStartupTimeout(5*time.Minute)).
		WaitForService("uat", wait.ForLog("Started").WithStartupTimeout(5*time.Minute)).
		Up(ctx)
	if err != nil {
		dumpServiceLogs(ctx, stack, "api", "migrations", "gateway")
		Fail(
			fmt.Sprintf(
				"failed to start compose stack and wait for services to become healthy: %v",
				err,
			),
		)
	}

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
	}).WithTimeout(60 * time.Second).WithPolling(2 * time.Second).Should(Succeed(),
		"failed to obtain API token for user %q", testAdminUser)
	GinkgoWriter.Println("API token obtained successfully")

	By("building the rp CLI binary")

	rpBinaryPath, err = buildCLIBinary()
	Expect(err).NotTo(HaveOccurred(),
		"failed to build the rp CLI binary")
	GinkgoWriter.Printf("CLI binary built at: %s\n", rpBinaryPath)

	DeferCleanup(func() {
		os.RemoveAll(filepath.Dir(rpBinaryPath))
	})

	By("seeding demo data")

	// The API route may take a moment to register with Traefik after the service starts.
	Eventually(func() error {
		return seedDemoData(ctx, baseURL, bearerToken, testProject)
	}).WithTimeout(60 * time.Second).WithPolling(2 * time.Second).Should(Succeed(),
		"failed to seed demo data into project %q", testProject)
	GinkgoWriter.Println("Demo data seeded")
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

// buildCLIBinary compiles the rp binary into a temporary directory.
func buildCLIBinary() (string, error) {
	tmpDir, err := os.MkdirTemp("", "rp-integration-test-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}

	binaryPath := filepath.Join(tmpDir, "rp")

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/rp")
	cmd.Dir = filepath.Join("..", "..")
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)

		return "", fmt.Errorf("building CLI binary: %w", err)
	}

	return binaryPath, nil
}
