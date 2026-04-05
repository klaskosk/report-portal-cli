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

func TestFilterScope_List_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/api/v1/my_project/filter")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": 1, "name": "Filter 1", "type": "launch", "owner": "admin", "conditions": [], "orders": []},
				{"id": 2, "name": "Filter 2", "type": "launch", "owner": "admin", "conditions": [], "orders": []}
			],
			"page": {"number": 1, "size": 10, "totalElements": 2, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Filters().List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result.Content)
	require.Len(t, *result.Content, 2)

	assert.Equal(t, "Filter 1", (*result.Content)[0].Name)
	assert.EqualValues(t, 2, *result.Page.TotalElements)
}

func TestFilterScope_List_WithOptions(t *testing.T) {
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

	_, err := c.Project("my_project").Filters().List(t.Context(),
		client.WithFilterPageNumber(2),
		client.WithFilterPageSize(5),
	)
	require.NoError(t, err)
}

func TestFilterScope_List_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Filters().List(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestFilterScope_GetByID_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/filter/42")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"id": 42,
			"name": "My Filter",
			"type": "launch",
			"owner": "admin",
			"conditions": [{"filteringField": "name", "condition": "eq", "value": "test"}],
			"orders": [{"sortingColumn": "startTime", "isAsc": false}]
		}`))
	}))

	filter, err := c.Project("my_project").Filters().GetByID(t.Context(), 42)
	require.NoError(t, err)

	assert.Equal(t, int64(42), filter.Id)
	assert.Equal(t, "My Filter", filter.Name)
	assert.Equal(t, "launch", filter.Type)
	require.Len(t, filter.Conditions, 1)
}

func TestFilterScope_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User filter '999' not found."}`))

	_, err := c.Project("my_project").Filters().GetByID(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestFilterScope_Create_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/filter")
		assertMethod(t, r, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "New Filter", payload["name"])
		assert.Equal(t, "launch", payload["type"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":10}`))
	}))

	result, err := c.Project("my_project").Filters().Create(t.Context(), types.UpdateUserFilterRQ{
		Name:       "New Filter",
		Type:       types.UpdateUserFilterRQTypeLaunch,
		Conditions: []types.UserFilterCondition{},
		Orders:     []types.Order{},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, int64(10), *result.Id)
}

func TestFilterScope_Create_BadRequest(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusBadRequest,
		`{"errorCode": 4001, "message": "Incorrect request."}`))

	_, err := c.Project("my_project").Filters().Create(t.Context(), types.UpdateUserFilterRQ{})
	require.Error(t, err)
	assert.True(t, client.IsBadRequest(err))
}

func TestFilterScope_Update_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/filter/42")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "Updated Filter", payload["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Filter successfully updated."}`))
	}))

	result, err := c.Project("my_project").Filters().Update(
		t.Context(),
		42,
		types.UpdateUserFilterRQ{
			Name:       "Updated Filter",
			Type:       types.UpdateUserFilterRQTypeLaunch,
			Conditions: []types.UserFilterCondition{},
			Orders:     []types.Order{},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Filter successfully updated.", *result.Message)
}

func TestFilterScope_Update_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User filter '999' not found."}`))

	_, err := c.Project("my_project").Filters().Update(
		t.Context(),
		999,
		types.UpdateUserFilterRQ{
			Name:       "Missing",
			Type:       types.UpdateUserFilterRQTypeLaunch,
			Conditions: []types.UserFilterCondition{},
			Orders:     []types.Order{},
		},
	)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestFilterScope_Delete_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/filter/42")
		assertMethod(t, r, http.MethodDelete)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Filter successfully deleted."}`))
	}))

	result, err := c.Project("my_project").Filters().Delete(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Filter successfully deleted.", *result.Message)
}

func TestFilterScope_Delete_NotFound(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusNotFound,
		`{"errorCode": 40410, "message": "User filter '999' not found."}`))

	_, err := c.Project("my_project").Filters().Delete(t.Context(), 999)
	require.Error(t, err)
	assert.True(t, client.IsNotFound(err))
}

