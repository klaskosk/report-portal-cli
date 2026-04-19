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
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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
			StartTime: startTime,
			Uuid:      uuid.NewString(),
		})
		if err != nil {
			return fmt.Errorf("starting launch %q: %w", launchName, err)
		}
		if startedLaunch == nil || startedLaunch.Id == nil || *startedLaunch.Id == "" {
			return fmt.Errorf("start launch %q returned empty uuid", launchName)
		}

		finishErr := retryTransient(ctx, 3, func() error {
			_, err := launches.Finish(ctx, *startedLaunch.Id, types.FinishExecutionRQ{
				EndTime: endTime,
				Status:  &finishedStatus,
			})

			return err
		})
		if finishErr != nil {
			return fmt.Errorf("finishing launch %q: %w", launchName, finishErr)
		}
	}

	return nil
}

type seededTestItems struct {
	LaunchID   int64
	LaunchUUID string
	SuiteID    int64
	SuiteUUID  string
	TestID     int64
	TestUUID   string
}

const (
	seedResolutionAttempts = 60
	seedResolutionInterval = 1 * time.Second

	// Activity and audit events require longer indexing time.
	activityResolutionAttempts = 120
	activityResolutionInterval = 500 * time.Millisecond
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
// in integration tests. The v2 async API occasionally drops items under load, so the entire seed is retried up to 3
// times before giving up.
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

	const maxSeedAttempts = 3

	var lastErr error

	for attempt := range maxSeedAttempts {
		result, seedErr := trySeedTestItems(ctx, launches, items)
		if seedErr == nil {
			return result, nil
		}

		lastErr = seedErr

		if attempt < maxSeedAttempts-1 {
			GinkgoWriter.Printf("seed attempt %d failed (%v), retrying...\n", attempt+1, seedErr)
		}
	}

	return seededTestItems{}, lastErr
}

func trySeedTestItems(
	ctx context.Context,
	launches *client.LaunchScope,
	items *client.TestItemScope,
) (seededTestItems, error) {
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
	SuiteUUID  string
	TestUUID   string
}

// startSeededItemHierarchy creates a launch with one SUITE root and one TEST child. It returns the launch and item
// UUIDs produced by the async creation APIs, which are later finished and resolved to numeric IDs.
func startSeededItemHierarchy(
	ctx context.Context,
	launches *client.LaunchScope,
	items *client.TestItemScope,
	now time.Time,
	launchName, suiteName, testName string,
) (seededItemRefs, error) {
	launchUUID := uuid.NewString()
	startedLaunch, err := launches.Start(ctx, types.StartLaunchRQ{
		Name:      launchName,
		StartTime: now,
		Uuid:      launchUUID,
	})
	if err != nil {
		return seededItemRefs{}, fmt.Errorf("starting launch for item seeding: %w", err)
	}
	if startedLaunch == nil || startedLaunch.Id == nil || *startedLaunch.Id == "" {
		return seededItemRefs{}, fmt.Errorf("start launch returned empty uuid")
	}

	hasStats := true
	suiteUUID := uuid.NewString()

	suiteType := types.StartTestItemRQType("SUITE")
	startedSuite, err := items.StartRoot(ctx, types.StartTestItemRQ{
		LaunchUuid: launchUUID,
		Name:       suiteName,
		StartTime:  now,
		Type:       suiteType,
		HasStats:   &hasStats,
		Uuid:       suiteUUID,
	})
	if err != nil {
		return seededItemRefs{}, fmt.Errorf("starting suite item: %w", err)
	}
	if startedSuite == nil || startedSuite.Id == nil || *startedSuite.Id == "" {
		return seededItemRefs{}, fmt.Errorf("starting suite item returned empty uuid")
	}

	testUUID := uuid.NewString()
	testType := types.StartTestItemRQType("TEST")
	startedTest, err := items.StartChild(ctx, suiteUUID, types.StartTestItemRQ{
		LaunchUuid: launchUUID,
		Name:       testName,
		StartTime:  now.Add(1 * time.Second),
		Type:       testType,
		HasStats:   &hasStats,
		Uuid:       testUUID,
	})
	if err != nil {
		return seededItemRefs{}, fmt.Errorf("starting child test item: %w", err)
	}
	if startedTest == nil || startedTest.Id == nil || *startedTest.Id == "" {
		return seededItemRefs{}, fmt.Errorf("starting child test item returned empty uuid")
	}

	return seededItemRefs{
		LaunchUUID: launchUUID,
		SuiteUUID:  suiteUUID,
		TestUUID:   testUUID,
	}, nil
}

