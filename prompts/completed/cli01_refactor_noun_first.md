# PR 1: Refactor Existing Resources to Noun-First with Separate list/get

## Goal

Convert the existing CLI from verb-first (`rp get launch`, `rp create user`) to noun-first (`rp launch get`, `rp user create`). Separate the dual-purpose `get` subcommand (which currently handles both listing and single-resource retrieval) into distinct `list` and `get` subcommands for each resource. Add `--json` as a global shortcut for `-o json`.

## Context

The current CLI uses verb-first grouping:
- `rp get launch [id]` — lists or gets a single launch
- `rp get user [login]` — lists or gets a single user
- `rp create user` / `rp update user` / `rp delete user`

The target noun-first structure:
- `rp launch list` / `rp launch get <id>`
- `rp user list` / `rp user get <login>` / `rp user create` / `rp user update` / `rp user delete`

### Files to modify

- `internal/cli/root.go` — remove verb-group registrations, register noun commands directly
- `internal/cli/get.go` — delete entirely (verb-group parent)
- `internal/cli/user.go` — restructure to define `rp user` parent with subcommands
- `internal/cli/launch.go` — restructure to define `rp launch` parent with subcommands
- `internal/cli/item.go` — restructure to define `rp item` parent with subcommands
- `internal/cli/log.go` — restructure to define `rp log` parent with subcommands
- `internal/cli/project.go` — restructure to define `rp project` parent with subcommands
- `internal/cli/download.go` — remains a top-level command (already noun-ish), no changes needed
- `internal/cli/whoami.go` — remains a top-level command, no changes needed
- All test files under `internal/cli/` — update to reflect new command structure

## Detailed requirements

### 1. Noun-first parent commands

Each resource gets a parent command that groups its verb subcommands:

```go
func newLaunchCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "launch",
        Short: "Manage launches",
    }
    cmd.AddCommand(newLaunchListCmd())
    cmd.AddCommand(newLaunchGetCmd())
    return cmd
}
```

Register these in `root.go`:
```go
cmd.AddCommand(newUserCmd())
cmd.AddCommand(newLaunchCmd())
cmd.AddCommand(newItemCmd())
cmd.AddCommand(newLogCmd())
cmd.AddCommand(newProjectCmd())
cmd.AddCommand(newDownloadCmd())  // unchanged
cmd.AddCommand(newWhoamiCmd())    // unchanged
```

### 2. Separate `list` and `get` subcommands

Each resource currently uses zero-vs-one positional argument to switch between list and single-get. Split this into two commands:

**`list` subcommand:**
- `Use: "list"` — no positional arguments
- Accepts pagination flags (`--page`, `--page-size`, `--sort`)
- Accepts resource-specific filter flags
- Returns paginated results

**`get` subcommand:**
- `Use: "get <identifier>"` — exactly one positional argument
- `Args: cobra.ExactArgs(1)`
- For launch/item/log: identifier is numeric ID or UUID (keep existing parse logic)
- For user: identifier is login
- For project: identifier is project name

### 3. User resource — full restructure

The user resource currently has commands split across multiple verb-group parents (`get`, `create`, `update`, `delete`). Consolidate under `rp user`:

- `rp user list` — paginated listing with filter flags
- `rp user get <login>` — single user by login
- `rp user create` — create (flags unchanged)
- `rp user update <login>` — update (flags unchanged)
- `rp user delete <login>` — delete

Remove `newCreateCmd()`, `newUpdateCmd()`, `newDeleteCmd()` from `user.go` — these top-level verb groups are no longer needed.

### 4. `--json` global shortcut

Add a `--json` boolean flag to the root command's persistent flags. When set, it overrides the `-o`/`--output` flag value to `"json"`. Implementation approach:

- Register `--json` as a persistent bool flag on root (default `false`)
- In `PersistentPreRunE` (or at printer creation time), if `--json` is set, treat the output format as `"json"` regardless of what `-o` says
- The simplest implementation: in `PersistentPreRunE`, if `cmd.Flags().Changed("json")` is true, call `cmd.Root().PersistentFlags().Set("output", "json")` to keep a single source of truth

Mark `--json` and `-o json` as mutually exclusive for clarity — if both are set with conflicting values, the behavior should be deterministic (prefer `--json`). However, `--json` with `-o json` (redundant) should not error.

### 5. Delete `get.go`

The file `internal/cli/get.go` defines the parent `get` verb command. Once all resources are registered noun-first, this file is no longer needed. Delete it.

### 6. Project flag (`-p`) placement

Project-scoped resources (launch, item, log) register `-p`/`--project` on their **parent noun command**, so it's inherited by both `list` and `get`:

```go
func newLaunchCmd() *cobra.Command {
    cmd := &cobra.Command{Use: "launch", Short: "Manage launches"}
    cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
    cmd.AddCommand(newLaunchListCmd())
    cmd.AddCommand(newLaunchGetCmd())
    return cmd
}
```

Instance-scoped resources (user, project) do **not** register `-p`.

### 7. Command examples and help text

Update all `Long` descriptions and `Examples` in help text to reflect noun-first paths:
- `rp launch list` instead of `rp get launch`
- `rp launch get 42` instead of `rp get launch 42`
- `rp user create --login ...` instead of `rp create user --login ...`
- etc.

### 8. Aliases

Keep the `test-item` alias on the `item` parent command for discoverability:
```go
cmd := &cobra.Command{Use: "item", Aliases: []string{"test-item"}, ...}
```

## Test requirements

Update all existing tests to reflect the new noun-first structure:

- `launch_test.go` — update any command-path references
- `user_test.go` — update to test commands under `newUserCmd()` parent
- `config_test.go` — verify `resolveProject` still works through noun-first commands
- `listflags_test.go` — should be unaffected (tests helpers directly)

Add new tests:
- Verify `--json` flag sets output to JSON (test that printer receives `"json"` format)
- Verify `rp user list` produces list output (not the old combined behavior)
- Verify `rp user get <login>` requires exactly 1 argument
- Verify `rp launch get` requires exactly 1 argument (previously accepted 0)

## Edge cases and constraints

- **Backward compatibility:** This is a breaking change to the CLI surface. That's intentional — the verb-first pattern is being retired before the CLI has external consumers.
- **`download` and `whoami`:** These remain top-level. `download` is already action-oriented (not a resource CRUD). `whoami` is a standalone introspection command.
- **Project resolution in `get` vs `list`:** Both subcommands need access to the `-p` flag. Using `PersistentFlags` on the parent noun command ensures both inherit it.
- **The `--json` + `-o` interaction:** If a user passes `--json` and `-o table` simultaneously, `--json` wins. Do not error on this combination; just let `--json` take precedence. If a user passes `--json` and `-o json`, that's redundant but not an error.

## Validation

Run before finalizing:

```
make lint
make vet
make test
```

Integration tests (`make test-integration`) should still pass — those invoke through the binary and will need command path updates if they reference verb-first paths. If integration tests exist that reference `rp get launch`, update them too.

## Definition of done

- [ ] `get.go` deleted; no top-level `get`, `create`, `update`, `delete` verb commands remain
- [ ] Each resource (user, launch, item, log, project) has a parent noun command with `list` and `get` subcommands
- [ ] User additionally has `create`, `update`, `delete` subcommands under `rp user`
- [ ] `--json` global flag added and wired as an alias for `-o json`
- [ ] All existing unit tests updated and passing under noun-first structure
- [ ] New tests added for the `list`/`get` split and `--json` flag
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