func TestFilterScope_GetNames_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/filter/names")
		assertMethod(t, r, http.MethodGet)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [
				{"id": "1", "name": "Filter 1", "owner": "admin"},
				{"id": "2", "name": "Filter 2", "owner": "admin"}
			],
			"page": {"number": 1, "size": 10, "totalElements": 2, "totalPages": 1}
		}`))
	}))

	result, err := c.Project("my_project").Filters().GetNames(t.Context(), "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, *result.Content, 2)
}

func TestFilterScope_GetNames_Empty(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/filter/names")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"content": [],
			"page": {"number": 1, "size": 10, "totalElements": 0, "totalPages": 0}
		}`))
	}))

	result, err := c.Project("my_project").Filters().GetNames(t.Context(), "nonexistent")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Content)
}

func TestUserFilterScope_BulkUpdate_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/filter")
		assertMethod(t, r, http.MethodPut)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		elements, ok := payload["elements"].([]any)
		require.True(t, ok)
		assert.Len(t, elements, 2)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
			{"message": "Filter 1 successfully updated."},
			{"message": "Filter 2 successfully updated."}
		]`))
	}))

	results, err := c.Project("my_project").Filters().BulkUpdate(t.Context(), types.CollectionsRQBulkUpdateFilterRQ{
		Elements: []types.BulkUpdateFilterRQ{
			{
				Id:         "1",
				Name:       "Filter 1",
				Type:       types.BulkUpdateFilterRQTypeLaunch,
				Conditions: []types.UserFilterCondition{},
				Orders:     []types.Order{},
			},
			{
				Id:         "2",
				Name:       "Filter 2",
				Type:       types.BulkUpdateFilterRQTypeLaunch,
				Conditions: []types.UserFilterCondition{},
				Orders:     []types.Order{},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, results, 2)

	require.NotNil(t, results[0].Message)
	assert.Equal(t, "Filter 1 successfully updated.", *results[0].Message)
	require.NotNil(t, results[1].Message)
	assert.Equal(t, "Filter 2 successfully updated.", *results[1].Message)
}

func TestUserFilterScope_BulkUpdate_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Filters().BulkUpdate(t.Context(), types.CollectionsRQBulkUpdateFilterRQ{
		Elements: []types.BulkUpdateFilterRQ{
			{
				Id:         "1",
				Name:       "Filter 1",
				Type:       types.BulkUpdateFilterRQTypeLaunch,
				Conditions: []types.UserFilterCondition{},
				Orders:     []types.Order{},
			},
		},
	})
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestUserFilterScope_BulkUpdate_EmptyResponse(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))

	results, err := c.Project("my_project").Filters().BulkUpdate(t.Context(), types.CollectionsRQBulkUpdateFilterRQ{
		Elements: []types.BulkUpdateFilterRQ{},
	})
	require.NoError(t, err)
	assert.Empty(t, results)
	assert.NotNil(t, results)
}

func TestUserFilterScope_BulkUpdate_NullResponse(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`null`))
	}))

	results, err := c.Project("my_project").Filters().BulkUpdate(t.Context(), types.CollectionsRQBulkUpdateFilterRQ{
		Elements: []types.BulkUpdateFilterRQ{},
	})
	require.NoError(t, err)
	assert.Empty(t, results)
	assert.NotNil(t, results)
}

func TestFilterScope_GetByIds_Success(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertPath(t, r, "/api/v1/my_project/filter/filters")
		assertMethod(t, r, http.MethodGet)

		query := r.URL.Query()
		assert.Equal(t, []string{"1", "2"}, query["ids"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[
			{
				"id": 1,
				"name": "Filter 1",
				"type": "launch",
				"owner": "admin",
				"conditions": [],
				"orders": []
			},
			{
				"id": 2,
				"name": "Filter 2",
				"type": "launch",
				"owner": "admin",
				"conditions": [],
				"orders": []
			}
		]`))
	}))

	filters, err := c.Project("my_project").Filters().GetByIds(t.Context(), []int64{1, 2})
	require.NoError(t, err)
	require.Len(t, filters, 2)

	assert.Equal(t, int64(1), filters[0].Id)
	assert.Equal(t, "Filter 1", filters[0].Name)
	assert.Equal(t, int64(2), filters[1].Id)
	assert.Equal(t, "Filter 2", filters[1].Name)
}

func TestFilterScope_GetByIds_Empty(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))

	filters, err := c.Project("my_project").Filters().GetByIds(t.Context(), []int64{999})
	require.NoError(t, err)
	assert.Empty(t, filters)
}

func TestFilterScope_GetByIds_Unauthorized(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, jsonResponse(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Filters().GetByIds(t.Context(), []int64{1})
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}
