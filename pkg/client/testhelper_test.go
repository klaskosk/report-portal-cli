package client_test

import (
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/clienttest"
)

func newTestClient(t *testing.T, handler http.Handler) *client.Client {
	t.Helper()

	return clienttest.NewClient(t, handler)
}
