// Package client provides a high-level client for the Report Portal API.
package client

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
)

// Client is a high-level wrapper around the generated Report Portal API client.
type Client struct {
	raw    api.ClientWithResponsesInterface
	logger *slog.Logger
}

// Option configures the Client during construction.
type Option func(*config) error

type config struct {
	httpClient api.HttpRequestDoer
	logger     *slog.Logger
}

// New creates a new Client for the given Report Portal instance.
// The bearerToken is sent as an Authorization header on every request.
func New(baseURL, bearerToken string, opts ...Option) (*Client, error) {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	baseURL = strings.TrimSuffix(baseURL, "/")
	if !strings.HasSuffix(baseURL, "/api") {
		baseURL += "/api"
	}

	config := &config{}
	for _, option := range opts {
		err := option(config)
		if err != nil {
			return nil, err
		}
	}

	apiOpts := []api.ClientOption{
		api.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+bearerToken)

			return nil
		}),
	}
	if config.httpClient != nil {
		apiOpts = append(apiOpts, api.WithHTTPClient(config.httpClient))
	}

	raw, err := api.NewClientWithResponses(baseURL, apiOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	logger := config.logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	return &Client{raw: raw, logger: logger}, nil
}

// WithHTTPClient overrides the default HTTP client used for requests.
func WithHTTPClient(doer api.HttpRequestDoer) Option {
	return func(config *config) error {
		config.httpClient = doer

		return nil
	}
}

// Users returns a UsersScope for instance-level user operations.
func (c *Client) Users() *UsersScope {
	return &UsersScope{client: c}
}

// Projects returns a ProjectsScope for instance-level project operations.
func (c *Client) Projects() *ProjectsScope {
	return &ProjectsScope{client: c}
}

// Integrations returns an IntegrationsScope for instance-level global integration operations.
func (c *Client) Integrations() *IntegrationsScope {
	return &IntegrationsScope{client: c}
}

// WithLogger configures the logger used by the client for structured diagnostic
// output. If not set, the client defaults to discarding all log output.
func WithLogger(logger *slog.Logger) Option {
	return func(config *config) error {
		config.logger = logger

		return nil
	}
}
