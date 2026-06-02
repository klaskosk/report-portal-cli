package core

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type viperContextKey struct{}

// ErrNoActiveProfile is returned when no profile can be resolved from flags or config.
var ErrNoActiveProfile = errors.New("no active profile")

// Profile holds connection details for a single Report Portal instance.
type Profile struct {
	URL                string `mapstructure:"url"`
	Token              string `mapstructure:"token"`
	DefaultProject     string `mapstructure:"default-project"`
	InsecureSkipVerify bool   `mapstructure:"insecure-skip-verify"`
}

// Config is the top-level CLI configuration, typically loaded from ~/.config/rp/config.yaml.
type Config struct {
	DefaultProfile string             `mapstructure:"default-profile"`
	Profiles       map[string]Profile `mapstructure:"profiles"`
}

// Connection holds the fully resolved parameters needed to create an API client.
type Connection struct {
	URL                string
	Token              string
	Project            string
	InsecureSkipVerify bool
}

// InitConfig creates a per-invocation viper instance and stores it on cmd's context.
func InitConfig(cmd *cobra.Command) error {
	viperInst := viper.New()

	cfgFile, _ := cmd.Root().Flags().GetString("config")

	if cfgFile != "" {
		viperInst.SetConfigFile(cfgFile)
	} else {
		cfgDir, err := configDir()
		if err != nil {
			return err
		}

		viperInst.AddConfigPath(cfgDir)
		viperInst.SetConfigName("config")
		viperInst.SetConfigType("yaml")
	}

	viperInst.SetEnvPrefix("RP")
	viperInst.AutomaticEnv()

	logger := LoggerFromCmd(cmd)

	err := viperInst.ReadInConfig()
	if err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return fmt.Errorf("reading config: %w", err)
		}

		logger.Debug("no config file found")
	} else {
		logger.Info("using config file", "path", viperInst.ConfigFileUsed())
	}

	cmd.SetContext(context.WithValue(cmd.Context(), viperContextKey{}, viperInst))

	return nil
}

func viperFromCmd(cmd *cobra.Command) *viper.Viper {
	v, _ := cmd.Context().Value(viperContextKey{}).(*viper.Viper)

	return v
}

// configDir returns the XDG-compatible configuration directory for rp (~/.config/rp).
func configDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "rp"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}

	return filepath.Join(home, ".config", "rp"), nil
}

// ResolveConnection merges CLI flags, env vars, and the active profile into a single Connection.
func ResolveConnection(cmd *cobra.Command) (*Connection, error) {
	logger := LoggerFromCmd(cmd)
	v := viperFromCmd(cmd)

	var config Config

	err := v.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	profile, err := activeProfile(cmd, &config)
	if err != nil && !errors.Is(err, ErrNoActiveProfile) {
		return nil, err
	}

	connection := connectionFromProfile(profile)
	applyEnvOverrides(connection)
	applyFlagOverrides(cmd, connection)

	if connection.URL == "" {
		return nil, errors.New(
			"no Report Portal URL configured (use --url, RP_URL env var, or a config profile)",
		)
	}

	if connection.Token == "" {
		return nil, errors.New(
			"no API token configured (use --token, RP_TOKEN env var, or a config profile)",
		)
	}

	logger.Info("resolved connection",
		"url", connection.URL,
		"insecureSkipVerify", connection.InsecureSkipVerify,
	)

	return connection, nil
}

func connectionFromProfile(profile *Profile) *Connection {
	connection := &Connection{}
	if profile != nil {
		connection.URL = profile.URL
		connection.Token = profile.Token
		connection.Project = profile.DefaultProject
		connection.InsecureSkipVerify = profile.InsecureSkipVerify
	}

	return connection
}

func applyEnvOverrides(connection *Connection) {
	if value, ok := os.LookupEnv("RP_URL"); ok && value != "" {
		connection.URL = value
	}

	if value, ok := os.LookupEnv("RP_TOKEN"); ok && value != "" {
		connection.Token = value
	}

	if value, ok := os.LookupEnv("RP_PROJECT"); ok && value != "" {
		connection.Project = value
	}

	if value, ok := os.LookupEnv("RP_INSECURE_SKIP_VERIFY"); ok {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			connection.InsecureSkipVerify = parsed
		}
	}
}

func applyFlagOverrides(cmd *cobra.Command, connection *Connection) {
	root := cmd.Root()

	if root.Flags().Changed("url") {
		connection.URL, _ = root.Flags().GetString("url")
	}

	if root.Flags().Changed("token") {
		connection.Token, _ = root.Flags().GetString("token")
	}

	if root.Flags().Changed("insecure-skip-verify") {
		connection.InsecureSkipVerify, _ = root.Flags().GetBool("insecure-skip-verify")
	}
}

func resolveProject(cmd *cobra.Command, connection *Connection) (string, error) {
	logger := LoggerFromCmd(cmd)

	if cmd.Flags().Changed("project") {
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			return "", errors.New("project name cannot be empty (omit -p to use the default)")
		}

		logger.Info("using project", "project", project, "source", "flag")

		return project, nil
	}

	if connection.Project != "" {
		logger.Info("using project", "project", connection.Project, "source", "profile")

		return connection.Project, nil
	}

	return "", errors.New(
		"no project specified (use -p flag, RP_PROJECT env var, or set default-project in your profile)",
	)
}

// NewClient builds a *client.Client from a resolved Connection.
func NewClient(connection *Connection, logger *slog.Logger) (*client.Client, error) {
	var options []client.Option

	options = append(options, client.WithLogger(logger))

	if connection.InsecureSkipVerify {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true, //nolint:gosec // user-requested
			},
		}

		options = append(options, client.WithHTTPClient(&http.Client{Transport: transport}))
	}

	clientInstance, err := client.New(connection.URL, connection.Token, options...)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}

	logger.Debug("created API client", "url", connection.URL)

	return clientInstance, nil
}

func activeProfile(cmd *cobra.Command, config *Config) (*Profile, error) {
	logger := LoggerFromCmd(cmd)
	name := config.DefaultProfile

	if cmd.Root().Flags().Changed("profile") {
		name, _ = cmd.Root().Flags().GetString("profile")
		logger.Debug("profile overridden by flag", "profile", name)
	}

	if name == "" {
		return nil, ErrNoActiveProfile
	}

	profile, ok := config.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in config", name)
	}

	logger.Info("using profile", "profile", name)

	return &profile, nil
}
