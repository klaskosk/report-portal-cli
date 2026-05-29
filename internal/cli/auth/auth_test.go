package auth //nolint:testpackage // white-box tests

import (
	"context"
	"testing"
	"time"

	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAuthUserIDSource implements authUserIDSource for helper tests.
type fakeAuthUserIDSource struct {
	user *types.UserResource
	err  error
}

func (f *fakeAuthUserIDSource) GetMyself(_ context.Context) (*types.UserResource, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.user, nil
}

func TestNewAuthCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()

	assert.Equal(
		t,
		[]string{"apikey", "change-password", "invite", "reset-password"},
		testutil.CommandNames(cmd),
	)

	apiKeyCmd, _, err := cmd.Find([]string{"apikey"})
	require.NoError(t, err)
	assert.Equal(t, []string{"create", "delete", "list"}, testutil.CommandNames(apiKeyCmd))
}

func TestNewAuthAPIKeyListCmd_UserIDFlagOptional(t *testing.T) {
	t.Parallel()

	cmd := newAuthAPIKeyListCmd()

	require.NoError(t, cmd.ParseFlags(nil), "--user-id should be optional")

	flag := cmd.Flags().Lookup("user-id")
	require.NotNil(t, flag, "--user-id flag should exist")
}

func TestResolveAuthUserID_explicitUserID(t *testing.T) {
	t.Parallel()

	cmd := newAuthAPIKeyListCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--user-id", "42"}))

	id, err := resolveAuthUserID(cmd, &fakeAuthUserIDSource{})
	require.NoError(t, err)
	assert.EqualValues(t, 42, id)
}

func TestResolveAuthUserID_currentUser(t *testing.T) {
	t.Parallel()

	cmd := newAuthAPIKeyListCmd()
	source := &fakeAuthUserIDSource{
		user: &types.UserResource{Id: 7},
	}

	id, err := resolveAuthUserID(cmd, source)
	require.NoError(t, err)
	assert.EqualValues(t, 7, id)
}

func TestResolveAuthUserID_getMyselfError(t *testing.T) {
	t.Parallel()

	cmd := newAuthAPIKeyListCmd()
	source := &fakeAuthUserIDSource{err: assert.AnError}

	_, err := resolveAuthUserID(cmd, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolving current user")
}

func TestResolveAuthUserID_noUserID(t *testing.T) {
	t.Parallel()

	cmd := newAuthAPIKeyListCmd()
	source := &fakeAuthUserIDSource{
		user: &types.UserResource{Id: 0},
	}

	_, err := resolveAuthUserID(cmd, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "current user has no ID")
}

func TestRunAuthAPIKeyCreate_requiresName(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newAuthAPIKeyCreateCmd())

	err := runAuthAPIKeyCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestNewAuthAPIKeyDeleteCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	cmd := newAuthAPIKeyDeleteCmd()

	require.Error(t, cmd.Args(cmd, []string{}), "should reject zero args")
	require.NoError(t, cmd.Args(cmd, []string{"3"}), "should accept exactly one arg")
	require.Error(t, cmd.Args(cmd, []string{"1", "2"}), "should reject two args")
}

func TestRunAuthAPIKeyDelete_invalidKeyID(t *testing.T) {
	t.Parallel()

	cmd := testutil.AttachCmdToRoot(t, newAuthAPIKeyDeleteCmd())

	err := runAuthAPIKeyDelete(cmd, []string{"abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid api key ID")
}

func TestRunAuthChangePassword_requiresBothPasswordFlags(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newAuthChangePasswordCmd())

	err := runAuthChangePassword(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--old-password")
	assert.Contains(t, err.Error(), "--new-password")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunAuthResetPassword_requiresEmail(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := testutil.AttachCmdToRoot(t, newAuthResetPasswordCmd())

	err := runAuthResetPassword(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--email")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestResolveInviteRequest_requiresEmail(t *testing.T) {
	t.Parallel()

	cmd := newAuthInviteCmd()

	_, err := resolveInviteRequest(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--email")
}

func TestResolveInviteRequest_optionalProjectAndRole(t *testing.T) {
	t.Parallel()

	cmd := newAuthInviteCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--email", "new@example.com",
	}))

	request, err := resolveInviteRequest(cmd)
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", request.Email)
	assert.Nil(t, request.DefaultProject)
	assert.Nil(t, request.Role)

	require.NoError(t, cmd.ParseFlags([]string{
		"--email", "new@example.com",
		"--default-project", "my_project",
		"--role", "USER",
	}))

	request, err = resolveInviteRequest(cmd)
	require.NoError(t, err)
	require.NotNil(t, request.DefaultProject)
	assert.Equal(t, "my_project", *request.DefaultProject)
	require.NotNil(t, request.Role)
	assert.Equal(t, "USER", *request.Role)
}

func TestApiKeyListRow(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	key := &types.ApiKeyRS{
		Id:        1,
		Name:      "ci-token",
		CreatedAt: createdAt,
	}

	assert.Equal(t, []string{"ID", "NAME", "CREATED AT"}, apiKeyListHeaders())
	assert.Equal(t, []string{"1", "ci-token", "2024-01-15T10:30:00Z"}, apiKeyListRow(key))
}

func TestApiKeyCreateRow(t *testing.T) {
	t.Parallel()

	apiKey := "rp_abc123"
	key := &types.ApiKeyRS{
		Id:     3,
		Name:   "new-ci-key",
		ApiKey: &apiKey,
	}

	headers, rows := apiKeyCreateRow(key)
	assert.Equal(t, []string{"ID", "NAME", "API KEY"}, headers)
	assert.Equal(t, []string{"3", "new-ci-key", "rp_abc123"}, rows[0])
}

func TestInviteResultRow(t *testing.T) {
	t.Parallel()

	bid := "abc-123"
	message := "Invitation has been sent."
	result := &types.CreateUserBidRS{
		Bid:     &bid,
		Message: &message,
	}

	headers, rows := inviteResultRow(result)
	assert.Equal(t, []string{"BID", "MESSAGE"}, headers)
	assert.Equal(t, []string{"abc-123", "Invitation has been sent."}, rows[0])
}
