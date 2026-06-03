package settings_test

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

func TestProjectSettingsScope_Get_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/settings")
			testutil.AssertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`{
			"project": 1,
			"subTypes": {
				"TO_INVESTIGATE": [
					{
						"id": 1,
						"locator": "ti001",
						"typeRef": "TO_INVESTIGATE",
						"longName": "To Investigate",
						"shortName": "TI",
						"color": "#ffb743"
					}
				],
				"PRODUCT_BUG": [
					{
						"id": 2,
						"locator": "pb001",
						"typeRef": "PRODUCT_BUG",
						"longName": "Product Bug",
						"shortName": "PB",
						"color": "#ec3900"
					}
				]
			}
		}`))
		},
	))

	settings, err := testClient.Project("my_project").Settings().Get(t.Context())
	require.NoError(t, err)
	require.NotNil(t, settings)

	assert.Equal(t, int64(1), settings.Project)
	assert.Contains(t, settings.SubTypes, "TO_INVESTIGATE")
	assert.Contains(t, settings.SubTypes, "PRODUCT_BUG")

	tiSubTypes := settings.SubTypes["TO_INVESTIGATE"]
	require.Len(t, tiSubTypes, 1)
	require.NotNil(t, tiSubTypes[0].Id)
	assert.Equal(t, int64(1), *tiSubTypes[0].Id)
	require.NotNil(t, tiSubTypes[0].Locator)
	assert.Equal(t, "ti001", *tiSubTypes[0].Locator)
	require.NotNil(t, tiSubTypes[0].LongName)
	assert.Equal(t, "To Investigate", *tiSubTypes[0].LongName)
	require.NotNil(t, tiSubTypes[0].ShortName)
	assert.Equal(t, "TI", *tiSubTypes[0].ShortName)
}

func TestProjectSettingsScope_Get_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    int
		body      string
		assertErr func(error) bool
	}{
		{
			name:      "Unauthorized",
			status:    http.StatusUnauthorized,
			body:      `{"errorCode": 4001, "message": "Full authentication is required."}`,
			assertErr: client.IsUnauthorized,
		},
		{
			name:      "NotFound",
			status:    http.StatusNotFound,
			body:      `{"errorCode": 40410, "message": "Project 'unknown' not found."}`,
			assertErr: client.IsNotFound,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			c := clienttest.NewClient(t, testutil.JSONHandler(testCase.status, testCase.body))

			_, err := c.Project("my_project").Settings().Get(t.Context())
			require.Error(t, err)
			assert.True(t, testCase.assertErr(err))
		})
	}
}

func TestProjectSettingsScope_GetNotifications_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/settings/notification")
			testutil.AssertMethod(t, httpReq, http.MethodGet)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)

			_, _ = responseWriter.Write([]byte(`[
			{
				"id": 10,
				"ruleName": "Launch finished",
				"sendCase": "ALWAYS",
				"enabled": true,
				"recipients": ["admin@example.com"]
			}
		]`))
		},
	))

	notifications, err := testClient.Project("my_project").Settings().GetNotifications(t.Context())
	require.NoError(t, err)
	require.Len(t, notifications, 1)

	assert.Equal(t, int64(10), *notifications[0].Id)
	assert.Equal(t, "Launch finished", notifications[0].RuleName)
}

func TestProjectSettingsScope_GetNotifications_Unauthorized(t *testing.T) {
	t.Parallel()

	c := clienttest.NewClient(t, testutil.JSONHandler(http.StatusUnauthorized,
		`{"errorCode": 4001, "message": "Full authentication is required."}`))

	_, err := c.Project("my_project").Settings().GetNotifications(t.Context())
	require.Error(t, err)
	assert.True(t, client.IsUnauthorized(err))
}

func TestProjectSettingsScope_CreateNotification_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/settings/notification")
			testutil.AssertMethod(t, httpReq, http.MethodPost)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "Test Rule", payload["ruleName"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(`{"id":42}`))
		},
	))

	ruleName := "Test Rule"
	result, err := testClient.Project("my_project").Settings().CreateNotification(
		t.Context(),
		types.SenderCaseDTO{
			RuleName: ruleName,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, int64(42), *result.Id)
}

