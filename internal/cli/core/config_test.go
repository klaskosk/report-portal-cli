package core

import (
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		connection  *Connection
		wantProject string
	}{
		{
			name:        "uses flag value",
			args:        []string{"-p", "my-project"},
			connection:  &Connection{},
			wantProject: "my-project",
		},
		{
			name:        "falls back to connection default",
			connection:  &Connection{Project: "default-project"},
			wantProject: "default-project",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cmd := newTestCmdWithProjectFlag(t)

			if len(test.args) > 0 {
				err := cmd.ParseFlags(test.args)
				require.NoError(t, err)
			}

			project, err := resolveProject(cmd, test.connection)
			require.NoError(t, err)
			assert.Equal(t, test.wantProject, project)
		})
	}
}

func TestResolveProject_EmptyFlag(t *testing.T) {
	t.Parallel()

	cmd := newTestCmdWithProjectFlag(t)

	err := cmd.ParseFlags([]string{"-p", ""})
	require.NoError(t, err)

	_, err = resolveProject(cmd, &Connection{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project name cannot be empty")
}

func TestResolveProject_NoProjectSpecified(t *testing.T) {
	t.Parallel()

	cmd := newTestCmdWithProjectFlag(t)

	_, err := resolveProject(cmd, &Connection{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project specified")
}

func clearRPEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{"RP_URL", "RP_TOKEN", "RP_PROJECT", "RP_INSECURE_SKIP_VERIFY"} {
		t.Setenv(key, "")
	}
}

func TestApplyEnvOverrides_StringOverrides(t *testing.T) {
	clearRPEnv(t)

	t.Setenv("RP_URL", "https://env.example.com")
	t.Setenv("RP_TOKEN", "env-token")
	t.Setenv("RP_PROJECT", "env-project")

	connection := &Connection{
		URL:     "https://profile.example.com",
		Token:   "profile-token",
		Project: "profile-project",
	}

	applyEnvOverrides(connection)

	assert.Equal(t, "https://env.example.com", connection.URL)
	assert.Equal(t, "env-token", connection.Token)
	assert.Equal(t, "env-project", connection.Project)
}

func TestApplyEnvOverrides_EmptyEnvDoesNotOverride(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	connection := &Connection{
		URL:                "https://profile.example.com",
		Token:              "profile-token",
		Project:            "profile-project",
		InsecureSkipVerify: true,
	}

	applyEnvOverrides(connection)

	assert.Equal(t, "https://profile.example.com", connection.URL)
	assert.Equal(t, "profile-token", connection.Token)
	assert.Equal(t, "profile-project", connection.Project)
	assert.True(t, connection.InsecureSkipVerify)
}

func TestApplyEnvOverrides_InsecureSkipVerify_TrueOverridesProfile(t *testing.T) {
	clearRPEnv(t)

	t.Setenv("RP_INSECURE_SKIP_VERIFY", "true")

	connection := &Connection{InsecureSkipVerify: false}
	applyEnvOverrides(connection)

	assert.True(t, connection.InsecureSkipVerify)
}

func TestApplyEnvOverrides_InsecureSkipVerify_FalseOverridesProfile(t *testing.T) {
	clearRPEnv(t)

	t.Setenv("RP_INSECURE_SKIP_VERIFY", "false")

	connection := &Connection{InsecureSkipVerify: true}
	applyEnvOverrides(connection)

	assert.False(t, connection.InsecureSkipVerify,
		"RP_INSECURE_SKIP_VERIFY=false should override a profile value of true")
}

func TestApplyEnvOverrides_InsecureSkipVerify_InvalidValue(t *testing.T) {
	clearRPEnv(t)

	t.Setenv("RP_INSECURE_SKIP_VERIFY", "not-a-bool")

	connection := &Connection{InsecureSkipVerify: true}
	applyEnvOverrides(connection)

	assert.True(t, connection.InsecureSkipVerify,
		"invalid RP_INSECURE_SKIP_VERIFY should leave connection unchanged")
}

func TestNewInsecureTransport_PreservesProxy(t *testing.T) {
	t.Parallel()

	requestURLs := make(chan *url.URL, 1)
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestURLs <- request.URL

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"page":"WELCOME"}`))
	})
	proxy := httptest.NewServer(handler)
	t.Cleanup(proxy.Close)

	proxyURL, err := url.Parse(proxy.URL)
	require.NoError(t, err)

	transport := newInsecureTransport(&http.Transport{Proxy: http.ProxyURL(proxyURL)})
	t.Cleanup(transport.CloseIdleConnections)

	apiClient, err := client.New("http://report-portal.invalid", "test-token",
		client.WithHTTPClient(&http.Client{Transport: transport}))
	require.NoError(t, err)

	result, err := apiClient.Onboarding().GetStatus(t.Context(), nil)
	require.NoError(t, err)
	assert.True(t, (<-requestURLs).IsAbs(), "proxy requests use an absolute request target")
	assert.Equal(t, map[string]any{"page": "WELCOME"}, result)
}

func TestNewClient_InsecureSkipVerify(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"page":"WELCOME"}`))
	})
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	connection := &Connection{URL: server.URL, Token: "test-token"}
	logger := slog.New(slog.DiscardHandler)
	apiClient, err := NewClient(connection, logger)
	require.NoError(t, err)

	_, err = apiClient.Onboarding().GetStatus(t.Context(), nil)
	require.ErrorAs(t, err, new(x509.UnknownAuthorityError))

	cmd := NewMinimalRootCmd()
	err = cmd.ParseFlags([]string{"--insecure-skip-verify"})
	require.NoError(t, err)

	applyFlagOverrides(cmd, connection)
	require.True(t, connection.InsecureSkipVerify)

	apiClient, err = NewClient(connection, logger)
	require.NoError(t, err)

	result, err := apiClient.Onboarding().GetStatus(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"page": "WELCOME"}, result)
}
