//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// httpClient is used for all outbound HTTP calls in the test suite,
// with a timeout to prevent hung requests from stalling the suite.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// newClient creates a client.Client pointing at the live Report Portal instance.
func newClient() *client.Client {
	GinkgoHelper()

	c, err := client.New(baseURL, bearerToken)
	Expect(err).NotTo(HaveOccurred(), "failed to create test client")

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

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		tokenURL,
		strings.NewReader(form.Encode()),
	)
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

		return "", fmt.Errorf(
			"token request failed with status %d: %s",
			resp.StatusCode,
			string(respBody),
		)
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	return tokenResponse.AccessToken, nil
}

// seedTestLaunches creates the given number of launches in the project using the
// StartLaunch and FinishLaunch API endpoints. This provides predictable, version-controlled
// test data instead of relying on Report Portal's demo data endpoint.
func seedTestLaunches(ctx context.Context, rpBaseURL, token, projectName string, count int) error {
	testClient, err := client.New(rpBaseURL, token)
	if err != nil {
		return fmt.Errorf("creating client for launch seeding: %w", err)
	}

	launches := testClient.Project(projectName).Launches()
	finishedStatus := types.FinishExecutionRQStatusPASSED

	now := time.Now()
	for i := 0; i < count; i++ {
		startTime := now.Add(-time.Duration(count-i) * time.Minute)
		endTime := startTime.Add(30 * time.Second)
		launchName := fmt.Sprintf("integration-test-launch-%d", i+1)

		startedLaunch, err := launches.Start(ctx, types.StartLaunchRQ{
			Name:      launchName,
			StartTime: types.EpochMillis(startTime),
		})
		if err != nil {
			return fmt.Errorf("starting launch %q: %w", launchName, err)
		}
		if startedLaunch == nil || startedLaunch.Id == nil || *startedLaunch.Id == "" {
			return fmt.Errorf("start launch %q returned empty id", launchName)
		}

		_, err = launches.Finish(ctx, *startedLaunch.Id, types.FinishExecutionRQ{
			EndTime: types.EpochMillis(endTime),
			Status:  &finishedStatus,
		})
		if err != nil {
			return fmt.Errorf("finishing launch %q: %w", launchName, err)
		}
	}

	return nil
}

type seededTestItems struct {
	LaunchID  int64
	SuiteID   int64
	SuiteUUID string
}

const (
	seedResolutionAttempts = 20
	seedResolutionInterval = 250 * time.Millisecond
)

// waitForSeedResolution handles eventual consistency in Report Portal indexing
// after async start/finish API calls.
func waitForSeedResolution(
	ctx context.Context,
	attempts int,
	interval time.Duration,
	check func() bool,
) bool {
	for attempt := range attempts {
		if check() {
			return true
		}

		if attempt == attempts-1 {
			break
		}

		select {
		case <-ctx.Done():
			return false
		case <-time.After(interval):
		}
	}

	return false
}

// seedTestItems creates one launch with one SUITE and one TEST child item and returns their resolved IDs/UUIDs for use
// in integration tests.
func seedTestItems(
	ctx context.Context,
	rpBaseURL, token, projectName string,
) (seededTestItems, error) {
	testClient, err := client.New(rpBaseURL, token)
	if err != nil {
		return seededTestItems{}, fmt.Errorf("creating client for item seeding: %w", err)
	}

	project := testClient.Project(projectName)
	launches := project.Launches()
	items := project.Items()

	now := time.Now()
	launchName := fmt.Sprintf("integration-item-launch-%d", now.UnixNano())
	suiteName := fmt.Sprintf("integration-suite-%d", now.UnixNano())
	testName := fmt.Sprintf("integration-test-%d", now.UnixNano())

	refs, err := startSeededItemHierarchy(
		ctx,
		launches,
		items,
		now,
		launchName,
		suiteName,
		testName,
	)
	if err != nil {
		return seededTestItems{}, err
	}

	if err := finishSeededItemHierarchy(ctx, launches, items, now, refs); err != nil {
		return seededTestItems{}, err
	}

	resolved, err := resolveSeededItemIdentifiers(ctx, launches, items, refs)
	if err != nil {
		return seededTestItems{}, err
	}

	return resolved, nil
}

type seededItemRefs struct {
	LaunchUUID string
	SuiteRef   string
	TestRef    string
}

