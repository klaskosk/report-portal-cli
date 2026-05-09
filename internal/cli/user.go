package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// userLookupClient describes the user-lookup behavior needed by delete and update helpers.
type userLookupClient interface {
	GetByLogin(ctx context.Context, login string) (*types.UserResource, error)
}

// userListFilterFlags holds resolved filter flags for user listing.
type userListFilterFlags struct {
	role        *string
	accountType *string
	email       *string
	active      *bool
}

// userListOptions captures the resolved CLI options for listing users.
type userListOptions struct {
	pageSize    int32
	page        int32
	sort        []string
	role        *string
	accountType *string
	email       *string
	active      *bool
}

// createUserFlags holds the raw create-user flags before they are converted to an API request.
type createUserFlags struct {
	login          string
	email          string
	fullName       string
	password       *string
	accountRole    *string
	accountType    *string
	defaultProject *string
	projectRole    *string
	externalID     *string
	active         *bool
}

// editUserFlags holds update-user flag values, using nil to represent omitted flags.
type editUserFlags struct {
	fullName    *string
	email       *string
	accountRole *string
	accountType *string
	externalID  *string
	active      *bool
}

// newUserCmd builds the "user" parent command that groups all user subcommands.
func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage instance users",
	}

	cmd.AddCommand(newUserListCmd())
	cmd.AddCommand(newUserGetCmd())
	cmd.AddCommand(newUserCreateCmd())
	cmd.AddCommand(newUserUpdateCmd())
	cmd.AddCommand(newUserDeleteCmd())

	return cmd
}

// newUserListCmd builds the "user list" subcommand for listing instance users.
func newUserListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List instance users",
		Long: `List instance users.

This command operates at the instance level and does not require --project/-p.

Examples:
  rp user list
  rp user list --role ADMINISTRATOR --active
  rp user list --email admin@example.com -o json`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runUserList),
	}

	registerListFlagsWithDefaults(cmd, listFlagDefaults{sort: nil})
	registerUserListFilterFlags(cmd)

	return cmd
}

// newUserGetCmd builds the "user get" subcommand for retrieving a single user by login.
func newUserGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <login>",
		Short: "Get details of a specific user by login",
		Long: `Get details of a specific user by login.

This command operates at the instance level and does not require --project/-p.

Examples:
  rp user get superadmin
  rp user get superadmin -o json`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runUserGet),
	}
}

// newUserCreateCmd builds the "user create" subcommand.
func newUserCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a user",
		Long: `Create a new instance user.

This command operates at the instance level and does not require --project/-p.
Most user-management endpoints require an administrator account.

Examples:
  rp user create --login jane --email jane@example.com --full-name "Jane Doe"
  rp user create --login jane --email jane@example.com --full-name "Jane Doe" \
    --password 'Secret123!' --account-role USER --project-role MEMBER`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runUserCreate),
	}

	registerCreateUserFlags(cmd)

	return cmd
}

// newUserUpdateCmd builds the "user update" subcommand.
func newUserUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <login>",
		Short: "Update a user",
		Long: `Update an existing instance user by login.

If --full-name is omitted, the command fetches the current user first so it can
preserve the required fullName field in the edit payload.

Examples:
  rp user update jane --email jane.doe@example.com
  rp user update jane --full-name "Jane Doe" --active=false
  rp user update jane --account-role ADMINISTRATOR --account-type INTERNAL`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runUserUpdate),
	}

	registerEditUserFlags(cmd)

	return cmd
}

// newUserDeleteCmd builds the "user delete" subcommand.
func newUserDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <login>",
		Short: "Delete a user",
		Long: `Delete an existing instance user by login.

The underlying API deletes by numeric ID, so this command first resolves the
given login to the current user ID to keep the CLI login-based.

Examples:
  rp user delete jane`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runUserDelete),
	}
}

// runUserList executes the user list command.
func runUserList(cmd *cobra.Command, _ []string) error {
	logger := loggerFromCmd(cmd)
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
	printer := newPrinter(outputFmt)

	conn, err := resolveConnection(cmd)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	clientInstance, err := newClient(conn, logger)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	logger.Debug("listing users")

	return listUsers(cmd, clientInstance, printer)
}

// runUserGet executes the user get command.
func runUserGet(cmd *cobra.Command, args []string) error {
	logger := loggerFromCmd(cmd)
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
	printer := newPrinter(outputFmt)

	conn, err := resolveConnection(cmd)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	clientInstance, err := newClient(conn, logger)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	logger.Debug("getting single user", "login", args[0])

	return getSingleUser(cmd, clientInstance, args[0], printer)
}

