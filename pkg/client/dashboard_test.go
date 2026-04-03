package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboardScope_List_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/my_project/dashboard")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": 1, "name": "Dashboard 1"},
				{"id": 2, "name": "Dashboard 2"}
			],
			"page": {"number": 1, "size": 10, "totalElements": 2, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Dashboards().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result.Content)
	require.Len(t, *result.Content, 2)

	assert.Equal(t, "Dashboard 1", (*result.Content)[0].Name)
	assert.EqualValues(t, 2, *result.Page.TotalElements)
}

func TestDashboardScope_List_WithOptions(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		assert.Equal(t, "2", query.Get("page.page"))
		assert.Equal(t, "5", query.Get("page.size"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 2, "size": 5, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	_, err := c.Project("my_project").Dashboards().List(t.Context(),
		client.WithDashboardPageNumber(2),
		client.WithDashboardPageSize(5),
	)
	require.NoError(t, err)
}

func TestDashboardScope_List_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Dashboards().List(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestDashboardScope_GetByID_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/dashboard/42")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"name": "My Dashboard",
			"description": "Test dashboard",
			"owner": "admin"
		}`))
	}))

	dashboard, err := c.Project("my_project").Dashboards().GetByID(t.Context(), 42)
	require.NoError(t, err)

	assert.Equal(t, int64(42), dashboard.Id)
	assert.Equal(t, "My Dashboard", dashboard.Name)
	require.NotNil(t, dashboard.Description)
	assert.Equal(t, "Test dashboard", *dashboard.Description)
}

func TestDashboardScope_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Dashboard '999' not found."}`))

	_, err := c.Project("my_project").Dashboards().GetByID(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestDashboardScope_Create_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/dashboard")
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "New Dashboard", payload["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":10}`))
	}))

	result, err := c.Project("my_project").Dashboards().Create(t.Context(), types.CreateDashboardRQ{
		Name: "New Dashboard",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, int64(10), *result.Id)
}

func TestDashboardScope_Create_BadRequest(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 4001, "message": "Incorrect request."}`))

	_, err := c.Project("my_project").Dashboards().Create(t.Context(), types.CreateDashboardRQ{})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestDashboardScope_Update_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/dashboard/42")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "Updated Dashboard", payload["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Dashboard successfully updated."}`))
	}))

	result, err := c.Project("my_project").Dashboards().Update(
		t.Context(),
		42,
		types.UpdateDashboardRQ{Name: "Updated Dashboard"},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Dashboard successfully updated.", *result.Message)
}

func TestDashboardScope_Update_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Dashboard '999' not found."}`))

	_, err := c.Project("my_project").Dashboards().Update(
		t.Context(),
		999,
		types.UpdateDashboardRQ{},
	)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestDashboardScope_Delete_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/dashboard/42")
		assertMethod(t, r, http.MethodDelete)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Dashboard successfully deleted."}`))
	}))

	result, err := c.Project("my_project").Dashboards().Delete(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Dashboard successfully deleted.", *result.Message)
}

func TestDashboardScope_Delete_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Dashboard '999' not found."}`))

	_, err := c.Project("my_project").Dashboards().Delete(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestDashboardScope_AddWidget_Success(t *testing.T) {
	t.Parallel()

	widgetID := int64(5)

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/dashboard/42/add")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		addWidget := payload["addWidget"].(map[string]any)
		assert.Equal(t, float64(5), addWidget["widgetId"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Widget added to dashboard."}`))
	}))

	result, err := c.Project("my_project").Dashboards().AddWidget(
		t.Context(),
		42,
		types.AddWidgetRq{
			AddWidget: types.WidgetObjectModel{
				WidgetId: widgetID,
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Widget added to dashboard.", *result.Message)
}

func TestDashboardScope_AddWidget_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Dashboard '999' not found."}`))

	_, err := c.Project("my_project").Dashboards().AddWidget(
		t.Context(),
		999,
		types.AddWidgetRq{},
	)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestDashboardScope_RemoveWidget_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/dashboard/42/5")
		assertMethod(t, r, http.MethodDelete)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Widget removed from dashboard."}`))
	}))

	result, err := c.Project("my_project").Dashboards().RemoveWidget(t.Context(), 42, 5)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Widget removed from dashboard.", *result.Message)
}

func TestDashboardScope_RemoveWidget_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Dashboard '999' not found."}`))

	_, err := c.Project("my_project").Dashboards().RemoveWidget(t.Context(), 999, 5)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
