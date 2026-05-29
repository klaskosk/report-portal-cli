package cli //nolint:testpackage // white-box tests for project CLI helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectListRows(t *testing.T) {
	t.Parallel()

	headers, rows := projectListRows([]string{"alpha", "beta"})

	assert.Equal(t, []string{"PROJECT NAME"}, headers)
	assert.Equal(t, [][]string{{"alpha"}, {"beta"}}, rows)
}

func TestProjectListRows_empty(t *testing.T) {
	t.Parallel()

	headers, rows := projectListRows(nil)

	assert.Equal(t, []string{"PROJECT NAME"}, headers)
	assert.Empty(t, rows)
}

func TestListProjects_success(t *testing.T) {
	t.Parallel()

	srv := newProjectListTestServer(t)
	apiClient, err := client.New(srv.URL, "test-token")
	require.NoError(t, err)

	cmd := attachCmdToRoot(t, newProjectListCmd())

	var buf bytes.Buffer

	printer := &Printer{format: "json", out: &buf, errOut: io.Discard}

	err = listProjects(cmd, apiClient, printer)
	require.NoError(t, err)

	var names []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &names))
	assert.Equal(t, []string{"beta_proj", "zebra_proj"}, names)
}

func TestPrinter_Print_projectNameList_formats(t *testing.T) {
	t.Parallel()

	names := []string{"beta_proj", "zebra_proj"}
	headers, rows := projectListRows(names)

	t.Run("table", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		printer := &Printer{format: "table", out: &buf, errOut: io.Discard}

		require.NoError(t, printer.Print(names, headers, rows))

		out := buf.String()
		assert.Contains(t, out, "PROJECT NAME")
		assert.Contains(t, out, "beta_proj")
	})

	t.Run("json", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		printer := &Printer{format: "json", out: &buf, errOut: io.Discard}

		require.NoError(t, printer.Print(names, headers, rows))

		var decoded []string
		require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
		assert.Equal(t, names, decoded)
	})

	t.Run("go-template", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		printer := &Printer{
			format: `go-template={{range .}}{{.}};{{end}}`,
			out:    &buf,
			errOut: io.Discard,
		}

		require.NoError(t, printer.Print(names, headers, rows))

		assert.Equal(t, "beta_proj;zebra_proj;\n", buf.String())
	})
}

func newProjectListTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")

			switch {
			case request.Method == http.MethodGet && request.URL.Path == "/api/users":
				responseWriter.WriteHeader(http.StatusOK)
				_, _ = responseWriter.Write(
					[]byte(`{"id":1,"userId":"tester","email":"t@example.com"}`),
				)
			case request.Method == http.MethodGet && request.URL.Path == "/api/users/tester/projects":
				responseWriter.WriteHeader(http.StatusOK)
				_, _ = responseWriter.Write([]byte(`{
				"zebra_proj": {"entryType": "INTERNAL", "projectRole": "MEMBER"},
				"beta_proj": {"entryType": "INTERNAL", "projectRole": "MEMBER"}
			}`))
			default:
				t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
				responseWriter.WriteHeader(http.StatusNotFound)
			}
		}),
	)
	t.Cleanup(srv.Close)

	return srv
}

func TestNewProjectCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newProjectCmd()

	assert.Equal(
		t,
		[]string{"create", "delete", "export", "get", "list", "update", "user"},
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