// runUserCreate executes the user create command.
func runUserCreate(cmd *cobra.Command, _ []string) error {
	logger := loggerFromCmd(cmd)
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
	printer := newPrinter(outputFmt)

	request, err := resolveCreateUserRequest(cmd)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	conn, err := resolveConnection(cmd)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	clientInstance, err := newClient(conn, logger)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	result, err := clientInstance.Users().Create(cmd.Context(), request)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("creating user %s: %w", request.Login, err)
	}

	headers, row := createUserRow(result)

	return printer.Print(result, headers, [][]string{row})
}

// runUserUpdate executes the user update command.
func runUserUpdate(cmd *cobra.Command, args []string) error {
	logger := loggerFromCmd(cmd)
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
	printer := newPrinter(outputFmt)

	flags := resolveEditUserFlags(cmd)
	if !flags.hasChanges() {
		err := errors.New("at least one mutable flag must be provided to update a user")
		printer.PrintError(err)

		return err
	}

	conn, err := resolveConnection(cmd)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	clientInstance, err := newClient(conn, logger)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	var currentUser *types.UserResource
	if flags.fullName == nil {
		currentUser, err = getUserByLogin(cmd.Context(), clientInstance.Users(), args[0])
		if err != nil {
			printer.PrintError(err)

			return err
		}
	}

	request, err := buildEditUserRequest(currentUser, flags)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	result, err := clientInstance.Users().Edit(cmd.Context(), args[0], request)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("updating user %s: %w", args[0], err)
	}

	return printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// runUserDelete executes the user delete command.
func runUserDelete(cmd *cobra.Command, args []string) error {
	logger := loggerFromCmd(cmd)
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
	printer := newPrinter(outputFmt)

	conn, err := resolveConnection(cmd)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	clientInstance, err := newClient(conn, logger)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	userID, err := resolveUserIDByLogin(cmd.Context(), clientInstance.Users(), args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	result, err := clientInstance.Users().Delete(cmd.Context(), userID)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("deleting user %s: %w", args[0], err)
	}

	return printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// getSingleUser retrieves and prints a single user by login.
func getSingleUser(
	cmd *cobra.Command,
	clientInstance *client.Client,
	login string,
	printer *Printer,
) error {
	user, err := clientInstance.Users().GetByLogin(cmd.Context(), login)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("getting user %s: %w", login, err)
	}

	headers, row := userRow(user)

	return printer.Print(user, headers, [][]string{row})
}

// listUsers retrieves and prints a filtered, paginated list of users.
func listUsers(
	cmd *cobra.Command,
	clientInstance *client.Client,
	printer *Printer,
) error {
	options, err := resolveUserListOptions(cmd)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("invalid user list flags: %w", err)
	}

	result, err := clientInstance.Users().List(cmd.Context(), options.clientOptions()...)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("listing users: %w", err)
	}

	var users []types.UserResource
	if result.Content != nil {
		users = *result.Content
	}

	headers := userHeaders()
	rows := make([][]string, 0, len(users))

	for i := range users {
		_, row := userRow(&users[i])
		rows = append(rows, row)
	}

	return printer.Print(users, headers, rows)
}

// registerUserListFilterFlags adds user-specific list filters to cmd.
func registerUserListFilterFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.String("role", "", "filter by account role")
	flags.String("account-type", "", "filter by account type")
	flags.Bool("active", false, "filter by active status")
	flags.String("email", "", "filter by email address")
}

// resolveUserListFilterFlags reads user-specific list filters from cmd.
func resolveUserListFilterFlags(cmd *cobra.Command) userListFilterFlags {
	return userListFilterFlags{
		role:        optionalStringFlag(cmd, "role"),
		accountType: optionalStringFlag(cmd, "account-type"),
		email:       optionalStringFlag(cmd, "email"),
		active:      optionalBoolFlag(cmd, "active"),
	}
}

// resolveUserListOptions combines standard list flags and user filters into a single request description.
func resolveUserListOptions(cmd *cobra.Command) (userListOptions, error) {
	flags, err := resolveListFlags(cmd)
	if err != nil {
		return userListOptions{}, err
	}

	filterFlags := resolveUserListFilterFlags(cmd)

	options := userListOptions{
		pageSize:    flags.pageSize,
		page:        flags.page,
		sort:        flags.sort,
		role:        filterFlags.role,
		accountType: filterFlags.accountType,
		email:       filterFlags.email,
		active:      filterFlags.active,
	}

	return options, nil
}

// clientOptions converts resolved list options into client.ListUsersOption values.
func (options userListOptions) clientOptions() []client.ListUsersOption {
	resolved := []client.ListUsersOption{
		client.WithUsersPageSize(options.pageSize),
		client.WithUsersPageNumber(options.page),
		client.WithUsersSort(options.sort...),
	}

	if options.role != nil {
		resolved = append(resolved, client.WithUsersRole(*options.role))
	}

	if options.accountType != nil {
		resolved = append(resolved, client.WithUsersType(*options.accountType))
	}

	if options.email != nil {
		resolved = append(resolved, client.WithUsersEmail(*options.email))
	}

	if options.active != nil {
		resolved = append(resolved, client.WithUsersActive(*options.active))
	}

	return resolved
}

