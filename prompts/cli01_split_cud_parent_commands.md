# PR 1: Split Parent CUD Command Builders Out of `user.go`

## Goal

Move the top-level `create`, `update`, and `delete` parent command constructors out of `internal/cli/user.go` into their own files (`create.go`, `update.go`, `delete.go`), mirroring the existing `get.go` pattern. This eliminates `user.go` as a merge hotspot for future resource PRs.

## Context

Today, `user.go` owns both the user-specific command logic and the parent command constructors (`newCreateCmd`, `newUpdateCmd`, `newDeleteCmd`). Every future resource that adds CUD commands would need to touch `user.go` to add its `cmd.AddCommand(...)` call. By extracting the parents into dedicated files, future resource PRs only touch the small parent file — not the 788-line `user.go`.

The `get` parent already lives in its own file (`get.go`). This PR brings `create`, `update`, and `delete` to parity.

## Scope

This is a pure code-motion refactor. No behavior changes, no new commands, no new tests beyond updating existing ones.

## Implementation

### 1. Create `internal/cli/create.go`

Move `newCreateCmd()` (currently at lines 87–97 of `user.go`) into a new file. The function stays identical:

```go
package cli

import "github.com/spf13/cobra"

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create resources",
	}

	cmd.AddCommand(newCreateUserCmd())

	return cmd
}
```

### 2. Create `internal/cli/update.go`

Move `newUpdateCmd()` (currently at lines 99–109 of `user.go`) into a new file:

```go
package cli

import "github.com/spf13/cobra"

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update resources",
	}

	cmd.AddCommand(newUpdateUserCmd())

	return cmd
}
```

### 3. Create `internal/cli/delete.go`

Move `newDeleteCmd()` (currently at lines 111–121 of `user.go`) into a new file:

```go
package cli

import "github.com/spf13/cobra"

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete resources",
	}

	cmd.AddCommand(newDeleteUserCmd())

	return cmd
}
```

### 4. Remove the three functions from `user.go`

Delete the `newCreateCmd`, `newUpdateCmd`, and `newDeleteCmd` function definitions from `user.go`. No other changes to `user.go` are needed — `newCreateUserCmd`, `newUpdateUserCmd`, `newDeleteUserCmd` and all helpers stay in `user.go`.

### 5. Move corresponding tests

Move the following tests from `user_test.go` to new test files alongside their parent:

| Test | From | To |
|------|------|----|
| `TestNewCreateCmd_Subcommands` | `user_test.go` | `create_test.go` |
| `TestNewUpdateCmd_Subcommands` | `user_test.go` | `update_test.go` |
| `TestNewDeleteCmd_Subcommands` | `user_test.go` | `delete_test.go` |

Each new test file should use `package cli` (white-box, same as `user_test.go`). The `commandNames` helper is defined in `user_test.go` and can stay there — it's accessible to all test files in the same package. If the linter complains about it being in `user_test.go` while used from other test files, move `commandNames` to a shared `test_helpers_test.go` file.

Preserve the `//nolint:testpackage` directive at the top of each new test file.

## What NOT to do

- **Do not move shared helpers** like `operationResultHeaders`, `operationResultRows`, `formatBoolPtr`, `optionalStringFlag`, `optionalBoolFlag`, `mustGetStringFlag` out of `user.go` yet. Those are addressed in a later PR (shared helper extraction after the pilot slices).
- **Do not change `root.go`**. It already calls `newCreateCmd()`, `newUpdateCmd()`, `newDeleteCmd()` — those calls are unaffected by moving the functions to new files.
- **Do not add any new commands or functionality**.

## Validation

Run before finalizing:

```bash
make build
make test
make lint
```

These Makefile targets cover the multi-module layout (root, `pkg/`, `test/integration/internal/`). Since this PR only touches the root module's CLI code, `make test` and `make lint` are sufficient — integration tests are not affected.

Verify `rp create --help`, `rp update --help`, and `rp delete --help` still show the `user` subcommand.

## Definition of done

- [ ] `internal/cli/create.go` exists with `newCreateCmd()` moved from `user.go`
- [ ] `internal/cli/update.go` exists with `newUpdateCmd()` moved from `user.go`
- [ ] `internal/cli/delete.go` exists with `newDeleteCmd()` moved from `user.go`
- [ ] `user.go` no longer contains `newCreateCmd`, `newUpdateCmd`, or `newDeleteCmd`
- [ ] Tests for parent command subcommand assertions moved to `create_test.go`, `update_test.go`, `delete_test.go`
- [ ] All existing tests pass unchanged
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
