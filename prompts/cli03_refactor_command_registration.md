# PR 3: Refactor Parent Command Registration for Low-Risk Parallelism

## Goal

Centralize subcommand registration into a single function so that (a) parent command files never change when a new resource is added, and (b) scaffolded verb groups with no subcommands are automatically hidden from `rp --help`.

## Context

After PRs 1 and 2, the codebase has ~19 parent command files (`get.go`, `create.go`, ..., `generate.go`). Today, each parent file lists its children via `cmd.AddCommand(...)`. When multiple resource PRs land in parallel, every parent file that gains a child is a merge hotspot — agents must edit the same function in the same file.

This PR introduces a centralized registration model:

1. Parent files become **immutable stubs** — they define the `cobra.Command` and nothing else.
2. A single `commands.go` file calls all `AddCommand` registrations and conditionally attaches parents to root.
3. Scaffolded parents with zero children (e.g., `start`, `finish`) are silently omitted from the root command, so users never see empty verb groups.

After this PR, adding a new resource command (e.g., `rp create filter`) requires exactly two edits:
1. The resource file (`filter.go`) — define `newCreateFilterCmd()`
2. The central registration file (`commands.go`) — add one `createCmd.AddCommand(newCreateFilterCmd())` line

**Prerequisite:** PRs 1 and 2 must land first.

## Implementation

### 1. Strip `AddCommand` calls from parent command files

Each parent file should contain only the command definition. Remove all `cmd.AddCommand(...)` calls.

**Before** (`get.go`):
```go
func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or many resources",
	}

	cmd.AddCommand(newGetLaunchCmd())
	cmd.AddCommand(newGetItemCmd())
	cmd.AddCommand(newGetLogCmd())
	cmd.AddCommand(newGetProjectCmd())
	cmd.AddCommand(newGetUserCmd())

	return cmd
}
```

**After** (`get.go`):
```go
func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Display one or many resources",
	}
}
```

Apply the same transformation to `create.go`, `update.go`, `delete.go`, and all 15 scaffolded verb files from PR 2.

### 2. Create `internal/cli/commands.go`

This file is the single source of truth for which subcommands belong to which parent. It exports one function called from `root.go`.

```go
package cli

import "github.com/spf13/cobra"

// buildCommandTree constructs all parent commands, wires subcommands to parents,
// and attaches only non-empty parents to root. Standalone commands (download,
// whoami) are always attached.
func buildCommandTree(root *cobra.Command) {
	// Build parent commands
	get := newGetCmd()
	create := newCreateCmd()
	update := newUpdateCmd()
	del := newDeleteCmd()
	start := newStartCmd()
	finish := newFinishCmd()
	assign := newAssignCmd()
	unassign := newUnassignCmd()
	link := newLinkCmd()
	unlink := newUnlinkCmd()
	analyze := newAnalyzeCmd()
	compare := newCompareCmd()
	merge := newMergeCmd()
	importCmd := newImportCmd()
	export := newExportCmd()
	exec := newExecCmd()
	auth := newAuthCmd()
	configCmd := newConfigCmd()
	generate := newGenerateCmd()

	// --- Register subcommands ---

	// get
	get.AddCommand(newGetLaunchCmd())
	get.AddCommand(newGetItemCmd())
	get.AddCommand(newGetLogCmd())
	get.AddCommand(newGetProjectCmd())
	get.AddCommand(newGetUserCmd())

	// create
	create.AddCommand(newCreateUserCmd())

	// update
	update.AddCommand(newUpdateUserCmd())

	// delete
	del.AddCommand(newDeleteUserCmd())

	// --- Attach parents to root ---

	// Parents are only attached when they have subcommands.
	// This prevents scaffolded verb groups from appearing in help output
	// before any resource commands are implemented for them.
	verbParents := []*cobra.Command{
		get, create, update, del,
		start, finish,
		assign, unassign,
		link, unlink,
		analyze, compare, merge,
		importCmd, export,
		exec, auth, configCmd, generate,
	}

	for _, parent := range verbParents {
		if parent.HasSubCommands() {
			root.AddCommand(parent)
		}
	}

	// Standalone commands (always visible)
	root.AddCommand(newDownloadCmd())
	root.AddCommand(newWhoamiCmd())
}
```

### 3. Simplify `root.go`

Replace the individual `cmd.AddCommand(...)` calls in `newRootCmd()` with a single call to `buildCommandTree`:

**Before:**
```go
cmd.AddCommand(newGetCmd())
cmd.AddCommand(newCreateCmd())
cmd.AddCommand(newUpdateCmd())
cmd.AddCommand(newDeleteCmd())
cmd.AddCommand(newDownloadCmd())
cmd.AddCommand(newWhoamiCmd())
cmd.AddCommand(newStartCmd())
// ... all 15 scaffolded verbs ...
```

