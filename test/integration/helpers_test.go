//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	. "github.com/onsi/ginkgo/v2"
)

// httpClient is used for all outbound HTTP calls in the test suite,
// with a timeout to prevent hung requests from stalling the suite.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// newClient creates a client.Client pointing at the live Report Portal instance.
func newClient() *client.Client {
	c, err := client.New(baseURL, bearerToken)
	if err != nil {
		panic(fmt.Sprintf("failed to create test client: %v", err))
	}

	return c
}

// obtainAPIToken authenticates against the UAT service and returns a bearer token.
// Report Portal uses OAuth2 with the "ui" client (no secret).
func obtainAPIToken(ctx context.Context, rpBaseURL, username, password string) (string, error) {
	tokenURL := rpBaseURL + "/uat/sso/oauth/token"

	form := url.Values{
		"grant_type": {"password"},
		"username":   {username},
		"password":   {password},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("ui", "uiman")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)

		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	return tokenResponse.AccessToken, nil
}

// seedDemoData generates sample launches, test items, and logs in the given project
// using the Report Portal demo data endpoint.
func seedDemoData(ctx context.Context, rpBaseURL, token, projectName string) error {
	demoURL := fmt.Sprintf("%s/api/v1/demo/%s/generate", rpBaseURL, projectName)

	payload := `{"createDashboard": false, "launchesQuantity": 3}`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, demoURL, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating demo data request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("requesting demo data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("demo data request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// createProject creates a new Report Portal project with the given name.
// It returns the project name for convenience.
func createProject(ctx context.Context, rpBaseURL, token, projectName string) error {
	projectURL := rpBaseURL + "/api/v1/project"

	payload := fmt.Sprintf(`{"projectName":%q,"entryType":"INTERNAL"}`, projectName)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, projectURL, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating project request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("requesting project creation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("project creation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// deleteProject deletes the named Report Portal project. Errors are logged
// but not returned so cleanup always proceeds.
func deleteProject(ctx context.Context, rpBaseURL, token, projectName string) {
	projectURL := fmt.Sprintf("%s/api/v1/project/%s", rpBaseURL, projectName)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, projectURL, nil)
	if err != nil {
		GinkgoWriter.Printf("warning: could not build delete-project request for %q: %v\n", projectName, err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		GinkgoWriter.Printf("warning: delete-project request failed for %q: %v\n", projectName, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		GinkgoWriter.Printf("warning: delete-project returned %d for %q: %s\n", resp.StatusCode, projectName, string(respBody))
	}
}

// cliResult holds the output from a CLI invocation.
type cliResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// runCLI executes the built rp binary with the given arguments. It automatically injects
// --url and --token flags pointing at the live Report Portal instance.
func runCLI(args ...string) cliResult {
	fullArgs := append([]string{"--url", baseURL, "--token", bearerToken}, args...)

	cmd := exec.Command(rpBinaryPath, fullArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return cliResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// runCLIWithArgs is like runCLI but does not inject --url and --token automatically,
// allowing callers to provide their own (or omit them). The subprocess environment
// is sanitized to remove RP_* variables and prevent reading a host config file.
func runCLIWithArgs(args ...string) cliResult {
	cmd := exec.Command(rpBinaryPath, args...)
	cmd.Env = sanitizedEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return cliResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// runCLIWithEnv is like runCLIWithArgs but also sets the given environment
// variables for the subprocess, enabling tests that verify env-based config.
func runCLIWithEnv(env map[string]string, args ...string) cliResult {
	cmd := exec.Command(rpBinaryPath, args...)
	cmd.Env = sanitizedEnv()

	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return cliResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// sanitizedEnv returns a copy of the current environment with all RP_* variables
// removed and XDG_CONFIG_HOME pointed at a non-existent directory so the CLI
// cannot accidentally pick up connection details from the host.
func sanitizedEnv() []string {
	var env []string

	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "RP_") || key == "XDG_CONFIG_HOME" {
			continue
		}

		env = append(env, entry)
	}

	env = append(env, "XDG_CONFIG_HOME=/tmp/rp-integration-test-no-config")

	return env
}
