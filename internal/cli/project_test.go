package cli //nolint:testpackage // white-box tests for project CLI helpers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProjectCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newProjectCmd()

	assert.Equal(
		t,
		[]string{"create", "delete", "export", "get", "list", "update"},
		commandNames(cmd),
	)
}

func TestNewProjectCreateCmd_DefaultEntryType(t *testing.T) {
	t.Parallel()

	cmd := newProjectCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	entryType := mustGetStringFlag(cmd, "entry-type")
	assert.Equal(t, "INTERNAL", entryType)
}

func TestNewProjectCreateCmd_CustomEntryType(t *testing.T) {
	t.Parallel()

	cmd := newProjectCreateCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--entry-type", "PERSONAL"}))

	entryType := mustGetStringFlag(cmd, "entry-type")
	assert.Equal(t, "PERSONAL", entryType)
}

func TestRunProjectCreate_requiresName(t *testing.T) {
	t.Parallel()

	cmd := newProjectCreateCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := runProjectCreate(cmd, nil)
	require.Error(t, err)
	assert.EqualError(t, err, "--name is required")
}

func TestNewProjectUpdateCmd_requiresFromFile(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newProjectUpdateCmd())

	err := runProjectUpdate(cmd, []string{"my_project"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestReadProjectUpdateFromFile_missingFile(t *testing.T) {
	t.Parallel()

	_, err := readProjectUpdateFromFile(filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading project update payload")
}

func TestReadProjectUpdateFromFile_invalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{`), 0o600))

	_, err := readProjectUpdateFromFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing project update JSON")
}

func TestReadProjectUpdateFromFile_validJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	jsonContent := `{
  "configuration": {
    "attributes": {
      "job.keepLaunches": "3 months"
    }
  }
}`
	require.NoError(t, os.WriteFile(path, []byte(jsonContent), 0o600))

	request, err := readProjectUpdateFromFile(path)
	require.NoError(t, err)
	require.NotNil(t, request.Configuration)
	assert.Equal(t, "3 months", request.Configuration.Attributes["job.keepLaunches"])
}

func TestNewProjectDeleteCmd_requiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	cmd := newProjectDeleteCmd()

	require.Error(t, cmd.Args(cmd, []string{}), "should reject zero args")
	require.NoError(t, cmd.Args(cmd, []string{"my_project"}), "should accept exactly one arg")
	require.Error(t, cmd.Args(cmd, []string{"a", "b"}), "should reject two args")
}

func TestDeleteProject_notFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorCode":40410,"message":"Project not found."}`))
	}))
	t.Cleanup(srv.Close)

	c, err := client.New(srv.URL, "test-token")
	require.NoError(t, err)

	cmd := attachCmdToRoot(t, newProjectDeleteCmd())

	err = deleteProject(cmd, c, "missing-proj", newPrinter(""))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `project "missing-proj" not found: cannot delete`)
	assert.True(t, client.IsNotFound(err))
}

func TestDeleteProject_genericGetError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errorCode":5000,"message":"internal error"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := client.New(srv.URL, "test-token")
	require.NoError(t, err)

	cmd := attachCmdToRoot(t, newProjectDeleteCmd())

	err = deleteProject(cmd, c, "my-proj", newPrinter(""))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `resolving project "my-proj" for deletion`)
	assert.True(t, client.IsServerError(err))
}

func TestDeleteProject_success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")

			switch r.Method {
			case http.MethodGet:
				responseWriter.WriteHeader(http.StatusOK)
				_, _ = responseWriter.Write([]byte(
					`{"projectId":42,"projectName":"my-project","entryType":"INTERNAL","configuration":{}}`,
				))
			case http.MethodDelete:
				responseWriter.WriteHeader(http.StatusOK)
				_, _ = responseWriter.Write([]byte(`{"successfullyDeleted":[42]}`))
			}
		}),
	)
	t.Cleanup(srv.Close)

	c, err := client.New(srv.URL, "test-token")
	require.NoError(t, err)

	cmd := attachCmdToRoot(t, newProjectDeleteCmd())

	err = deleteProject(cmd, c, "my-project", newPrinter(""))
	require.NoError(t, err)
}
