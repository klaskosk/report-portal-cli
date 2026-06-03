package cli //nolint:testpackage // white-box tests for unexported init helpers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const testExistingProfileConfig = `default-profile: existing
profiles:
  existing:
    url: https://existing.example.com
    token: old-token
`

func readConfigMap(path string) (map[string]any, error) {
	content, err := os.ReadFile(path) //nolint:gosec // test path from t.TempDir()
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}

		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var data map[string]any

	err = yaml.Unmarshal(content, &data)
	if err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if data == nil {
		data = map[string]any{}
	}

	return data, nil
}

func profilesMap(data map[string]any) map[string]any {
	raw, ok := data["profiles"]
	if !ok || raw == nil {
		return map[string]any{}
	}

	profiles, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	return profiles
}

func executeInit(t *testing.T, stdin io.Reader, args ...string) (string, string, error) {
	t.Helper()

	cmd := newRootCmd()
	cmd.SetIn(stdin)

	var outBuf, errBuf bytes.Buffer

	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)

	err := cmd.Execute()

	return outBuf.String(), errBuf.String(), err
}

func TestInit_FlagMode_WritesConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	stdout, _, err := executeInit(
		t,
		os.Stdin,
		"init",
		"--config", cfgPath,
		"--url", "https://reportportal.example.com",
		"--token", "secret-token",
		"--default-project", "my_project",
	)
	require.NoError(t, err)
	assert.Contains(t, stdout, cfgPath)
	assert.Contains(t, stdout, `profile "default"`)

	content, err := os.ReadFile(cfgPath) //nolint:gosec // test path from t.TempDir()
	require.NoError(t, err)

	var data map[string]any

	require.NoError(t, yaml.Unmarshal(content, &data))
	assert.Equal(t, "default", data["default-profile"])

	profiles := profilesMap(data)
	profile, ok := profiles["default"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://reportportal.example.com", profile["url"])
	assert.Equal(t, "secret-token", profile["token"])
	assert.Equal(t, "my_project", profile["default-project"])
}

func TestInit_FlagMode_CustomProfile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	_, _, err := executeInit(
		t,
		os.Stdin,
		"init",
		"--config", cfgPath,
		"--profile", "staging",
		"--url", "https://staging.example.com",
		"--token", "tok",
	)
	require.NoError(t, err)

	data, err := readConfigMap(cfgPath)
	require.NoError(t, err)

	profiles := profilesMap(data)
	require.Contains(t, profiles, "staging")
}

func TestInit_FlagMode_InsecureSkipVerify(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	_, _, err := executeInit(
		t,
		os.Stdin,
		"init",
		"--config", cfgPath,
		"--url", "https://reportportal.example.com",
		"--token", "tok",
		"--insecure-skip-verify",
	)
	require.NoError(t, err)

	data, err := readConfigMap(cfgPath)
	require.NoError(t, err)

	profiles := profilesMap(data)
	profile, ok := profiles["default"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, profile["insecure-skip-verify"])
}

func TestInit_FlagMode_RequiresBothURLAndToken(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	_, stderr, err := executeInit(
		t,
		os.Stdin,
		"init",
		"--config", cfgPath,
		"--url", "https://reportportal.example.com",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--url and --token must both be provided")
	assert.Contains(t, stderr, "--url and --token must both be provided")
}

func TestInit_NonInteractive_RequiresTTY(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, stdinW.Close())

	_, stderr, err := executeInit(
		t,
		stdinR,
		"init",
		"--config", cfgPath,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interactive mode requires a terminal")
	assert.Contains(t, stderr, "interactive mode requires a terminal")
}

func TestInit_InteractiveMode(t *testing.T) { //nolint:paralleltest // overrides package-level hooks
	oldTTY := stdinIsTerminal
	stdinIsTerminal = func(uintptr) bool { return true }

	t.Cleanup(func() { stdinIsTerminal = oldTTY })

	oldSecret := readSecretInput
	readSecretInput = func(out io.Writer, _ uintptr) ([]byte, error) {
		_, _ = fmt.Fprint(out, "API token: ")

		return []byte("secret-token\n"), nil
	}

	t.Cleanup(func() { readSecretInput = oldSecret })

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	input := strings.Join([]string{
		"dev\n",
		"https://reportportal.example.com\n",
		"my_project\n",
		"y\n",
	}, "")

	stdout, _, err := executeInit(
		t,
		strings.NewReader(input),
		"init",
		"--config", cfgPath,
	)
	require.NoError(t, err)
	assert.Contains(t, stdout, cfgPath)
	assert.Contains(t, stdout, `profile "dev"`)

	data, err := readConfigMap(cfgPath)
	require.NoError(t, err)

	profiles := profilesMap(data)
	profile, ok := profiles["dev"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://reportportal.example.com", profile["url"])
	assert.Equal(t, "secret-token", profile["token"])
	assert.Equal(t, "my_project", profile["default-project"])
	assert.Equal(t, true, profile["insecure-skip-verify"])
}

//nolint:paralleltest // overrides package-level hooks
func TestInit_InteractiveMode_UsesFlagDefaults(t *testing.T) {
	oldTTY := stdinIsTerminal
	stdinIsTerminal = func(uintptr) bool { return true }

	t.Cleanup(func() { stdinIsTerminal = oldTTY })

	oldSecret := readSecretInput
	readSecretInput = func(out io.Writer, _ uintptr) ([]byte, error) {
		_, _ = fmt.Fprint(out, "API token: ")

		return []byte("secret-token\n"), nil
	}

	t.Cleanup(func() { readSecretInput = oldSecret })

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	input := strings.Join([]string{
		"\n",
		"https://reportportal.example.com\n",
		"\n",
		"n\n",
	}, "")

	stdout, _, err := executeInit(
		t,
		strings.NewReader(input),
		"init",
		"--config", cfgPath,
		"--profile", "staging",
		"--default-project", "my_project",
	)
	require.NoError(t, err)
	assert.Contains(t, stdout, `profile "staging"`)

	data, err := readConfigMap(cfgPath)
	require.NoError(t, err)

	profiles := profilesMap(data)
	profile, ok := profiles["staging"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "my_project", profile["default-project"])
}

func TestInit_FailsIfConfigExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	require.NoError(t, os.WriteFile(cfgPath, []byte(testExistingProfileConfig), 0o600))

	_, stderr, err := executeInit(
		t,
		os.Stdin,
		"init",
		"--config", cfgPath,
		"--url", "https://new.example.com",
		"--token", "new-token",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config file already exists")
	assert.Contains(t, stderr, "config file already exists")
	assert.Contains(t, err.Error(), "--force")

	content, readErr := os.ReadFile(cfgPath) //nolint:gosec // test path from t.TempDir()
	require.NoError(t, readErr)
	assert.Equal(t, testExistingProfileConfig, string(content))
}

//nolint:paralleltest // overrides package-level hooks
func TestInit_InteractiveFailsIfConfigExists(t *testing.T) {
	oldTTY := stdinIsTerminal
	stdinIsTerminal = func(uintptr) bool { return true }

	t.Cleanup(func() { stdinIsTerminal = oldTTY })

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	require.NoError(t, os.WriteFile(cfgPath, []byte(testExistingProfileConfig), 0o600))

	_, stderr, err := executeInit(
		t,
		strings.NewReader("should-not-be-read\n"),
		"init",
		"--config", cfgPath,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config file already exists")
	assert.Contains(t, stderr, "config file already exists")

	content, readErr := os.ReadFile(cfgPath) //nolint:gosec // test path from t.TempDir()
	require.NoError(t, readErr)
	assert.Equal(t, testExistingProfileConfig, string(content))
}

func TestInit_ForceOverwritesWhenComplete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	existing := `default-profile: old
profiles:
  old:
    url: https://old.example.com
    token: old-token
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(existing), 0o600))

	_, _, err := executeInit(
		t,
		os.Stdin,
		"init",
		"--config", cfgPath,
		"--force",
		"--url", "https://new.example.com",
		"--token", "new-token",
	)
	require.NoError(t, err)

	data, err := readConfigMap(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "default", data["default-profile"])

	profiles := profilesMap(data)
	profile, ok := profiles["default"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://new.example.com", profile["url"])
	assert.Equal(t, "new-token", profile["token"])
}

