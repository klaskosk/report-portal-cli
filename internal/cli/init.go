package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

const defaultInitProfile = "default"

// stdinIsTerminal reports whether fd is an interactive terminal. Tests may override.
var stdinIsTerminal = func(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}

// readSecretInput reads a secret value without echo. Tests may override.
var readSecretInput = func(out io.Writer, fd uintptr) ([]byte, error) {
	_, _ = fmt.Fprint(out, "API token: ")

	return term.ReadPassword(int(fd))
}

type initProfile struct {
	name               string
	url                string
	token              string
	defaultProject     string
	insecureSkipVerify bool
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a Report Portal CLI config file",
		Long: `Create a new Report Portal CLI configuration file with an initial profile.

Fails if the config file already exists unless --force is set. With --force,
the file is replaced only after all required values are collected; partial
flag input or an interrupted interactive session leaves the existing file
unchanged.

Provide --url and --token for non-interactive setup. Without those flags, an
interactive wizard runs and requires a terminal on stdin.

In interactive mode, --profile and --default-project pre-fill the corresponding
prompts (press Enter to accept). The --config and --insecure-skip-verify flags are
also respected; when --insecure-skip-verify is set, the TLS prompt is skipped.`,
		Annotations: map[string]string{core.GroupAnnotationKey: "true"},
		RunE: core.WithPrintedErrors(func(cmd *cobra.Command, _ []string) error {
			printer := core.NewPrinterFromCmd(cmd)

			cfgPath := initConfigPath(cmd)
			force, _ := cmd.Flags().GetBool("force")

			err := checkInitConfigPath(cfgPath, force)
			if err != nil {
				return core.PrintAndReturnError(printer, err)
			}

			profile, err := resolveInitProfile(cmd)
			if err != nil {
				return core.PrintAndReturnError(printer, err)
			}

			err = writeInitConfig(cfgPath, profile)
			if err != nil {
				return core.PrintAndReturnError(printer, err)
			}

			err = printer.PrintMessage(
				fmt.Sprintf("Wrote config to %s (profile %q)", cfgPath, profile.name),
			)
			if err != nil {
				return core.PrintAndReturnError(printer, err)
			}

			return nil
		}),
	}

	cmd.Flags().String("default-project", "", "default Report Portal project for this profile")
	cmd.Flags().
		Bool("force", false, "overwrite an existing config file after all values are collected")

	return cmd
}

func initStdinFd(cmd *cobra.Command) uintptr {
	stdin := cmd.InOrStdin()

	type fdReader interface {
		Fd() uintptr
	}

	if f, ok := stdin.(fdReader); ok {
		return f.Fd()
	}

	return os.Stdin.Fd()
}

func initConfigPath(cmd *cobra.Command) string {
	cfgPath, _ := cmd.Root().Flags().GetString("config")
	if cfgPath == "" {
		return core.DefaultConfigPath()
	}

	return cfgPath
}

func resolveInitProfile(cmd *cobra.Command) (initProfile, error) {
	root := cmd.Root()
	urlChanged := root.Flags().Changed("url")
	tokenChanged := root.Flags().Changed("token")

	switch {
	case urlChanged && tokenChanged:
		return collectInitFromFlags(cmd)
	case !urlChanged && !tokenChanged:
		if !stdinIsTerminal(initStdinFd(cmd)) {
			return initProfile{}, errors.New(
				"interactive mode requires a terminal; provide --url and --token flags",
			)
		}

		return collectInitInteractive(cmd)
	default:
		return initProfile{}, errors.New(
			"--url and --token must both be provided for non-interactive mode",
		)
	}
}

func collectInitFromFlags(cmd *cobra.Command) (initProfile, error) {
	root := cmd.Root()
	url, _ := root.Flags().GetString("url")
	token, _ := root.Flags().GetString("token")

	profile := initProfile{
		name:               initProfileName(cmd),
		url:                strings.TrimSpace(url),
		token:              strings.TrimSpace(token),
		defaultProject:     strings.TrimSpace(mustStringFlag(cmd, "default-project")),
		insecureSkipVerify: initInsecureFromFlags(cmd),
	}

	if profile.url == "" {
		return initProfile{}, errors.New("--url is required")
	}

	if profile.token == "" {
		return initProfile{}, errors.New("--token is required")
	}

	return profile, nil
}

