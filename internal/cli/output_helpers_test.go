package cli //nolint:testpackage // white-box tests for output helper functions

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatOptional(t *testing.T) {
	t.Parallel()

	t.Run("nil returns empty string", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, formatOptional[int32](nil))
	})

	t.Run("non-nil returns formatted integer", func(t *testing.T) {
		t.Parallel()

		v := int32(42)
		assert.Equal(t, "42", formatOptional(&v))
	})

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()

		v := int32(0)
		assert.Equal(t, "0", formatOptional(&v))
	})

	t.Run("string pointer", func(t *testing.T) {
		t.Parallel()

		s := "hello"
		assert.Equal(t, "hello", formatOptional(&s))
	})

	t.Run("bool pointer", func(t *testing.T) {
		t.Parallel()

		b := true
		assert.Equal(t, "true", formatOptional(&b))
	})
}

func TestFormatDeleteBulkResult_successfullyDeleted(t *testing.T) {
	t.Parallel()

	ids := []int64{42}
	result := &types.DeleteBulkRS{
		SuccessfullyDeleted: &ids,
	}

	headers, rows := formatDeleteBulkResult(result, "Project 'foo'")

	assert.Equal(t, []string{"STATUS", "MESSAGE"}, headers)
	require.Len(t, rows, 1)
	assert.Equal(t, "Deleted", rows[0][0])
	assert.Equal(t, "Project 'foo' deleted successfully", rows[0][1])
}

func TestFormatDeleteBulkResult_errors(t *testing.T) {
	t.Parallel()

	errs := []types.ErrorRS{{ErrorCode: 500, Message: "internal error"}}
	result := &types.DeleteBulkRS{
		Errors: &errs,
	}

	headers, rows := formatDeleteBulkResult(result, "Project 'foo'")

	assert.Equal(t, []string{"STATUS", "MESSAGE"}, headers)
	require.Len(t, rows, 1)
	assert.Equal(t, "Error", rows[0][0])
	assert.Equal(t, "internal error", rows[0][1])
}

func TestFormatDeleteBulkResult_notFound(t *testing.T) {
	t.Parallel()

	notFound := []int64{99}
	result := &types.DeleteBulkRS{
		NotFound: &notFound,
	}

	headers, rows := formatDeleteBulkResult(result, "Project 'foo'")

	assert.Equal(t, []string{"STATUS", "MESSAGE"}, headers)
	require.Len(t, rows, 1)
	assert.Equal(t, "Not Found", rows[0][0])
	assert.Equal(t, "ID 99 not found", rows[0][1])
}

func TestFormatDeleteBulkResult_emptyResponse(t *testing.T) {
	t.Parallel()

	result := &types.DeleteBulkRS{}

	headers, rows := formatDeleteBulkResult(result, "Project 'foo'")

	assert.Equal(t, []string{"STATUS", "MESSAGE"}, headers)
	require.Len(t, rows, 1)
	assert.Equal(t, "Unknown", rows[0][0])
	assert.Equal(t, "No status information in response", rows[0][1])
}

func TestDeleteSuccessResult(t *testing.T) {
	t.Parallel()

	result, headers, rows := deleteSuccessResult("Group 42 deleted")

	assert.Equal(t, deleteSuccessMessage{Message: "Group 42 deleted"}, result)
	assert.Equal(t, operationResultHeaders(), headers)
	assert.Equal(t, messageRows("Group 42 deleted"), rows)
}
