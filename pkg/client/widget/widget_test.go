package widget_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/clienttest"
	"github.com/klaskosk/report-portal-cli/pkg/client/internal/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWidgetScope_GetByID_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/widget/42")
			testutil.AssertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 42,
			"name": "My Widget",
			"widgetType": "statisticTrend",
			"contentParameters": {"contentFields": ["statistics$executions$total"], "itemsCount": 10, "widgetOptions": {}}
		}`))
		},
	))

	widget, err := testClient.Project("my_project").Widgets().GetByID(t.Context(), 42)
	require.NoError(t, err)

	assert.Equal(t, int64(42), widget.Id)
	assert.Equal(t, "My Widget", widget.Name)
	assert.Equal(t, types.WidgetResourceWidgetType("statisticTrend"), widget.WidgetType)
}

func TestWidgetScope_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Widget '999' not found."}`))

	_, err := c.Project("my_project").Widgets().GetByID(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestWidgetScope_Create_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/widget")
			testutil.AssertMethod(t, httpReq, http.MethodPost)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "New Widget", payload["name"])
			assert.Equal(t, "statisticTrend", payload["widgetType"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(`{"id":10}`))
		},
	))

	result, err := testClient.Project("my_project").Widgets().Create(t.Context(), types.WidgetRQ{
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

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusBadRequest,
		`{"errorCode": 4001, "message": "Incorrect request."}`))

	_, err := c.Project("my_project").Widgets().Create(t.Context(), types.WidgetRQ{})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestWidgetScope_Update_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/widget/42")
			testutil.AssertMethod(t, httpReq, http.MethodPut)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "Updated Widget", payload["name"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Widget successfully updated."}`))
		},
	))

	result, err := testClient.Project("my_project").Widgets().Update(
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

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
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

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/widget/names/all")
			testutil.AssertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [
				"Widget 1",
				"Widget 2"
			],
			"page": {"number": 1, "size": 10, "totalElements": 2, "totalPages": 1}
		}`))
		},
	))

	result, err := testClient.Project("my_project").Widgets().GetNames(t.Context(), "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 2)
	assert.Equal(t, "Widget 1", result.Content[0])
	assert.Equal(t, "Widget 2", result.Content[1])
}

func TestWidgetScope_GetNames_Empty(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/widget/names/all")

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
		},
	))

	result, err := testClient.Project("my_project").Widgets().GetNames(t.Context(), "nonexistent")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Content)
}

func TestWidgetScope_GetData_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/widget/multilevel/42")
			testutil.AssertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 42,
			"name": "Data Widget",
			"widgetType": "statisticTrend",
			"contentParameters": {"contentFields": ["statistics$executions$total"], "itemsCount": 10, "widgetOptions": {}}
		}`))
		},
	))

	widget, err := testClient.Project("my_project").Widgets().GetData(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, widget)

	assert.Equal(t, int64(42), widget.Id)
	assert.Equal(t, "Data Widget", widget.Name)
}

func TestWidgetScope_GetData_WithAttributes(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			query := httpReq.URL.Query()
			assert.Equal(t, []string{"attr1", "attr2"}, query["attributes"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"id": 42,
			"name": "Data Widget",
			"widgetType": "statisticTrend",
			"contentParameters": {"contentFields": [], "itemsCount": 10, "widgetOptions": {}}
		}`))
		},
	))

	_, err := testClient.Project("my_project").Widgets().GetData(t.Context(), 42,
		client.WithWidgetAttributes("attr1", "attr2"),
	)
	require.NoError(t, err)
}

func TestWidgetScope_GetData_NotFound(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusNotFound,
		`{"errorCode": 40410, "message": "Widget '999' not found."}`))

	_, err := c.Project("my_project").Widgets().GetData(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}