// retryTransient retries fn up to maxAttempts when it returns a 500-level error, which commonly indicates a transient
// PostgreSQL deadlock under parallel test load.
func retryTransient(ctx context.Context, maxAttempts int, fn func() error) error {
	var lastErr error
	for attempt := range maxAttempts {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if !client.IsServerError(lastErr) {
			return lastErr
		}

		if attempt < maxAttempts-1 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
			}
		}
	}

	return lastErr
}

// finishSeededItemHierarchy closes the child item, parent suite, and launch in that order so Report Portal persists a
// complete hierarchy for listing tests. Operations are retried on transient 500 errors (e.g. PostgreSQL deadlocks).
func finishSeededItemHierarchy(
	ctx context.Context,
	launches *client.LaunchScope,
	items *client.TestItemScope,
	now time.Time,
	refs seededItemRefs,
) error {
	finishedItemStatus := types.FinishTestItemRQStatusPASSED
	finishItem := func(itemUUID string, endTime time.Time) error {
		return retryTransient(ctx, 3, func() error {
			_, finishErr := items.Finish(ctx, itemUUID, types.FinishTestItemRQ{
				LaunchUuid: refs.LaunchUUID,
				EndTime:    endTime,
				Status:     &finishedItemStatus,
			})
			if finishErr != nil {
				return fmt.Errorf("finishing item %q: %w", itemUUID, finishErr)
			}

			return nil
		})
	}

	if err := finishItem(refs.TestUUID, now.Add(5*time.Second)); err != nil {
		return err
	}
	if err := finishItem(refs.SuiteUUID, now.Add(6*time.Second)); err != nil {
		return err
	}

	finishedLaunchStatus := types.FinishExecutionRQStatusPASSED

	return retryTransient(ctx, 3, func() error {
		_, err := launches.Finish(ctx, refs.LaunchUUID, types.FinishExecutionRQ{
			EndTime: now.Add(7 * time.Second),
			Status:  &finishedLaunchStatus,
		})
		if err != nil {
			return fmt.Errorf("finishing launch for item seeding: %w", err)
		}

		return nil
	})
}

