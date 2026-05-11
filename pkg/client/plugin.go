package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/textproto"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// PluginScope provides operations on installed plugins at the instance level.
// Most methods require admin privileges.
type PluginScope struct {
	client *Client
}

// List returns all installed plugins. Requires admin privileges.
func (s *PluginScope) List(ctx context.Context) ([]types.IntegrationTypeResource, error) {
	logger := s.client.logger.With(slog.String("operation", "ListPlugins"))
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetPluginsWithResponse(ctx)

	result, err := execute(ctx, logger, "list plugins", response, err)
	if err != nil {
		return nil, err
	}

	return *result, nil
}

// ListPublic returns all plugins via the public endpoint. Some deployments
// allow this endpoint without authentication.
func (s *PluginScope) ListPublic(ctx context.Context) ([]types.IntegrationTypeResource, error) {
	logger := s.client.logger.With(slog.String("operation", "ListPublicPlugins"))
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetPlugins1WithResponse(ctx)

	result, err := execute(ctx, logger, "list public plugins", response, err)
	if err != nil {
		return nil, err
	}

	return *result, nil
}

// UpdateState enables or disables a plugin. Requires admin privileges.
func (s *PluginScope) UpdateState(
	ctx context.Context,
	pluginID int64,
	stateUpdate types.UpdatePluginStateRQ,
) (*types.OperationCompletionRS, error) {
	logger := s.client.logger.With(
		slog.String("operation", "UpdatePluginState"),
		slog.Int64("pluginId", pluginID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.UpdatePluginStateWithResponse(ctx, pluginID, stateUpdate)

	return execute(ctx, logger, "update plugin state", response, err)
}

// Delete removes a plugin. Requires admin privileges.
func (s *PluginScope) Delete(
	ctx context.Context,
	pluginID int64,
) (*types.OperationCompletionRS, error) {
	logger := s.client.logger.With(
		slog.String("operation", "DeletePlugin"),
		slog.Int64("pluginId", pluginID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.DeletePluginWithResponse(ctx, pluginID)

	return execute(ctx, logger, "delete plugin", response, err)
}

// Upload uploads a plugin archive. Requires admin privileges.
// The filename is used for the multipart form field and data provides the file
// contents.
func (s *PluginScope) Upload(
	ctx context.Context,
	filename string,
	data io.Reader,
) (*types.EntryCreatedRS, error) {
	logger := s.client.logger.With(
		slog.String("operation", "UploadPlugin"),
		slog.String("filename", filename),
	)
	logger.InfoContext(ctx, "API request")

	body, contentType, err := buildPluginUploadMultipart(filename, data)
	if err != nil {
		return nil, fmt.Errorf("upload plugin: %w", err)
	}

	response, err := s.client.raw.UploadPluginWithBodyWithResponse(ctx, contentType, body)

	return execute(ctx, logger, "upload plugin", response, err)
}

// ExecuteCommand executes a named command on a plugin within a project.
// The body is an arbitrary JSON object whose schema depends on the plugin
// and command. The projectName parameter identifies the project context.
func (s *PluginScope) ExecuteCommand(
	ctx context.Context,
	projectName, pluginName, command string,
	body map[string]any,
) (map[string]any, error) {
	logger := s.client.logger.With(
		slog.String("operation", "ExecutePluginCommand"),
		slog.String("project", projectName),
		slog.String("pluginName", pluginName),
		slog.String("command", command),
	)
	logger.InfoContext(ctx, "API request")

	if body == nil {
		body = map[string]any{}
	}

	response, err := s.client.raw.ExecutePluginCommandWithResponse(
		ctx, projectName, pluginName, command, body,
	)

	result, err := execute(ctx, logger, "execute plugin command", response, err)
	if err != nil {
		return nil, err
	}

	if *result == nil {
		return map[string]any{}, nil
	}

	return *result, nil
}

// Import imports data through a plugin within a project using a multipart
// upload. The file part is required; launchImportRQ is optional metadata
// for the import (pass nil to omit). The projectName parameter identifies
// the project context.
func (s *PluginScope) Import(
	ctx context.Context,
	projectName, pluginName string,
	filename string,
	data io.Reader,
	launchImportRQ *types.LaunchImportRQ,
) (map[string]any, error) {
	logger := s.client.logger.With(
		slog.String("operation", "ImportPlugin"),
		slog.String("project", projectName),
		slog.String("pluginName", pluginName),
		slog.String("filename", filename),
	)
	logger.InfoContext(ctx, "API request")

	body, contentType, err := buildPluginImportMultipart(filename, data, launchImportRQ)
	if err != nil {
		return nil, fmt.Errorf("import plugin: %w", err)
	}

	response, err := s.client.raw.ExecuteImportPluginCommandWithBodyWithResponse(
		ctx, projectName, pluginName, contentType, body,
	)

	result, err := execute(ctx, logger, "import plugin", response, err)
	if err != nil {
		return nil, err
	}

	if *result == nil {
		return map[string]any{}, nil
	}

	return *result, nil
}

// ExecutePublicCommand executes a named command on a plugin via the public
// endpoint. The body is an arbitrary JSON object whose schema depends on the
// plugin and command. Some deployments allow this without authentication.
func (s *PluginScope) ExecutePublicCommand(
	ctx context.Context,
	pluginName, command string,
	body map[string]any,
) (map[string]any, error) {
	logger := s.client.logger.With(
		slog.String("operation", "ExecutePublicPluginCommand"),
		slog.String("pluginName", pluginName),
		slog.String("command", command),
	)
	logger.InfoContext(ctx, "API request")

	if body == nil {
		body = map[string]any{}
	}

	response, err := s.client.raw.ExecutePublicPluginCommandWithResponse(
		ctx, pluginName, command, body,
	)

	result, err := execute(ctx, logger, "execute public plugin command", response, err)
	if err != nil {
		return nil, err
	}

	if *result == nil {
		return map[string]any{}, nil
	}

	return *result, nil
}

// GetPublicFile downloads a public file from a plugin (e.g., plugin icon).
// Returns the raw file bytes. Some deployments allow this without authentication.
func (s *PluginScope) GetPublicFile(
	ctx context.Context,
	pluginName, fileName string,
) ([]byte, error) {
	logger := s.client.logger.With(
		slog.String("operation", "GetPublicPluginFile"),
		slog.String("pluginName", pluginName),
		slog.String("fileName", fileName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetPublicFileWithResponse(ctx, pluginName, fileName)
	if err != nil {
		return nil, fmt.Errorf("get public plugin file: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.JSONDefault != nil {
		return nil, fmt.Errorf(
			"get public plugin file: %w",
			newAPIError(response.StatusCode(), response.JSONDefault),
		)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf(
			"get public plugin file: %w",
			newUnexpectedStatusError(code, response.Body),
		)
	}

	return response.Body, nil
}

// buildPluginUploadMultipart assembles a multipart body containing a single
// "file" form field for the plugin upload endpoint.
func buildPluginUploadMultipart(filename string, data io.Reader) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)

	filePart, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", fmt.Errorf("creating file part: %w", err)
	}

	_, copyErr := io.Copy(filePart, data)
	if copyErr != nil {
		return nil, "", fmt.Errorf("writing file part: %w", copyErr)
	}

	closeErr := writer.Close()
	if closeErr != nil {
		return nil, "", fmt.Errorf("closing multipart writer: %w", closeErr)
	}

	return &buf, writer.FormDataContentType(), nil
}

// buildPluginImportMultipart assembles a multipart body for the plugin import
// endpoint. It includes a required "file" part and an optional "launchImportRq"
// JSON part.
func buildPluginImportMultipart(
	filename string,
	data io.Reader,
	launchImportRQ *types.LaunchImportRQ,
) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)

	filePart, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", fmt.Errorf("creating file part: %w", err)
	}

	_, copyErr := io.Copy(filePart, data)
	if copyErr != nil {
		return nil, "", fmt.Errorf("writing file part: %w", copyErr)
	}

	if launchImportRQ != nil {
		jsonHeader := make(textproto.MIMEHeader)
		jsonHeader.Set("Content-Disposition", `form-data; name="launchImportRq"`)
		jsonHeader.Set("Content-Type", "application/json")

		jsonPart, err := writer.CreatePart(jsonHeader)
		if err != nil {
			return nil, "", fmt.Errorf("creating launch import part: %w", err)
		}

		encodeErr := json.NewEncoder(jsonPart).Encode(launchImportRQ)
		if encodeErr != nil {
			return nil, "", fmt.Errorf("writing launch import part: %w", encodeErr)
		}
	}

	closeErr := writer.Close()
	if closeErr != nil {
		return nil, "", fmt.Errorf("closing multipart writer: %w", closeErr)
	}

	return &buf, writer.FormDataContentType(), nil
}
