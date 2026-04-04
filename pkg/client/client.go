// Package client provides a high-level client for the Report Portal API.
package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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
	httpClient *http.Client
	logger     *slog.Logger
}

// New creates a new Client for the given Report Portal instance. It is tolerant of many baseURLs and will accept with
// or without a scheme and ensures it ends in /api. The bearerToken must be non-empty and is sent as an Authorization
// header on every request.
func New(baseURL, bearerToken string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(bearerToken) == "" {
		return nil, errors.New("failed to create client: bearer token must not be empty")
	}

	normalizedBaseURL, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	config := &config{}
	for _, option := range opts {
		err = option(config)
		if err != nil {
			return nil, err
		}
	}

	httpClient := config.httpClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	transport := httpClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	wrappedClient := &http.Client{
		Transport:     &emptyJSONBodyNormalizer{next: transport},
		CheckRedirect: httpClient.CheckRedirect,
		Jar:           httpClient.Jar,
		Timeout:       httpClient.Timeout,
	}

	apiOpts := []api.ClientOption{
		api.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+bearerToken)

			return nil
		}),
		api.WithHTTPClient(wrappedClient),
	}

	raw, err := api.NewClientWithResponses(normalizedBaseURL, apiOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	logger := config.logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	return &Client{raw: raw, logger: logger}, nil
}

// normalizeBaseURL parses a user-provided Report Portal URL and returns a normalized API base URL with an explicit
// scheme and /api path prefix. The default scheme is https.
func normalizeBaseURL(baseURL string) (string, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil || (parsedBaseURL.Scheme != "http" && parsedBaseURL.Scheme != "https") {
		parsedBaseURL, err = url.Parse("https://" + strings.TrimPrefix(baseURL, "//"))
		if err != nil {
			return "", fmt.Errorf("parse base URL: %w", err)
		}
	}

	// The API base must be path-only; query/fragment create brittle request URLs.
	parsedBaseURL.RawQuery = ""
	parsedBaseURL.Fragment = ""

	normalizedPath := strings.TrimRight(parsedBaseURL.Path, "/")
	if normalizedPath == "" {
		parsedBaseURL.Path = "/"
	} else {
		parsedBaseURL.Path = normalizedPath + "/"
	}

	if strings.HasSuffix(normalizedPath, "/api") {
		parsedBaseURL.Path = normalizedPath
	} else {
		parsedBaseURL = parsedBaseURL.ResolveReference(&url.URL{Path: "api"})
	}

	return parsedBaseURL.String(), nil
}

// WithHTTPClient overrides the default HTTP client used for requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(config *config) error {
		config.httpClient = httpClient

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

// Settings returns a SettingsScope for server-level settings operations.
func (c *Client) Settings() *SettingsScope {
	return &SettingsScope{client: c}
}

// Plugins returns a PluginScope for instance-level plugin operations.
func (c *Client) Plugins() *PluginScope {
	return &PluginScope{client: c}
}

// Onboarding returns an OnboardingScope for first-run onboarding operations.
func (c *Client) Onboarding() *OnboardingScope {
	return &OnboardingScope{client: c}
}

// ActivityEvents returns an ActivityEventsScope for instance-level activity event queries.
func (c *Client) ActivityEvents() *ActivityEventsScope {
	return &ActivityEventsScope{client: c}
}

// WithLogger configures the logger used by the client for structured diagnostic
// output. If not set, the client defaults to discarding all log output.
func WithLogger(logger *slog.Logger) Option {
	return func(config *config) error {
		config.logger = logger

		return nil
	}
}

// emptyJSONBodyNormalizer is an http.RoundTripper that replaces empty response
// bodies on 2xx JSON responses with the JSON literal "null". Some Report Portal
// endpoints may return 200 with Content-Type: application/json and an empty
// body; without this normalizer the generated parser would fail with
// "unexpected end of JSON input" because json.Unmarshal rejects empty input.
type emptyJSONBodyNormalizer struct {
	next http.RoundTripper
}

func (n *emptyJSONBodyNormalizer) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := n.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 &&
		resp.ContentLength == 0 &&
		strings.Contains(resp.Header.Get("Content-Type"), "json") {
		resp.Body = io.NopCloser(strings.NewReader("null"))
		resp.ContentLength = 4
	}

	return resp, nil
}
