package cli //nolint:testpackage // white-box tests for unexported user CLI helpers

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUserLookup implements userLookupClient for helper tests.
type fakeUserLookup struct {
	user        *types.UserResource
	err         error
	gotLogin    string
	lookupCalls int
}

// GetByLogin records the requested login and returns the configured result.
func (f *fakeUserLookup) GetByLogin(_ context.Context, login string) (*types.UserResource, error) {
	f.gotLogin = login
	f.lookupCalls++

	if f.err != nil {
		return nil, f.err
	}

	return f.user, nil
}

// commandNames returns the immediate child command names for cmd.
func commandNames(cmd *cobra.Command) []string {
	names := make([]string, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		names = append(names, child.Name())
	}

	slices.Sort(names)

	return names
}

func TestNewGetUserCmd_DefaultListOptions(t *testing.T) {
	t.Parallel()

	cmd := newGetUserCmd()

	options, err := resolveUserListOptions(cmd)
	require.NoError(t, err)

	assert.Equal(t, int32(10), options.pageSize)
	assert.Equal(t, int32(0), options.page)
	assert.Empty(t, options.sort)
	assert.Empty(t, options.role)
	assert.Empty(t, options.accountType)
	assert.Empty(t, options.email)
	assert.Nil(t, options.active)
}

func TestResolveUserListOptions_WithFilters(t *testing.T) {
	t.Parallel()

	cmd := newGetUserCmd()
	err := cmd.ParseFlags([]string{
		"--page-size", "25",
		"--page", "3",
		"--sort", "email,desc",
		"--role", "ADMINISTRATOR",
		"--account-type", "INTERNAL",
		"--active=false",
		"--email", "admin@example.com",
	})
	require.NoError(t, err)

	options, err := resolveUserListOptions(cmd)
	require.NoError(t, err)
	require.NotNil(t, options.role)
	require.NotNil(t, options.accountType)
	require.NotNil(t, options.email)
	require.NotNil(t, options.active)

	assert.Equal(t, int32(25), options.pageSize)
	assert.Equal(t, int32(2), options.page)
	assert.Equal(t, []string{"email,desc"}, options.sort)
	assert.Equal(t, "ADMINISTRATOR", *options.role)
	assert.Equal(t, "INTERNAL", *options.accountType)
	assert.Equal(t, "admin@example.com", *options.email)
	assert.False(t, *options.active)
}

func TestNewUserCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newUserCmd()

	assert.Equal(t, []string{"create", "delete", "update"}, commandNames(cmd))
}

func TestResolveCreateUserRequest_MissingRequiredFlags(t *testing.T) {
	t.Parallel()

	cmd := newUserCreateCmd()

	_, err := resolveCreateUserRequest(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--login")
	assert.Contains(t, err.Error(), "--email")
	assert.Contains(t, err.Error(), "--full-name")
}

func TestResolveCreateUserRequest_MapsFlags(t *testing.T) {
	t.Parallel()

	cmd := newUserCreateCmd()
	err := cmd.ParseFlags([]string{
		"--login", "newuser",
		"--email", "new@example.com",
		"--full-name", "New User",
		"--password", "Secret123!",
		"--account-role", "USER",
		"--account-type", "INTERNAL",
		"--default-project", "superadmin_personal",
		"--project-role", "MEMBER",
		"--external-id", "external-123",
		"--active=false",
	})
	require.NoError(t, err)

	request, err := resolveCreateUserRequest(cmd)
	require.NoError(t, err)
	require.NotNil(t, request.Password)
	require.NotNil(t, request.AccountRole)
	require.NotNil(t, request.AccountType)
	require.NotNil(t, request.DefaultProject)
	require.NotNil(t, request.ProjectRole)
	require.NotNil(t, request.ExternalId)
	require.NotNil(t, request.Active)

	assert.Equal(t, "newuser", request.Login)
	assert.Equal(t, "new@example.com", request.Email)
	assert.Equal(t, "New User", request.FullName)
	assert.Equal(t, "Secret123!", *request.Password)
	assert.Equal(t, types.CreateUserRQFullAccountRole("USER"), *request.AccountRole)
	assert.Equal(t, types.CreateUserRQFullAccountType("INTERNAL"), *request.AccountType)
	assert.Equal(t, "superadmin_personal", *request.DefaultProject)
	assert.Equal(t, types.CreateUserRQFullProjectRole("MEMBER"), *request.ProjectRole)
	assert.Equal(t, "external-123", *request.ExternalId)
	assert.False(t, *request.Active)
}

func TestBuildEditUserRequest_MergesExistingFullName(t *testing.T) {
	t.Parallel()

	currentFullName := "Existing Name"
	request, err := buildEditUserRequest(&types.UserResource{
		FullName: &currentFullName,
	}, editUserFlags{
		email:  new("updated@example.com"),
		active: new(false),
	})
	require.NoError(t, err)
	require.NotNil(t, request.Email)
	require.NotNil(t, request.Active)

	assert.Equal(t, "Existing Name", request.FullName)
	assert.Equal(t, "updated@example.com", *request.Email)
	assert.False(t, *request.Active)
}

func TestBuildEditUserRequest_RequiresFullNameWhenUnavailable(t *testing.T) {
	t.Parallel()

	_, err := buildEditUserRequest(nil, editUserFlags{
		email: new("updated@example.com"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--full-name")
}

func TestBuildEditUserRequest_RejectsNoChanges(t *testing.T) {
	t.Parallel()

	_, err := buildEditUserRequest(&types.UserResource{}, editUserFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one mutable flag")
}

func TestResolveUserIDByLogin(t *testing.T) {
	t.Parallel()

	lookup := &fakeUserLookup{user: &types.UserResource{Id: 42, UserId: "testuser"}}

	userID, err := resolveUserIDByLogin(context.Background(), lookup, "testuser")
	require.NoError(t, err)

	assert.Equal(t, int64(42), userID)
	assert.Equal(t, "testuser", lookup.gotLogin)
	assert.Equal(t, 1, lookup.lookupCalls)
}

func TestResolveUserIDByLogin_WrapsLookupErrors(t *testing.T) {
	t.Parallel()

	lookup := &fakeUserLookup{err: errors.New("boom")}

	_, err := resolveUserIDByLogin(context.Background(), lookup, "missing-user")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting user missing-user")
	assert.Contains(t, err.Error(), "boom")
}