func collectInitInteractive(cmd *cobra.Command) (initProfile, error) {
	reader := bufio.NewReader(cmd.InOrStdin())
	out := cmd.OutOrStdout()

	name, err := promptString(reader, out, "Profile name", initProfileName(cmd))
	if err != nil {
		return initProfile{}, err
	}

	url, err := promptString(reader, out, "Report Portal base URL", "")
	if err != nil {
		return initProfile{}, err
	}

	if url == "" {
		return initProfile{}, errors.New("report portal base URL is required")
	}

	token, err := promptSecret(cmd)
	if err != nil {
		return initProfile{}, fmt.Errorf("reading API token: %w", err)
	}

	if token == "" {
		return initProfile{}, errors.New("API token is required")
	}

	defaultProject, err := promptString(
		reader,
		out,
		"Default project (optional)",
		strings.TrimSpace(mustStringFlag(cmd, "default-project")),
	)
	if err != nil {
		return initProfile{}, err
	}

	insecure := initInsecureFromFlags(cmd)
	if !cmd.Root().Flags().Changed("insecure-skip-verify") {
		var promptErr error

		insecure, promptErr = promptBool(reader, out, "Skip TLS certificate verification?", false)
		if promptErr != nil {
			return initProfile{}, promptErr
		}
	}

	return initProfile{
		name:               name,
		url:                url,
		token:              token,
		defaultProject:     defaultProject,
		insecureSkipVerify: insecure,
	}, nil
}

func initProfileName(cmd *cobra.Command) string {
	name, _ := cmd.Root().Flags().GetString("profile")

	name = strings.TrimSpace(name)
	if name == "" {
		return defaultInitProfile
	}

	return name
}

func initInsecureFromFlags(cmd *cobra.Command) bool {
	if !cmd.Root().Flags().Changed("insecure-skip-verify") {
		return false
	}

	insecure, _ := cmd.Root().Flags().GetBool("insecure-skip-verify")

	return insecure
}

func mustStringFlag(cmd *cobra.Command, name string) string {
	val, _ := cmd.Flags().GetString(name)

	return val
}

func checkInitConfigPath(path string, force bool) error {
	exists, err := configFileExists(path)
	if err != nil {
		return fmt.Errorf("checking config file: %w", err)
	}

	if exists && !force {
		return fmt.Errorf(
			"config file already exists at %s; edit it manually, remove it, or re-run with --force",
			path,
		)
	}

	return nil
}

func configFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, fmt.Errorf("stating config file: %w", err)
}

func promptString(reader *bufio.Reader, out io.Writer, label, defaultVal string) (string, error) {
	if defaultVal != "" {
		_, _ = fmt.Fprintf(out, "%s [%s]: ", label, defaultVal)
	} else {
		_, _ = fmt.Fprintf(out, "%s: ", label)
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line), nil
		}

		return "", errors.New("input interrupted")
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}

	return line, nil
}

func promptSecret(cmd *cobra.Command) (string, error) {
	secret, err := readSecretInput(cmd.OutOrStdout(), initStdinFd(cmd))
	if err != nil {
		return "", err
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout())

	return strings.TrimSpace(string(secret)), nil
}

func promptBool(reader *bufio.Reader, out io.Writer, label string, defaultVal bool) (bool, error) {
	defaultHint := "y/N"
	if defaultVal {
		defaultHint = "Y/n"
	}

	_, _ = fmt.Fprintf(out, "%s [%s]: ", label, defaultHint)

	line, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && strings.TrimSpace(line) == "" {
			return defaultVal, errors.New("input interrupted")
		}

		if errors.Is(err, io.EOF) {
			line = strings.TrimSpace(strings.ToLower(line))

			return line == "y" || line == "yes", nil
		}

		return defaultVal, errors.New("input interrupted")
	}

	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultVal, nil
	}

	return line == "y" || line == "yes", nil
}

func writeInitConfig(path string, profile initProfile) error {
	profileData := map[string]any{
		"url":   profile.url,
		"token": profile.token,
	}
	if profile.defaultProject != "" {
		profileData["default-project"] = profile.defaultProject
	}

	if profile.insecureSkipVerify {
		profileData["insecure-skip-verify"] = true
	}

	data := map[string]any{
		"default-profile": profile.name,
		"profiles": map[string]any{
			profile.name: profileData,
		},
	}

	return writeConfigMap(path, data)
}

func writeConfigMap(path string, data map[string]any) error {
	dir := filepath.Dir(path)

	err := os.MkdirAll(dir, 0o750)
	if err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	encoded, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("encoding config file: %w", err)
	}

	err = os.WriteFile(path, encoded, 0o600)
	if err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