func TestInit_ForcePreservesConfigOnPartialFlags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	require.NoError(t, os.WriteFile(cfgPath, []byte(testExistingProfileConfig), 0o600))

	_, _, err := executeInit(
		t,
		os.Stdin,
		"init",
		"--config", cfgPath,
		"--force",
		"--url", "https://new.example.com",
	)
	require.Error(t, err)

	content, readErr := os.ReadFile(cfgPath) //nolint:gosec // test path from t.TempDir()
	require.NoError(t, readErr)
	assert.Equal(t, testExistingProfileConfig, string(content))
}

//nolint:paralleltest // overrides package-level hooks
func TestInit_ForcePreservesConfigOnInterruptedInteractive(t *testing.T) {
	oldTTY := stdinIsTerminal
	stdinIsTerminal = func(uintptr) bool { return true }

	t.Cleanup(func() { stdinIsTerminal = oldTTY })

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	require.NoError(t, os.WriteFile(cfgPath, []byte(testExistingProfileConfig), 0o600))

	_, _, err := executeInit(
		t,
		strings.NewReader("dev\n"),
		"init",
		"--config", cfgPath,
		"--force",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input interrupted")

	content, readErr := os.ReadFile(cfgPath) //nolint:gosec // test path from t.TempDir()
	require.NoError(t, readErr)
	assert.Equal(t, testExistingProfileConfig, string(content))
}

//nolint:paralleltest // overrides package-level hooks
func TestInit_ForceInteractiveOverwritesWhenComplete(t *testing.T) {
	oldTTY := stdinIsTerminal
	stdinIsTerminal = func(uintptr) bool { return true }

	t.Cleanup(func() { stdinIsTerminal = oldTTY })

	oldSecret := readSecretInput
	readSecretInput = func(out io.Writer, _ uintptr) ([]byte, error) {
		_, _ = fmt.Fprint(out, "API token: ")

		return []byte("new-token\n"), nil
	}

	t.Cleanup(func() { readSecretInput = oldSecret })

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	existing := `default-profile: old
profiles:
  old:
    url: https://old.example.com
    token: old-token
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(existing), 0o600))

	input := strings.Join([]string{
		"staging\n",
		"https://new.example.com\n",
		"my_project\n",
		"n\n",
	}, "")

	_, _, err := executeInit(
		t,
		strings.NewReader(input),
		"init",
		"--config", cfgPath,
		"--force",
	)
	require.NoError(t, err)

	data, err := readConfigMap(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "staging", data["default-profile"])

	profiles := profilesMap(data)
	profile, ok := profiles["staging"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://new.example.com", profile["url"])
	assert.Equal(t, "new-token", profile["token"])
}

func TestInit_JSONOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	stdout, _, err := executeInit(
		t,
		os.Stdin,
		"init",
		"--config", cfgPath,
		"--output", "json",
		"--url", "https://reportportal.example.com",
		"--token", "tok",
	)
	require.NoError(t, err)

	var decoded map[string]string
	require.NoError(t, json.Unmarshal([]byte(stdout), &decoded))
	assert.Contains(t, decoded["message"], cfgPath)
	assert.Contains(t, decoded["message"], "default")
}

func TestInit_ConfigFilePermissions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "rp")
	cfgPath := filepath.Join(cfgDir, "config.yaml")

	_, _, err := executeInit(
		t,
		os.Stdin,
		"init",
		"--config", cfgPath,
		"--url", "https://reportportal.example.com",
		"--token", "tok",
	)
	require.NoError(t, err)

	info, err := os.Stat(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	dirInfo, err := os.Stat(cfgDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o750), dirInfo.Mode().Perm())
}

func TestPromptString_UsesDefault(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	got, err := promptString(bufio.NewReader(strings.NewReader("\n")), &out, "Label", "default-val")
	require.NoError(t, err)
	assert.Equal(t, "default-val", got)
}

func TestPromptBool(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	yes, err := promptBool(bufio.NewReader(strings.NewReader("y\n")), &out, "Q", false)
	require.NoError(t, err)
	assert.True(t, yes)

	no, err := promptBool(bufio.NewReader(strings.NewReader("n\n")), &out, "Q", true)
	require.NoError(t, err)
	assert.False(t, no)

	def, err := promptBool(bufio.NewReader(strings.NewReader("\n")), &out, "Q", true)
	require.NoError(t, err)
	assert.True(t, def)
}