// startSeededItemHierarchy creates a launch with one SUITE root and one TEST child. It returns async references that
// are later finished and resolved to numeric IDs.
func startSeededItemHierarchy(
	ctx context.Context,
	launches *client.LaunchScope,
	items *client.TestItemScope,
	now time.Time,
	launchName, suiteName, testName string,
) (seededItemRefs, error) {
	startedLaunch, err := launches.Start(ctx, types.StartLaunchRQ{
		Name:      launchName,
		StartTime: types.EpochMillis(now),
	})
	if err != nil {
		return seededItemRefs{}, fmt.Errorf("starting launch for item seeding: %w", err)
	}
	if startedLaunch == nil || startedLaunch.Id == nil || *startedLaunch.Id == "" {
		return seededItemRefs{}, fmt.Errorf("start launch returned empty id")
	}

	launchUUID := *startedLaunch.Id
	hasStats := true

	suiteType := types.StartTestItemRQType("SUITE")
	startedSuite, err := items.StartRoot(ctx, types.StartTestItemRQ{
		LaunchUuid: launchUUID,
		Name:       suiteName,
		StartTime:  types.EpochMillis(now),
		Type:       suiteType,
		HasStats:   &hasStats,
	})
	if err != nil {
		return seededItemRefs{}, fmt.Errorf("starting suite item: %w", err)
	}
	if startedSuite == nil || startedSuite.Id == nil || *startedSuite.Id == "" {
		return seededItemRefs{}, fmt.Errorf("starting suite item returned empty id")
	}

	suiteRef := *startedSuite.Id
	testType := types.StartTestItemRQType("TEST")
	startedTest, err := items.StartChild(ctx, suiteRef, types.StartTestItemRQ{
		LaunchUuid: launchUUID,
		Name:       testName,
		StartTime:  types.EpochMillis(now.Add(1 * time.Second)),
		Type:       testType,
		HasStats:   &hasStats,
	})
	if err != nil {
		return seededItemRefs{}, fmt.Errorf("starting child test item: %w", err)
	}
	if startedTest == nil || startedTest.Id == nil || *startedTest.Id == "" {
		return seededItemRefs{}, fmt.Errorf("starting child test item returned empty id")
	}

	return seededItemRefs{
		LaunchUUID: launchUUID,
		SuiteRef:   suiteRef,
		TestRef:    *startedTest.Id,
	}, nil
}

// finishSeededItemHierarchy closes the child item, parent suite, and launch in that order so Report Portal persists a
// complete hierarchy for listing tests.
func finishSeededItemHierarchy(
	ctx context.Context,
	launches *client.LaunchScope,
	items *client.TestItemScope,
	now time.Time,
	refs seededItemRefs,
) error {
	finishedItemStatus := types.FinishTestItemRQStatusPASSED
	finishItem := func(itemRef string, endTime time.Time) error {
		_, finishErr := items.Finish(ctx, itemRef, types.FinishTestItemRQ{
			LaunchUuid: refs.LaunchUUID,
			EndTime:    types.EpochMillis(endTime),
			Status:     &finishedItemStatus,
		})
		if finishErr != nil {
			return fmt.Errorf("finishing item %q: %w", itemRef, finishErr)
		}

		return nil
	}

	if err := finishItem(refs.TestRef, now.Add(5*time.Second)); err != nil {
		return err
	}
	if err := finishItem(refs.SuiteRef, now.Add(6*time.Second)); err != nil {
		return err
	}

	finishedLaunchStatus := types.FinishExecutionRQStatusPASSED
	_, err := launches.Finish(ctx, refs.LaunchUUID, types.FinishExecutionRQ{
		EndTime: types.EpochMillis(now.Add(7 * time.Second)),
		Status:  &finishedLaunchStatus,
	})
	if err != nil {
		return fmt.Errorf("finishing launch for item seeding: %w", err)
	}

	return nil
}

// resolveSeededItemIdentifiers resolves async references into the stable IDs consumed by integration tests and waits
// for child indexing before returning.
func resolveSeededItemIdentifiers(
	ctx context.Context,
	launches *client.LaunchScope,
	items *client.TestItemScope,
	refs seededItemRefs,
) (seededTestItems, error) {
	suiteID, suiteUUID, err := resolveItemByUUID(ctx, items, refs.SuiteRef)
	if err != nil {
		return seededTestItems{}, fmt.Errorf("resolving suite item identifiers: %w", err)
	}

	// Ensure the child item is indexed before item-list tests execute.
	if _, _, err := resolveItemByUUID(ctx, items, refs.TestRef); err != nil {
		return seededTestItems{}, fmt.Errorf("resolving child test item identifiers: %w", err)
	}

	launchID, err := resolveLaunchIDByUUID(ctx, launches, refs.LaunchUUID)
	if err != nil {
		return seededTestItems{}, err
	}

	return seededTestItems{
		LaunchID:  launchID,
		SuiteID:   suiteID,
		SuiteUUID: suiteUUID,
	}, nil
}

