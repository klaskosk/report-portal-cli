package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// authUserIDSource resolves the current authenticated user's numeric ID.
type authUserIDSource interface {
	GetMyself(ctx context.Context) (*types.UserResource, error)
}

// newAuthCmd builds the "auth" parent command that groups authentication operations.
func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication and identity operations",
	}

	cmd.AddCommand(newAuthAPIKeyCmd())
	cmd.AddCommand(newAuthChangePasswordCmd())
	cmd.AddCommand(newAuthResetPasswordCmd())
	cmd.AddCommand(newAuthInviteCmd())

	return cmd
}

// newAuthAPIKeyCmd builds the "auth apikey" parent command.
func newAuthAPIKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apikey",
		Short: "Manage API keys",
	}

	cmd.AddCommand(newAuthAPIKeyListCmd())
	cmd.AddCommand(newAuthAPIKeyCreateCmd())
	cmd.AddCommand(newAuthAPIKeyDeleteCmd())

	return cmd
}

// newAuthAPIKeyListCmd builds the "auth apikey list" subcommand.
func newAuthAPIKeyListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API keys",
		Long: `List API keys for a user.

This command operates at the instance level and does not require --project/-p.
When --user-id is omitted, keys for the currently authenticated user are listed.
The raw API key value is not shown in list output; it is only returned at creation time.

Examples:
  rp auth apikey list
  rp auth apikey list --user-id 1
  rp auth apikey list -o json`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runAuthAPIKeyList),
	}

	cmd.Flags().Int64("user-id", 0, "target user numeric ID (defaults to current user)")

	return cmd
}

// newAuthAPIKeyCreateCmd builds the "auth apikey create" subcommand.
func newAuthAPIKeyCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an API key",
		Long: `Create an API key for a user.

This command operates at the instance level and does not require --project/-p.
When --user-id is omitted, a key is created for the currently authenticated user.
The raw API key value is only returned once at creation time; store it immediately.

Examples:
  rp auth apikey create --name "ci-token"
  rp auth apikey create --name "dev-token" --user-id 1
  rp auth apikey create --name "ci-token" -o json`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runAuthAPIKeyCreate),
	}

	cmd.Flags().String("name", "", "API key name (required)")
	cmd.Flags().Int64("user-id", 0, "target user numeric ID (defaults to current user)")

	return cmd
}

// newAuthAPIKeyDeleteCmd builds the "auth apikey delete" subcommand.
func newAuthAPIKeyDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <key-id>",
		Short: "Delete an API key",
		Long: `Delete an API key by numeric ID.

This command operates at the instance level and does not require --project/-p.
When --user-id is omitted, the key is deleted for the currently authenticated user.

Examples:
  rp auth apikey delete 3
  rp auth apikey delete 3 --user-id 1`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runAuthAPIKeyDelete),
	}

	cmd.Flags().Int64("user-id", 0, "target user numeric ID (defaults to current user)")

	return cmd
}

// newAuthChangePasswordCmd builds the "auth change-password" subcommand.
func newAuthChangePasswordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change-password",
		Short: "Change the current user's password",
		Long: `Change the password for the currently authenticated user.

This command operates at the instance level and does not require --project/-p.
Passwords are passed via flags and will be visible in shell history.

Examples:
  rp auth change-password --old-password "old123" --new-password "new456"`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runAuthChangePassword),
	}

	cmd.Flags().String("old-password", "", "current password (required)")
	cmd.Flags().String("new-password", "", "new password (required)")

	return cmd
}

// newAuthResetPasswordCmd builds the "auth reset-password" subcommand.
func newAuthResetPasswordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-password",
		Short: "Initiate a password reset by email",
		Long: `Send a password reset link to the given email address.

This command operates at the instance level and does not require --project/-p.
It initiates the password reset flow; it does not complete the reset.

Examples:
  rp auth reset-password --email user@example.com`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runAuthResetPassword),
	}

	cmd.Flags().String("email", "", "email address for the account to reset (required)")

	return cmd
}

// newAuthInviteCmd builds the "auth invite" subcommand.
func newAuthInviteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invite",
		Short: "Invite a new user",
		Long: `Create an invitation for a new user.

This command operates at the instance level and does not require --project/-p.

Examples:
  rp auth invite --email newuser@example.com
  rp auth invite --email newuser@example.com --default-project my_project --role USER`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runAuthInvite),
	}

	cmd.Flags().String("email", "", "email address for the invited user (required)")
	cmd.Flags().String("default-project", "", "default project for the invited user")
	cmd.Flags().String("role", "", "role for the invited user")

	return cmd
}

func runAuthAPIKeyList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	userID, err := resolveAuthUserID(cmd, deps.client.Users())
	if err != nil {
		deps.printer.PrintError(err)

		return err
	}

	keys, err := deps.client.Users().ListAPIKeys(cmd.Context(), userID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("listing api keys: %w", err)
	}

	headers := apiKeyListHeaders()

	rows := make([][]string, 0, len(keys))
	for i := range keys {
		rows = append(rows, apiKeyListRow(&keys[i]))
	}

	return deps.printer.Print(keys, headers, rows)
}