func TestProjectSettingsScope_CreateNotification_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "BadRequest",
			statusCode: http.StatusBadRequest,
			body:       `{"errorCode": 4001, "message": "Invalid notification rule."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)
			_, err := testClient.Project("my_project").Settings().CreateNotification(
				t.Context(),
				types.SenderCaseDTO{},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestProjectSettingsScope_UpdateNotification_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/settings/notification")
			testutil.AssertMethod(t, httpReq, http.MethodPut)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "Updated Rule", payload["ruleName"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(`{"message":"Notification rule updated."}`))
		},
	))

	ruleName := "Updated Rule"
	result, err := testClient.Project("my_project").Settings().UpdateNotification(
		t.Context(),
		types.SenderCaseDTO{
			RuleName: ruleName,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Notification rule updated.", *result.Message)
}

func TestProjectSettingsScope_UpdateNotification_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "BadRequest",
			statusCode: http.StatusBadRequest,
			body:       `{"errorCode": 4001, "message": "Invalid notification rule."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)
			_, err := testClient.Project("my_project").Settings().UpdateNotification(
				t.Context(),
				types.SenderCaseDTO{},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestProjectSettingsScope_DeleteNotification_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/settings/notification/42")
			testutil.AssertMethod(t, httpReq, http.MethodDelete)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Notification rule deleted."}`))
		},
	))

	result, err := testClient.Project("my_project").Settings().DeleteNotification(t.Context(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Notification rule deleted.", *result.Message)
}

func TestProjectSettingsScope_DeleteNotification_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40410, "message": "Notification rule not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)
			_, err := testClient.Project("my_project").
				Settings().
				DeleteNotification(t.Context(), 999)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestProjectSettingsScope_CreatePatternTemplate_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/settings/pattern")
			testutil.AssertMethod(t, httpReq, http.MethodPost)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "Timeout Pattern", payload["name"])
			assert.Equal(t, "string", payload["type"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(`{"id":10}`))
		},
	))

	name := "Timeout Pattern"
	patternType := "string"
	result, err := testClient.Project("my_project").Settings().CreatePatternTemplate(
		t.Context(),
		types.CreatePatternTemplateRQ{
			Enabled: false,
			Name:    name,
			Type:    patternType,
			Value:   "",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	assert.Equal(t, int64(10), *result.Id)
}

func TestProjectSettingsScope_CreatePatternTemplate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "BadRequest",
			statusCode: http.StatusBadRequest,
			body:       `{"errorCode": 4001, "message": "Invalid pattern template."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)
			_, err := testClient.Project("my_project").Settings().CreatePatternTemplate(
				t.Context(),
				types.CreatePatternTemplateRQ{},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestProjectSettingsScope_UpdatePatternTemplate_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/settings/pattern/10")
			testutil.AssertMethod(t, httpReq, http.MethodPut)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "Updated Pattern", payload["name"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Pattern template updated."}`))
		},
	))

	name := "Updated Pattern"
	result, err := testClient.Project("my_project").Settings().UpdatePatternTemplate(
		t.Context(),
		10,
		types.UpdatePatternTemplateRQ{
			Enabled: true,
			Name:    name,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Pattern template updated.", *result.Message)
}

func TestProjectSettingsScope_UpdatePatternTemplate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40410, "message": "Pattern template not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)
			_, err := testClient.Project("my_project").Settings().UpdatePatternTemplate(
				t.Context(),
				999,
				types.UpdatePatternTemplateRQ{},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestProjectSettingsScope_DeletePatternTemplate_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/settings/pattern/10")
			testutil.AssertMethod(t, httpReq, http.MethodDelete)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Pattern template deleted."}`))
		},
	))

	result, err := testClient.Project("my_project").Settings().
		DeletePatternTemplate(t.Context(), 10)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Pattern template deleted.", *result.Message)
}