// registerCreateUserFlags adds the supported user-create flags to cmd.
func registerCreateUserFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.String("login", "", "login name for the new user")
	flags.String("email", "", "email address for the new user")
	flags.String("full-name", "", "full name for the new user")
	flags.String("password", "", "initial password for the new user")
	flags.String("account-role", "", "instance account role (for example: USER, ADMINISTRATOR)")
	flags.String("account-type", "", "account type (for example: INTERNAL, SCIM)")
	flags.String("default-project", "", "default project for the new user")
	flags.String("project-role", "", "project role within the default project")
	flags.String("external-id", "", "external identifier for the new user")
	flags.Bool("active", false, "set whether the new user is active")
}

// resolveCreateUserRequest converts create-user flags into the API request payload.
func resolveCreateUserRequest(cmd *cobra.Command) (types.CreateUserRQFull, error) {
	flags := createUserFlags{
		login:          mustGetStringFlag(cmd, "login"),
		email:          mustGetStringFlag(cmd, "email"),
		fullName:       mustGetStringFlag(cmd, "full-name"),
		password:       optionalStringFlag(cmd, "password"),
		accountRole:    optionalStringFlag(cmd, "account-role"),
		accountType:    optionalStringFlag(cmd, "account-type"),
		defaultProject: optionalStringFlag(cmd, "default-project"),
		projectRole:    optionalStringFlag(cmd, "project-role"),
		externalID:     optionalStringFlag(cmd, "external-id"),
		active:         optionalBoolFlag(cmd, "active"),
	}

	err := validateRequiredCreateUserFlags(flags)
	if err != nil {
		return types.CreateUserRQFull{}, err
	}

	request := types.CreateUserRQFull{
		Login:    flags.login,
		Email:    flags.email,
		FullName: flags.fullName,
	}
	applyCreateUserOptionalFields(&request, flags)

	return request, nil
}

func validateRequiredCreateUserFlags(flags createUserFlags) error {
	missingFlags := make([]string, 0, 3)
	if flags.login == "" {
		missingFlags = append(missingFlags, "--login")
	}

	if flags.email == "" {
		missingFlags = append(missingFlags, "--email")
	}

	if flags.fullName == "" {
		missingFlags = append(missingFlags, "--full-name")
	}

	if len(missingFlags) == 0 {
		return nil
	}

	return fmt.Errorf(
		"missing required flags: %s",
		strings.Join(missingFlags, ", "),
	)
}

func applyCreateUserOptionalFields(request *types.CreateUserRQFull, flags createUserFlags) {
	if flags.password != nil {
		request.Password = flags.password
	}

	if flags.accountRole != nil {
		accountRole := types.CreateUserRQFullAccountRole(*flags.accountRole)
		request.AccountRole = &accountRole
	}

	if flags.accountType != nil {
		accountType := types.CreateUserRQFullAccountType(*flags.accountType)
		request.AccountType = &accountType
	}

	if flags.defaultProject != nil {
		request.DefaultProject = flags.defaultProject
	}

	if flags.projectRole != nil {
		projectRole := types.CreateUserRQFullProjectRole(*flags.projectRole)
		request.ProjectRole = &projectRole
	}

	if flags.externalID != nil {
		request.ExternalId = flags.externalID
	}

	if flags.active != nil {
		request.Active = flags.active
	}
}

// registerEditUserFlags adds the supported user-update flags to cmd.
func registerEditUserFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.String("full-name", "", "updated full name")
	flags.String("email", "", "updated email address")
	flags.String("account-role", "", "updated account role")
	flags.String("account-type", "", "updated account type")
	flags.String("external-id", "", "updated external identifier")
	flags.Bool("active", false, "updated active status")
}

// resolveEditUserFlags reads update-user flags from cmd while preserving whether each one was explicitly set.
func resolveEditUserFlags(cmd *cobra.Command) editUserFlags {
	return editUserFlags{
		fullName:    optionalStringFlag(cmd, "full-name"),
		email:       optionalStringFlag(cmd, "email"),
		accountRole: optionalStringFlag(cmd, "account-role"),
		accountType: optionalStringFlag(cmd, "account-type"),
		externalID:  optionalStringFlag(cmd, "external-id"),
		active:      optionalBoolFlag(cmd, "active"),
	}
}

// hasChanges reports whether any mutable update flag was explicitly set.
func (flags editUserFlags) hasChanges() bool {
	return flags.fullName != nil ||
		flags.email != nil ||
		flags.accountRole != nil ||
		flags.accountType != nil ||
		flags.externalID != nil ||
		flags.active != nil
}

