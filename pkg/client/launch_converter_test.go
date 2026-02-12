package client_test

import (
	"reflect"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fieldNames returns the exported field names of a struct type.
func fieldNames(t reflect.Type) map[string]bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	names := make(map[string]bool, t.NumField())

	for i := range t.NumField() {
		names[t.Field(i).Name] = true
	}

	return names
}

func TestLaunchParamStructs_HaveIdenticalFields(t *testing.T) {
	t.Parallel()

	sourceFields := fieldNames(reflect.TypeFor[api.GetProjectLaunchesUsingGETParams]())
	latestFields := fieldNames(reflect.TypeFor[api.GetLatestLaunchesUsingGETParams]())
	debugFields := fieldNames(reflect.TypeFor[api.GetDebugLaunchesUsingGETParams]())

	require.NotEmpty(t, sourceFields, "source params should have fields")

	t.Run("latest is subset of source", func(t *testing.T) {
		t.Parallel()

		for field := range latestFields {
			assert.True(t, sourceFields[field],
				"GetLatestLaunchesUsingGETParams has field %q not present in source", field)
		}
	})

	t.Run("debug is subset of source", func(t *testing.T) {
		t.Parallel()

		for field := range debugFields {
			assert.True(t, sourceFields[field],
				"GetDebugLaunchesUsingGETParams has field %q not present in source", field)
		}
	})

	t.Run("latest and debug have same fields", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, latestFields, debugFields,
			"latest and debug param types should have identical field sets")
	})
}

func TestConvertToLatestParams_CopiesAllFields(t *testing.T) {
	t.Parallel()

	src := populateAllFields(reflect.TypeOf(api.GetProjectLaunchesUsingGETParams{})).
		Interface().(api.GetProjectLaunchesUsingGETParams)

	result := client.ConvertToLatestParams(&src)
	require.NotNil(t, result)

	resultValue := reflect.ValueOf(*result)
	resultType := resultValue.Type()

	for i := range resultType.NumField() {
		fieldName := resultType.Field(i).Name
		resultField := resultValue.Field(i)

		assert.False(t, resultField.IsZero(),
			"convertToLatestParams did not copy field %q", fieldName)
	}
}

func TestConvertToDebugParams_CopiesAllFields(t *testing.T) {
	t.Parallel()

	src := populateAllFields(reflect.TypeOf(api.GetProjectLaunchesUsingGETParams{})).
		Interface().(api.GetProjectLaunchesUsingGETParams)

	result := client.ConvertToDebugParams(&src)
	require.NotNil(t, result)

	resultValue := reflect.ValueOf(*result)
	resultType := resultValue.Type()

	for i := range resultType.NumField() {
		fieldName := resultType.Field(i).Name
		resultField := resultValue.Field(i)

		assert.False(t, resultField.IsZero(),
			"convertToDebugParams did not copy field %q", fieldName)
	}
}

// populateAllFields creates a value of the given struct type with every pointer field set to a
// non-zero sentinel value. This enables detecting missed copies by checking for zero-value fields
// in the converter output.
func populateAllFields(structType reflect.Type) reflect.Value {
	value := reflect.New(structType).Elem()

	for i := range structType.NumField() {
		field := value.Field(i)
		fieldType := field.Type()

		switch fieldType.Kind() { //nolint:exhaustive // only pointer and slice fields exist in param structs
		case reflect.Ptr:
			sentinel := reflect.New(fieldType.Elem())
			setSentinel(sentinel.Elem())
			field.Set(sentinel)
		case reflect.Slice:
			slice := reflect.MakeSlice(fieldType, 1, 1)
			field.Set(slice)
		default:
			setSentinel(field)
		}
	}

	return value
}

// setSentinel sets a reflect.Value to a non-zero value appropriate for its kind.
func setSentinel(value reflect.Value) {
	switch value.Kind() { //nolint:exhaustive // only types used in param structs
	case reflect.String:
		value.SetString("sentinel")
	case reflect.Int32, reflect.Int64:
		value.SetInt(1)
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Slice:
		slice := reflect.MakeSlice(value.Type(), 1, 1)
		value.Set(slice)
	}
}