func TestProjectSettingsScope_DeletePatternTemplate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40410, "message": "Pattern template not found."}`,
			check:      client.IsNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)
			_, err := testClient.Project("my_project").
				Settings().
				DeletePatternTemplate(t.Context(), 999)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestProjectSettingsScope_CreateIssueSubType_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/settings/sub-type")
			testutil.AssertMethod(t, httpReq, http.MethodPost)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			assert.Equal(t, "Custom Bug", payload["longName"])
			assert.Equal(t, "CB", payload["shortName"])
			assert.Equal(t, "#ff0000", payload["color"])
			assert.Equal(t, "PRODUCT_BUG", payload["typeRef"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusCreated)
			_, _ = responseWriter.Write([]byte(`{"id":100,"locator":"custom_bug_001"}`))
		},
	))

	result, err := testClient.Project("my_project").Settings().CreateIssueSubType(
		t.Context(),
		types.CreateIssueSubTypeRQ{
			LongName:  "Custom Bug",
			ShortName: "CB",
			Color:     "#ff0000",
			TypeRef:   "PRODUCT_BUG",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Id)
	require.NotNil(t, result.Locator)
	assert.Equal(t, int64(100), *result.Id)
	assert.Equal(t, "custom_bug_001", *result.Locator)
}

func TestProjectSettingsScope_CreateIssueSubType_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "BadRequest",
			statusCode: http.StatusBadRequest,
			body:       `{"errorCode": 4001, "message": "Invalid issue sub-type."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)
			_, err := testClient.Project("my_project").Settings().CreateIssueSubType(
				t.Context(),
				types.CreateIssueSubTypeRQ{},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestProjectSettingsScope_UpdateIssueSubType_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/settings/sub-type")
			testutil.AssertMethod(t, httpReq, http.MethodPut)

			body, err := io.ReadAll(httpReq.Body)
			if !assert.NoError(t, err) {
				return
			}

			var payload map[string]any
			if !assert.NoError(t, json.Unmarshal(body, &payload)) {
				return
			}

			ids, ok := payload["ids"].([]any)
			if !assert.True(t, ok) {
				return
			}

			if !assert.Len(t, ids, 1) {
				return
			}

			entry, okEntry := ids[0].(map[string]any)
			if !assert.True(t, okEntry) {
				return
			}

			assert.Equal(t, "Updated Bug", entry["longName"])
			assert.Equal(t, "PRODUCT_BUG", entry["typeRef"])
			assert.Equal(t, "pb001", entry["locator"])

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Issue sub-type updated."}`))
		},
	))

	result, err := testClient.Project("my_project").Settings().UpdateIssueSubType(
		t.Context(),
		types.UpdateOneIssueSubTypeRQ{
			Color:     "#000000",
			Locator:   "pb001",
			LongName:  "Updated Bug",
			ShortName: "UB",
			TypeRef:   "PRODUCT_BUG",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Issue sub-type updated.", *result.Message)
}

func TestProjectSettingsScope_UpdateIssueSubType_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "BadRequest",
			statusCode: http.StatusBadRequest,
			body:       `{"errorCode": 4001, "message": "Invalid issue sub-type."}`,
			check:      client.IsBadRequest,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)
			_, err := testClient.Project("my_project").Settings().UpdateIssueSubType(
				t.Context(),
				types.UpdateOneIssueSubTypeRQ{
					Color:     "",
					Locator:   "pb001",
					LongName:  "",
					ShortName: "",
					TypeRef:   "PRODUCT_BUG",
				},
			)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}

func TestProjectSettingsScope_DeleteIssueSubType_Success(t *testing.T) {
	t.Parallel()

	testClient := clienttest.NewClient(t, http.HandlerFunc(
		func(responseWriter http.ResponseWriter, httpReq *http.Request) {
			testutil.AssertBearerToken(t, httpReq)
			testutil.AssertPath(t, httpReq, "/api/v1/my_project/settings/sub-type/100")
			testutil.AssertMethod(t, httpReq, http.MethodDelete)

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"message":"Issue sub-type deleted."}`))
		},
	))

	result, err := testClient.Project("my_project").Settings().DeleteIssueSubType(t.Context(), 100)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Message)
	assert.Equal(t, "Issue sub-type deleted.", *result.Message)
}

func TestProjectSettingsScope_DeleteIssueSubType_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		body       string
		check      func(error) bool
	}{
		{
			name:       "NotFound",
			statusCode: http.StatusNotFound,
			body:       `{"errorCode": 40410, "message": "Issue sub-type not found."}`,
			check:      client.IsNotFound,
		},
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorCode": 4001, "message": "Full authentication is required."}`,
			check:      client.IsUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testClient := clienttest.NewClient(
				t,
				testutil.JSONHandler(testCase.statusCode, testCase.body),
			)
			_, err := testClient.Project("my_project").
				Settings().
				DeleteIssueSubType(t.Context(), 999)

			require.Error(t, err)
			assert.True(t, testCase.check(err))
		})
	}
}