// resolveSeededItemIdentifiers resolves async references into the stable IDs consumed by integration tests and waits
// for child indexing before returning.
func resolveSeededItemIdentifiers(
	ctx context.Context,
	launches *client.LaunchScope,
	items *client.TestItemScope,
	refs seededItemRefs,
) (seededTestItems, error) {
	suiteID, suiteUUID, err := resolveItemByUUID(ctx, items, refs.SuiteUUID)
	if err != nil {
		return seededTestItems{}, fmt.Errorf("resolving suite item identifiers: %w", err)
	}

	// Ensure the child item is indexed before item-list tests execute.
	testID, testUUID, err := resolveItemByUUID(ctx, items, refs.TestUUID)
	if err != nil {
		return seededTestItems{}, fmt.Errorf("resolving child test item identifiers: %w", err)
	}

	launchID, err := resolveLaunchIDByUUID(ctx, launches, refs.LaunchUUID)
	if err != nil {
		return seededTestItems{}, err
	}

	return seededTestItems{
		LaunchID:   launchID,
		LaunchUUID: refs.LaunchUUID,
		SuiteID:    suiteID,
		SuiteUUID:  suiteUUID,
		TestID:     testID,
		TestUUID:   testUUID,
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

	var lastErr error

	resolved := waitForSeedResolution(
		ctx,
		seedResolutionAttempts,
		seedResolutionInterval,
		func() bool {
			item, getErr := items.GetByUUID(ctx, itemUUID)
			if getErr != nil {
				lastErr = getErr

				return false
			}

			lastErr = nil

			if item != nil && item.Id != nil && item.Uuid != nil {
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

		if lastErr != nil {
			return 0, "", fmt.Errorf(
				"could not resolve item by uuid %q (last error: %w)",
				itemUUID,
				lastErr,
			)
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
func createProject(ctx context.Context, c *client.Client, projectName string) error {
	_, err := c.Projects().Create(ctx, types.CreateProjectRQ{
		ProjectName: projectName,
		EntryType:   types.CreateProjectRQEntryTypeINTERNAL,
	})

	return err
}

// deleteProject deletes the named Report Portal project by first resolving its
// numeric ID, then calling the batch-delete endpoint. Errors are logged but not
// returned so cleanup always proceeds.
func deleteProject(ctx context.Context, c *client.Client, projectName string) {
	proj, err := c.Projects().Get(ctx, projectName)
	if err != nil {
		GinkgoWriter.Printf(
			"warning: could not look up project %q for deletion: %v\n",
			projectName,
			err,
		)

		return
	}

	_, err = c.Projects().Delete(ctx, proj.ProjectId)
	if err != nil {
		GinkgoWriter.Printf(
			"warning: delete-project failed for %q (id=%d): %v\n",
			projectName,
			proj.ProjectId,
			err,
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

// writeTempConfig writes content to a temporary config.yaml file and returns its path.
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

// expectCLISuccess asserts that the CLI exited with code 0.
func expectCLISuccess(result cliResult, description string) {
	GinkgoHelper()
	Expect(result.ExitCode).To(Equal(0),
		"%s should exit 0; stderr: %s", description, result.Stderr)
}

// expectCLIError asserts that the CLI exited with a non-zero exit code.
func expectCLIError(result cliResult, description string) {
	GinkgoHelper()
	Expect(result.ExitCode).NotTo(Equal(0),
		"%s should exit non-zero; stderr: %s", description, result.Stderr)
}

// expectJSONError asserts that stderr contains a JSON object with an "error" field matching the expected substring.
func expectJSONError(result cliResult, expectedSubstring string) {
	GinkgoHelper()

	Expect(result.Stderr).NotTo(BeEmpty(), "stderr should contain a JSON error payload")

	var payload map[string]string
	err := json.Unmarshal([]byte(result.Stderr), &payload)
	Expect(err).NotTo(HaveOccurred(), "stderr should contain valid JSON")
	Expect(payload).To(HaveKey("error"), "JSON error payload should include an error field")
	if expectedSubstring != "" {
		Expect(
			strings.ToLower(payload["error"]),
		).To(ContainSubstring(strings.ToLower(expectedSubstring)))
	}
}

// expectJSONObject asserts that jsonText decodes to a JSON object.
func expectJSONObject(jsonText string) {
	GinkgoHelper()

	var object map[string]json.RawMessage
	err := json.Unmarshal([]byte(jsonText), &object)
	Expect(err).NotTo(HaveOccurred(), "expected a JSON object, got: %s", jsonText)
}

// expectJSONArrayObjects asserts that jsonText decodes to a JSON array containing at least one object.
func expectJSONArrayObjects(jsonText string) {
	GinkgoHelper()

	var objects []map[string]json.RawMessage
	err := json.Unmarshal([]byte(jsonText), &objects)
	Expect(err).NotTo(HaveOccurred(), "expected a JSON array, got: %s", jsonText)
	Expect(objects).NotTo(BeEmpty(), "JSON array should contain at least one object")
}

// seedLogWithAttachment creates a log entry with a small text file attachment via the
// client's batch log API, then polls for the log to appear and returns the binaryContent.id.
func seedLogWithAttachment(
	ctx context.Context,
	c *client.Client,
	projectName string,
	seeded seededTestItems,
) (int64, error) {
	_, err := c.Project(projectName).Logs().CreateBatch(ctx, client.BatchLogEntry{
		LaunchUUID:     seeded.LaunchUUID,
		ItemUUID:       seeded.SuiteUUID,
		Level:          "INFO",
		Message:        "attachment test",
		Time:           time.Now(),
		AttachmentName: "test.txt",
		AttachmentData: []byte("integration test attachment content"),
	})
	if err != nil {
		return 0, fmt.Errorf("creating log with attachment: %w", err)
	}

	const (
		attachmentResolutionAttempts = 60
		attachmentResolutionInterval = 500 * time.Millisecond
	)

	var attachmentID int64

	resolved := waitForSeedResolution(
		ctx,
		attachmentResolutionAttempts,
		attachmentResolutionInterval,
		func() bool {
			result, listErr := c.Project(projectName).Logs().List(
				ctx,
				client.WithLogItemID(seeded.SuiteID),
				client.WithLogPageSize(50),
			)
			if listErr != nil || result == nil || result.Content == nil {
				return false
			}

			for _, entry := range *result.Content {
				if entry.BinaryContent == nil || entry.BinaryContent.Id == "" {
					continue
				}

				parsed, parseErr := strconv.ParseInt(entry.BinaryContent.Id, 10, 64)
				if parseErr != nil {
					continue
				}

				attachmentID = parsed

				return true
			}

			return false
		},
	)
	if !resolved {
		return 0, fmt.Errorf("attachment log entry did not become visible after polling")
	}

	return attachmentID, nil
}

// expectContent asserts that a paginated *[]T content field is non-nil and
// non-empty, reducing the verbose two-line pattern to a single call.
func expectContent[T any](content *[]T, description ...string) []T {
	GinkgoHelper()

	desc := "paginated content"
	if len(description) > 0 {
		desc = description[0]
	}

	Expect(content).NotTo(BeNil(), "%s should not be nil", desc)
	Expect(*content).NotTo(BeEmpty(), "%s should not be empty", desc)

	return *content
}

// expectLogLevel asserts that stderr contains log output at the given level.
func expectLogLevel(result cliResult, level string, description string) {
	GinkgoHelper()

	Expect(result.Stderr).To(ContainSubstring("level="+level),
		"%s should emit %s logs; stderr: %s", description, level, result.Stderr)
}

type seededLogEntries struct {
	LogIDs []int64
}

// seedTestLogEntries creates log entries for a seeded item/launch via the client
// library, then waits for the entries to become visible in listing queries.
func seedTestLogEntries(
	ctx context.Context,
	c *client.Client,
	projectName string,
	seeded seededTestItems,
	count int,
) (seededLogEntries, error) {
	entries := make([]client.BatchLogEntry, count)
	for i := range count {
		level := "INFO"
		if i%2 == 1 {
			level = "ERROR"
		}

		entries[i] = client.BatchLogEntry{
			LaunchUUID: seeded.LaunchUUID,
			ItemUUID:   seeded.SuiteUUID,
			Level:      level,
			Message:    fmt.Sprintf("integration-test-log-message-%d", i+1),
			Time:       time.Now(),
		}
	}

	_, err := c.Project(projectName).Logs().CreateBatch(ctx, entries...)
	if err != nil {
		return seededLogEntries{}, fmt.Errorf("creating log entries: %w", err)
	}

	var logIDs []int64

	resolved := waitForSeedResolution(
		ctx,
		seedResolutionAttempts,
		seedResolutionInterval,
		func() bool {
			result, listErr := c.Project(projectName).Logs().List(
				ctx,
				client.WithLogItemID(seeded.SuiteID),
				client.WithLogPageSize(int32(count+10)),
			)
			if listErr != nil || result == nil || result.Content == nil {
				return false
			}

			if len(*result.Content) < count {
				return false
			}

			logIDs = make([]int64, 0, len(*result.Content))
			for _, entry := range *result.Content {
				logIDs = append(logIDs, entry.Id)
			}

			return true
		},
	)
	if !resolved {
		return seededLogEntries{}, fmt.Errorf(
			"seeded log entries did not become visible after polling",
		)
	}

	return seededLogEntries{LogIDs: logIDs}, nil
}
