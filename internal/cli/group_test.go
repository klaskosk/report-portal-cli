package cli //nolint:testpackage // white-box tests for group CLI helpers

import (
	"context"
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroupCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newGroupCmd()

	assert.Equal(t, []string{"create", "delete", "get", "list", "update"}, commandNames(cmd))
}

func TestGroupRow_populated(t *testing.T) {
	t.Parallel()

	groupID := int64(42)
	name := "My Group"
	slug := "my-group"
	usersCount := int32(5)
	projectsCount := int32(3)
	createdAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	createdBy := int64(1)

	group := &types.GroupInfo{
		Id:        &groupID,
		Name:      &name,
		Slug:      &slug,
		CreatedAt: &createdAt,
		CreatedBy: &createdBy,
		Stats: &types.GroupInfoStats{
			UsersCount:    &usersCount,
			ProjectsCount: &projectsCount,
		},
	}

	headers, row := groupRow(group)

	assert.Equal(t, groupHeaders(), headers)
	assert.Equal(t, []string{
		"42",
		"My Group",
		"my-group",
		"5",
		"3",
		"2024-01-15 10:30:00",
	}, row)
}

func TestGroupRow_nilStats(t *testing.T) {
	t.Parallel()

	groupID := int64(1)
	name := "No Stats Group"

	group := &types.GroupInfo{
		Id:   &groupID,
		Name: &name,
	}

	_, row := groupRow(group)

	assert.Empty(t, row[3], "USERS should be empty when Stats is nil")
	assert.Empty(t, row[4], "PROJECTS should be empty when Stats is nil")
}

func TestGroupRow_nilName(t *testing.T) {
	t.Parallel()

	groupID := int64(2)

	group := &types.GroupInfo{
		Id: &groupID,
	}

	_, row := groupRow(group)

	assert.Empty(t, row[1], "NAME should be empty when nil")
}

func TestGroupRow_nilCreatedAt(t *testing.T) {
	t.Parallel()

	groupID := int64(3)
	name := "Group"

	group := &types.GroupInfo{
		Id:   &groupID,
		Name: &name,
	}

	_, row := groupRow(group)

	assert.Empty(t, row[5], "CREATED should be empty when CreatedAt is nil")
}

func TestParseGroupSort_fieldAndDirection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     []string
		wantField string
		wantOrder string
	}{
		{
			name:      "field and ASC",
			input:     []string{"name,ASC"},
			wantField: "name",
			wantOrder: "ASC",
		},
		{
			name:      "field and desc lowercased",
			input:     []string{"name,desc"},
			wantField: "name",
			wantOrder: "DESC",
		},
		{
			name:      "field only defaults to ASC",
			input:     []string{"name"},
			wantField: "name",
			wantOrder: "ASC",
		},
		{
			name:      "empty slice falls back to default name ASC",
			input:     []string{},
			wantField: "name",
			wantOrder: "ASC",
		},
		{
			name:      "multiple values uses only first",
			input:     []string{"name,ASC", "id,DESC"},
			wantField: "name",
			wantOrder: "ASC",
		},
		{
			name:      "whitespace trimmed from field and order",
			input:     []string{" name , desc "},
			wantField: "name",
			wantOrder: "DESC",
		},
		{
			name:      "empty string entry falls back to default name ASC",
			input:     []string{""},
			wantField: "name",
			wantOrder: "ASC",
		},
		{
			name:      "field with empty direction defaults to ASC",
			input:     []string{"name,"},
			wantField: "name",
			wantOrder: "ASC",
		},
		{
			name:      "field with whitespace-only direction defaults to ASC",
			input:     []string{"name,   "},
			wantField: "name",
			wantOrder: "ASC",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			field, order := parseGroupSort(testCase.input)
			assert.Equal(t, testCase.wantField, field)
			assert.Equal(t, testCase.wantOrder, order)
		})
	}
}

func TestGroupPaginationOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		page       int32
		pageSize   int32
		wantOffset int32
		wantLimit  int32
	}{
		{
			name:       "first page",
			page:       0,
			pageSize:   10,
			wantOffset: 0,
			wantLimit:  10,
		},
		{
			name:       "third page with larger page size",
			page:       2,
			pageSize:   25,
			wantOffset: 50,
			wantLimit:  25,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			offset := testCase.page * testCase.pageSize
			limit := testCase.pageSize

			assert.Equal(t, testCase.wantOffset, offset)
			assert.Equal(t, testCase.wantLimit, limit)
		})
	}
}

func TestFormatInt32Ptr(t *testing.T) {
	t.Parallel()

	t.Run("nil returns empty string", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, formatInt32Ptr(nil))
	})

	t.Run("non-nil returns formatted integer", func(t *testing.T) {
		t.Parallel()

		v := int32(42)
		assert.Equal(t, "42", formatInt32Ptr(&v))
	})

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()

		v := int32(0)
		assert.Equal(t, "0", formatInt32Ptr(&v))
	})
}

func TestGroupCreateCmd_missingName(t *testing.T) {
	t.Parallel()

	cmd := newGroupCreateCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := runGroupCreate(cmd, nil)
	require.Error(t, err)
	assert.EqualError(t, err, "--name is required")
}

func TestParseGroupID_valid(t *testing.T) {
	t.Parallel()

	id, err := parseGroupID("42")
	require.NoError(t, err)
	assert.EqualValues(t, 42, id)
}

func TestParseGroupID_invalid(t *testing.T) {
	t.Parallel()

	_, err := parseGroupID("not-a-number")
	require.Error(t, err)
	assert.EqualError(t, err, `invalid group ID "not-a-number": must be a numeric ID`)
}