func runAuthAPIKeyCreate(cmd *cobra.Command, _ []string) error {
	name := mustGetStringFlag(cmd, "name")
	if name == "" {
		err := errors.New("missing required flags: --name")
		resolvePrinter(cmd).PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	userID, err := resolveAuthUserID(cmd, deps.client.Users())
	if err != nil {
		deps.printer.PrintError(err)

		return err
	}

	key, err := deps.client.Users().CreateAPIKey(cmd.Context(), userID, types.ApiKeyRQ{Name: name})
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("creating api key: %w", err)
	}

	headers, rows := apiKeyCreateRow(key)

	return deps.printer.Print(key, headers, rows)
}

func runAuthAPIKeyDelete(cmd *cobra.Command, args []string) error {
	keyID, err := parseIDArg(args[0], "api key")
	if err != nil {
		resolvePrinter(cmd).PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	userID, err := resolveAuthUserID(cmd, deps.client.Users())
	if err != nil {
		deps.printer.PrintError(err)

		return err
	}

	result, err := deps.client.Users().DeleteAPIKey(cmd.Context(), userID, keyID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("deleting api key: %w", err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func runAuthChangePassword(cmd *cobra.Command, _ []string) error {
	oldPassword, newPassword, err := resolveChangePasswordFlags(cmd)
	if err != nil {
		resolvePrinter(cmd).PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	result, err := deps.client.Users().ChangePassword(cmd.Context(), types.ChangePasswordRQ{
		OldPassword: oldPassword,
		NewPassword: &newPassword,
	})
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("changing password: %w", err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func runAuthResetPassword(cmd *cobra.Command, _ []string) error {
	email := mustGetStringFlag(cmd, "email")
	if email == "" {
		err := errors.New("missing required flags: --email")
		resolvePrinter(cmd).PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	result, err := deps.client.Users().
		RestorePassword(cmd.Context(), types.RestorePasswordRQ{Email: email})
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("resetting password: %w", err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func runAuthInvite(cmd *cobra.Command, _ []string) error {
	request, err := resolveInviteRequest(cmd)
	if err != nil {
		resolvePrinter(cmd).PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	result, err := deps.client.Users().CreateInvitation(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("creating invitation: %w", err)
	}

	headers, rows := inviteResultRow(result)

	return deps.printer.Print(result, headers, rows)
}

func resolveAuthUserID(cmd *cobra.Command, users authUserIDSource) (int64, error) {
	if cmd.Flags().Changed("user-id") {
		id, _ := cmd.Flags().GetInt64("user-id")

		return id, nil
	}

	myself, err := users.GetMyself(cmd.Context())
	if err != nil {
		return 0, fmt.Errorf("resolving current user: %w", err)
	}

	if myself.Id == 0 {
		return 0, errors.New("current user has no ID")
	}

	return myself.Id, nil
}

func resolveChangePasswordFlags(cmd *cobra.Command) (string, string, error) {
	oldPassword := mustGetStringFlag(cmd, "old-password")
	newPassword := mustGetStringFlag(cmd, "new-password")

	missing := make([]string, 0, 2)
	if oldPassword == "" {
		missing = append(missing, "--old-password")
	}

	if newPassword == "" {
		missing = append(missing, "--new-password")
	}

	if len(missing) > 0 {
		return "", "", fmt.Errorf("missing required flags: %s", strings.Join(missing, ", "))
	}

	return oldPassword, newPassword, nil
}

func resolveInviteRequest(cmd *cobra.Command) (types.CreateUserRQ, error) {
	email := mustGetStringFlag(cmd, "email")
	if email == "" {
		return types.CreateUserRQ{}, errors.New("missing required flags: --email")
	}

	return types.CreateUserRQ{
		Email:          email,
		DefaultProject: optionalStringFlag(cmd, "default-project"),
		Role:           optionalStringFlag(cmd, "role"),
	}, nil
}

func apiKeyListHeaders() []string {
	return []string{"ID", "NAME", "CREATED AT"}
}

func apiKeyListRow(key *types.ApiKeyRS) []string {
	return []string{
		strconv.FormatInt(key.Id, 10),
		key.Name,
		key.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func apiKeyCreateRow(key *types.ApiKeyRS) ([]string, [][]string) {
	return []string{"ID", "NAME", "API KEY"}, [][]string{{
		strconv.FormatInt(key.Id, 10),
		key.Name,
		formatOptional(key.ApiKey),
	}}
}

func inviteResultRow(result *types.CreateUserBidRS) ([]string, [][]string) {
	return []string{"BID", "MESSAGE"}, [][]string{{
		formatOptional(result.Bid),
		formatOptional(result.Message),
	}}
}
