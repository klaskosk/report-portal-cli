package download //nolint:testpackage // white-box tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDownloadCmd_flags(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()
	require.NoError(t, cmd.ParseFlags([]string{"--output-file", "out.bin", "--log"}))

	assert.Equal(t, "out.bin", resolveDownloadOutputPath(cmd))

	logFlag, err := cmd.Flags().GetBool("log")
	require.NoError(t, err)
	assert.True(t, logFlag)
}

func TestRunDownload_invalidDataID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("RP_URL", srv.URL)
	t.Setenv("RP_TOKEN", "test-token")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, NewCommand())
	require.NoError(t, cmd.ParseFlags([]string{"-p", "my_project"}))

	err := runDownload(cmd, []string{"not-a-number"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid data ID "not-a-number"`)
}

func TestResolveDataIDFromLog_noBinaryContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/my_project/log/99", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":99,"message":"hello"}`))
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(srv.URL, "test-token")
	require.NoError(t, err)

	cmd := testutil.AttachCmdToRoot(t, NewCommand())
	printer := core.NewPrinter("")

	_, err = resolveDataIDFromLog(cmd, apiClient, "my_project", "99", printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no attachment")
}

func TestResolveDataIDFromLog_emptyBinaryContentID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/my_project/log/99", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":99,"binaryContent":{}}`))
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(srv.URL, "test-token")
	require.NoError(t, err)

	cmd := testutil.AttachCmdToRoot(t, NewCommand())

	_, err = resolveDataIDFromLog(cmd, apiClient, "my_project", "99", core.NewPrinter(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty binaryContent.id")
}

func TestResolveDataIDFromLog_success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/my_project/log/99", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":99,"binaryContent":{"id":"12345"}}`))
	}))
	t.Cleanup(srv.Close)

	apiClient, err := client.New(srv.URL, "test-token")
	require.NoError(t, err)

	cmd := testutil.AttachCmdToRoot(t, NewCommand())
	cmd.SetContext(context.Background())

	dataID, err := resolveDataIDFromLog(cmd, apiClient, "my_project", "99", core.NewPrinter(""))
	require.NoError(t, err)
	assert.Equal(t, int64(12345), dataID)
}