**After:**
```go
buildCommandTree(cmd)
```

`root.go` now has a single registration call and never needs to change when new verbs or resources are added.

### 4. Update tests

#### 4a. Update parent subcommand tests

After this change, calling `newGetCmd()` returns a bare command with no children. Tests that assert subcommand names (e.g., `TestNewCreateCmd_Subcommands`) need to be updated.

**Option A (preferred):** Update these tests to call `buildCommandTree` first, then find the parent and assert its children:

```go
func TestGetCmd_Subcommands(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "rp"}
	buildCommandTree(root)

	get := findSubcommand(root, "get")
	require.NotNil(t, get)
	assert.Equal(t, []string{"item", "launch", "log", "project", "user"}, commandNames(get))
}
```

Where `findSubcommand` is a small test helper:

```go
func findSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, cmd := range parent.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}
```

**Option B:** Test `buildCommandTree` holistically — assert the full set of top-level commands on root and spot-check a few parents for their expected children.

Apply whichever approach produces cleaner tests. The key invariant is: after `buildCommandTree`, `get` has 5 children, `create`/`update`/`delete` each have 1 (`user`), and scaffolded-only parents are absent from root.

#### 4b. Add a test that empty parents are hidden

```go
func TestBuildCommandTree_HidesEmptyParents(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "rp"}
	buildCommandTree(root)

	names := commandNames(root)

	// Verb groups with subcommands should be present
	assert.Contains(t, names, "get")
	assert.Contains(t, names, "create")
	assert.Contains(t, names, "update")
	assert.Contains(t, names, "delete")

	// Standalone commands should be present
	assert.Contains(t, names, "download")
	assert.Contains(t, names, "whoami")

	// Scaffolded-only parents with no children should be absent
	for _, empty := range []string{"start", "finish", "assign", "unassign",
		"link", "unlink", "analyze", "compare", "merge", "import",
		"export", "exec", "auth", "config", "generate"} {
		assert.NotContains(t, names, empty, "%s should be hidden (no subcommands)", empty)
	}
}
```

#### 4c. Move registration tests to `commands_test.go`

The subcommand assertion tests (from `create_test.go`, `update_test.go`, `delete_test.go`, and any in `user_test.go` that test parent command children) should move to `commands_test.go` since they now test `buildCommandTree` behavior.

## Design rationale

**Why a single registration function instead of per-resource `init()` or per-resource registration?**

- The project avoids `init()` functions and global variables (the linter enables `gochecknoglobals` awareness and the codebase is consistently functional).
- A single `buildCommandTree` function is explicit, easy to read, and produces trivial merge conflicts — each `AddCommand` line is independent. Two PRs adding lines to the same block resolve to adjacent, non-conflicting lines.

**Why conditional parent attachment?**

- Without it, `rp --help` would show ~15 empty verb groups that accept no arguments and produce no useful output. This is confusing for users.
- With it, verbs appear in help exactly when their first subcommand lands. No coordinated cleanup needed.

**Why `commands.go` instead of keeping registration in `root.go`?**

- `root.go` handles root command setup, global flags, and the `Execute` entrypoint. Mixing in the full registration tree would bloat it.
- Separating concerns makes `root.go` stable — it should rarely change after this PR.

## What NOT to do

- **Do not introduce `init()` functions or package-level `var` command instances.** Keep the functional constructor pattern.
- **Do not add any new resource commands.** This PR only restructures registration of existing commands.
- **Do not modify the behavior of any existing command.** This is a structural refactor only.

## Validation

```bash
make build
make test
make lint
```

These Makefile targets cover the multi-module layout. Since this PR only touches the root module's CLI code, integration tests are not affected.

Verify:
- `rp --help` shows `get`, `create`, `update`, `delete`, `download`, `whoami` — and **not** the empty scaffolded verbs.
- `rp get --help` still shows `launch`, `item`, `log`, `project`, `user`.
- `rp create --help` still shows `user`.
- All existing command behavior is unchanged.

## Definition of done

- [ ] `internal/cli/commands.go` created with `buildCommandTree`
- [ ] All parent command files (`get.go`, `create.go`, `update.go`, `delete.go`, plus all 15 scaffolded files) stripped down to bare command definitions
- [ ] `root.go` calls `buildCommandTree(cmd)` instead of individual `AddCommand` calls
- [ ] Scaffolded verb groups with no subcommands do not appear in `rp --help`
- [ ] Tests verify subcommand wiring and empty-parent hiding
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