// resolveItemByUUID polls GetByUUID until the item has both numeric ID and UUID, which avoids flakes from asynchronous
// indexing in Report Portal.
func resolveItemByUUID(
	ctx context.Context,
	items *client.TestItemScope,
	itemUUID string,
) (int64, string, error) {
	var (
		resolvedItemID   int64
		resolvedItemUUID string
	)

	resolved := waitForSeedResolution(
		ctx,
		seedResolutionAttempts,
		seedResolutionInterval,
		func() bool {
			item, getErr := items.GetByUUID(ctx, itemUUID)
			if getErr == nil && item != nil && item.Id != nil && item.Uuid != nil {
				if *item.Id != 0 && *item.Uuid != "" {
					resolvedItemID = *item.Id
					resolvedItemUUID = *item.Uuid

					return true
				}
			}

			return false
		},
	)
	if !resolved {
		if ctx.Err() != nil {
			return 0, "", fmt.Errorf("could not resolve item by uuid %q: %w", itemUUID, ctx.Err())
		}

		return 0, "", fmt.Errorf("could not resolve item by uuid %q", itemUUID)
	}

	return resolvedItemID, resolvedItemUUID, nil
}

// resolveLaunchIDByUUID polls GetByUUID until the launch has a non-zero ID.
func resolveLaunchIDByUUID(
	ctx context.Context,
	launches *client.LaunchScope,
	launchUUID string,
) (int64, error) {
	var launchID int64

	resolved := waitForSeedResolution(
		ctx,
		seedResolutionAttempts,
		seedResolutionInterval,
		func() bool {
			launch, getErr := launches.GetByUUID(ctx, launchUUID)
			if getErr == nil && launch != nil && launch.Id != 0 {
				launchID = launch.Id

				return true
			}

			return false
		},
	)
	if !resolved {
		if ctx.Err() != nil {
			return 0, fmt.Errorf("could not resolve launch id: %w", ctx.Err())
		}

		return 0, fmt.Errorf("could not resolve launch ID for seeded test items")
	}

	return launchID, nil
}

// createProject creates a new Report Portal project with the given name.
// It returns the project name for convenience.
func createProject(ctx context.Context, rpBaseURL, token, projectName string) error {
	projectURL := rpBaseURL + "/api/v1/project"

	payload := fmt.Sprintf(`{"projectName":%q,"entryType":"INTERNAL"}`, projectName)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		projectURL,
		strings.NewReader(payload),
	)
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

		return fmt.Errorf(
			"project creation failed with status %d: %s",
			resp.StatusCode,
			string(respBody),
		)
	}

	return nil
}

// deleteProject deletes the named Report Portal project. Errors are logged
// but not returned so cleanup always proceeds.
func deleteProject(ctx context.Context, rpBaseURL, token, projectName string) {
	projectURL := fmt.Sprintf("%s/api/v1/project/%s", rpBaseURL, projectName)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, projectURL, nil)
	if err != nil {
		GinkgoWriter.Printf(
			"warning: could not build delete-project request for %q: %v\n",
			projectName,
			err,
		)

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
		GinkgoWriter.Printf(
			"warning: delete-project returned %d for %q: %s\n",
			resp.StatusCode,
			projectName,
			string(respBody),
		)
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
	setCoverDir(cmd)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0

	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
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
	setCoverDir(cmd)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0

	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
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
	setCoverDir(cmd)

	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0

	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
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

// setCoverDir adds GOCOVERDIR to cmd's environment so the coverage-instrumented
// binary writes its profile data. If cmd.Env is nil (inheriting the parent env)
// it is first populated from os.Environ().
func setCoverDir(cmd *exec.Cmd) {
	if coverDir == "" {
		return
	}

	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}

	cmd.Env = append(cmd.Env, "GOCOVERDIR="+coverDir)
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

func writeTempConfig(content string) (string, error) {
	dir, err := os.MkdirTemp("", "rp-config-*")
	if err != nil {
		return "", fmt.Errorf("creating temp config dir: %w", err)
	}

	path := filepath.Join(dir, "config.yaml")
	trimmed := strings.TrimSpace(content) + "\n"
	if err := os.WriteFile(path, []byte(trimmed), 0o600); err != nil {
		os.RemoveAll(dir)

		return "", fmt.Errorf("writing config file: %w", err)
	}

	return path, nil
}

func expectJSONError(result cliResult, expectedSubstring string) {
	GinkgoHelper()

	Expect(result.Stdout).NotTo(BeEmpty(), "stdout should contain a JSON error payload")

	var payload map[string]string
	err := json.Unmarshal([]byte(result.Stdout), &payload)
	Expect(err).NotTo(HaveOccurred(), "stdout should contain valid JSON")
	Expect(payload).To(HaveKey("error"), "JSON error payload should include an error field")
	if expectedSubstring != "" {
		Expect(
			strings.ToLower(payload["error"]),
		).To(ContainSubstring(strings.ToLower(expectedSubstring)))
	}
}

func expectLogLevel(result cliResult, level string, description string) {
	GinkgoHelper()

	Expect(result.Stderr).To(ContainSubstring("level="+level),
		"%s should emit %s logs; stderr: %s", description, level, result.Stderr)
}
