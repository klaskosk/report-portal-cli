package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// ProjectsScope provides operations on projects at the instance level.
type ProjectsScope struct {
	client *Client
}

// Get returns the full details of a project by name.
func (p *ProjectsScope) Get(
	ctx context.Context,
	projectName string,
) (*types.ProjectResource, error) {
	logger := p.client.logger.With(
		slog.String("operation", "GetProject"),
		slog.String("projectName", projectName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetProjectUsingGETWithResponse(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "get project", response.JSONDefault, response.JSON200)
}

// ListNames returns all project names in the Report Portal instance. The underlying
// endpoint (GetAllProjectNames) may return all projects regardless of the caller's
// access, depending on deployment configuration.
func (p *ProjectsScope) ListNames(ctx context.Context) ([]string, error) {
	logger := p.client.logger.With(slog.String("operation", "ListProjectNames"))
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetAllProjectNamesUsingGETWithResponse(ctx)
	if err != nil {
		logger.DebugContext(ctx, "typed project names decode failed; retrying raw", "error", err)

		return p.listNamesRaw(ctx)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(
		response,
		"list project names",
		response.JSONDefault,
		response.JSON200,
	)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return []string{}, nil
	}

	return result.Content, nil
}

// listNamesRaw retries the project-names call and decodes deployments that return a plain JSON string array.
func (p *ProjectsScope) listNamesRaw(ctx context.Context) ([]string, error) {
	rawClient, ok := p.client.raw.(*api.ClientWithResponses)
	if !ok {
		return nil, fmt.Errorf(
			"list project names: raw fallback unavailable for client type %T",
			p.client.raw,
		)
	}

	rawResponse, err := rawClient.GetAllProjectNamesUsingGET(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project names: %w", err)
	}
	defer rawResponse.Body.Close()

	body, err := io.ReadAll(rawResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("reading project names response: %w", err)
	}

	if rawResponse.StatusCode != http.StatusOK {
		var errorResponse types.ErrorRS
		if json.Unmarshal(body, &errorResponse) == nil && errorResponse.ErrorCode != 0 {
			return nil, newAPIError("list project names", rawResponse.StatusCode, &errorResponse)
		}

		return nil, newUnexpectedStatusError("list project names", rawResponse.StatusCode)
	}

	var names []string
	if err := json.Unmarshal(body, &names); err != nil {
		return nil, fmt.Errorf("list project names: %w", err)
	}

	return names, nil
}
