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

func TestWidgetScope_GetByID_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/widget/42")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"name": "My Widget",
			"widgetType": "statisticTrend",
			"contentParameters": {"contentFields": ["statistics$executions$total"], "itemsCount": 10, "widgetOptions": {}}
		}`))
	}))

	widget, err := c.Project("my_project").Widgets().GetByID(t.Context(), 42)
	require.NoError(t, err)

	assert.Equal(t, int64(42), widget.Id)
	assert.Equal(t, "My Widget", widget.Name)
	assert.Equal(t, types.WidgetResourceWidgetType("statisticTrend"), widget.WidgetType)
}

func TestWidgetScope_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Widget '999' not found."}`))

	_, err := c.Project("my_project").Widgets().GetByID(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestWidgetScope_Create_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/widget")
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "New Widget", payload["name"])
		assert.Equal(t, "statisticTrend", payload["widgetType"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":10}`))
	}))

	result, err := c.Project("my_project").Widgets().Create(t.Context(), types.WidgetRQ{
		Name:       "New Widget",
		WidgetType: types.WidgetRQWidgetType("statisticTrend"),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, int64(10), *result.Id)
}

func TestWidgetScope_Create_BadRequest(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 4001, "message": "Incorrect request."}`))

	_, err := c.Project("my_project").Widgets().Create(t.Context(), types.WidgetRQ{})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestWidgetScope_Update_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/widget/42")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "Updated Widget", payload["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Widget successfully updated."}`))
	}))

	result, err := c.Project("my_project").Widgets().Update(
		t.Context(),
		42,
		types.WidgetRQ{
			Name:       "Updated Widget",
			WidgetType: types.WidgetRQWidgetType("statisticTrend"),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Widget successfully updated.", *result.Message)
}

func TestWidgetScope_Update_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Widget '999' not found."}`))

	_, err := c.Project("my_project").Widgets().Update(
		t.Context(),
		999,
		types.WidgetRQ{
			Name:       "Missing Widget",
			WidgetType: types.WidgetRQWidgetType("statisticTrend"),
		},
	)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestWidgetScope_GetNames_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/widget/names/all")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				"Widget 1",
				"Widget 2"
			],
			"page": {"number": 1, "size": 10, "totalElements": 2, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Widgets().GetNames(t.Context(), "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 2)
	assert.Equal(t, "Widget 1", result.Content[0]["name"])
	assert.Equal(t, "Widget 2", result.Content[1]["name"])
}

func TestWidgetScope_GetNames_Empty(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/widget/names/all")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	result, err := c.Project("my_project").Widgets().GetNames(t.Context(), "nonexistent")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Content)
}

func TestWidgetScope_GetData_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/widget/multilevel/42")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"name": "Data Widget",
			"widgetType": "statisticTrend",
			"contentParameters": {"contentFields": ["statistics$executions$total"], "itemsCount": 10, "widgetOptions": {}}
		}`))
	}))

	widget, err := c.Project("my_project").Widgets().GetData(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, widget)

	assert.Equal(t, int64(42), widget.Id)
	assert.Equal(t, "Data Widget", widget.Name)
}

func TestWidgetScope_GetData_WithAttributes(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Equal(t, []string{"attr1", "attr2"}, query["attributes"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"name": "Data Widget",
			"widgetType": "statisticTrend",
			"contentParameters": {"contentFields": [], "itemsCount": 10, "widgetOptions": {}}
		}`))
	}))

	_, err := c.Project("my_project").Widgets().GetData(t.Context(), 42,
		client.WithWidgetAttributes("attr1", "attr2"),
	)
	require.NoError(t, err)
}

func TestWidgetScope_GetData_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Widget '999' not found."}`))

	_, err := c.Project("my_project").Widgets().GetData(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