func resolveEditUserFullName(
	currentUser *types.UserResource,
	flags editUserFlags,
) (string, error) {
	if flags.fullName != nil {
		return *flags.fullName, nil
	}

	if currentUser == nil || currentUser.FullName == nil || *currentUser.FullName == "" {
		return "", errors.New(
			"current user has no full name; specify --full-name explicitly",
		)
	}

	return *currentUser.FullName, nil
}

// buildEditUserRequest converts resolved update flags into an API request, merging the current full name when needed.
func buildEditUserRequest(
	currentUser *types.UserResource,
	flags editUserFlags,
) (types.EditUserRQ, error) {
	if !flags.hasChanges() {
		return types.EditUserRQ{}, errors.New(
			"at least one mutable flag must be provided to update a user",
		)
	}

	fullName, err := resolveEditUserFullName(currentUser, flags)
	if err != nil {
		return types.EditUserRQ{}, err
	}

	request := types.EditUserRQ{FullName: fullName}

	if flags.email != nil {
		request.Email = flags.email
	}

	if flags.accountRole != nil {
		request.Role = flags.accountRole
	}

	if flags.accountType != nil {
		accountType := types.EditUserRQAccountType(*flags.accountType)
		request.AccountType = &accountType
	}

	if flags.externalID != nil {
		request.ExternalId = flags.externalID
	}

	if flags.active != nil {
		request.Active = flags.active
	}

	return request, nil
}

// getUserByLogin resolves a user by login and wraps any error with the login context.
func getUserByLogin(
	ctx context.Context,
	users userLookupClient,
	login string,
) (*types.UserResource, error) {
	user, err := users.GetByLogin(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("getting user %s: %w", login, err)
	}

	return user, nil
}

// resolveUserIDByLogin resolves a user login to the numeric ID required by the delete endpoint.
func resolveUserIDByLogin(
	ctx context.Context,
	users userLookupClient,
	login string,
) (int64, error) {
	user, err := getUserByLogin(ctx, users, login)
	if err != nil {
		return 0, err
	}

	return user.Id, nil
}

// userHeaders returns the table column headers for user output.
func userHeaders() []string {
	return []string{"ID", "LOGIN", "EMAIL", "FULL NAME", "ROLE", "ACTIVE", "ACCOUNT TYPE"}
}

// userRow formats a user resource as a table row, returning both headers and column values.
func userRow(user *types.UserResource) ([]string, []string) {
	return userHeaders(), []string{
		strconv.FormatInt(user.Id, 10),
		user.UserId,
		user.Email,
		formatStringPtr(user.FullName),
		formatStringPtr(user.UserRole),
		formatBoolPtr(user.Active),
		formatStringPtr(user.AccountType),
	}
}

// createUserHeaders returns the table column headers for create-user output.
func createUserHeaders() []string {
	return []string{
		"ID",
		"LOGIN",
		"EMAIL",
		"FULL NAME",
		"ACCOUNT ROLE",
		"ACTIVE",
		"ACCOUNT TYPE",
		"WARNING",
	}
}

// createUserRow formats a create-user response as a table row, returning both headers and column values.
func createUserRow(result *types.CreateUserRS) ([]string, []string) {
	return createUserHeaders(), []string{
		formatInt64Ptr(result.Id),
		formatStringPtr(result.Login),
		formatStringPtr(result.Email),
		formatStringPtr(result.FullName),
		formatStringPtr(result.AccountRole),
		formatBoolPtr(result.Active),
		formatStringPtr(result.AccountType),
		formatStringPtr(result.Warning),
	}
}

// operationResultHeaders returns the table column headers for mutation results that only contain a message.
func operationResultHeaders() []string {
	return []string{"MESSAGE"}
}

// operationResultRows formats an operation-completion response as a single table row.
func operationResultRows(result *types.OperationCompletionRS) [][]string {
	return [][]string{{formatStringPtr(result.Message)}}
}

// formatBoolPtr returns the dereferenced bool as a string or an empty string if nil.
func formatBoolPtr(value *bool) string {
	if value == nil {
		return ""
	}

	return strconv.FormatBool(*value)
}

// mustGetStringFlag returns the string value for name, ignoring impossible Cobra access errors.
func mustGetStringFlag(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)

	return value
}

// optionalStringFlag returns a pointer to the explicit string flag value, or nil when the flag was omitted.
func optionalStringFlag(cmd *cobra.Command, name string) *string {
	if !cmd.Flags().Changed(name) {
		return nil
	}

	value, _ := cmd.Flags().GetString(name)

	return &value
}

// optionalBoolFlag returns a pointer to the explicit bool flag value, or nil when the flag was omitted.
func optionalBoolFlag(cmd *cobra.Command, name string) *bool {
	if !cmd.Flags().Changed(name) {
		return nil
	}

	value, _ := cmd.Flags().GetBool(name)

	return &value
}
