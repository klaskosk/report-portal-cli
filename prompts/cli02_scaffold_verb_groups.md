# PR 2: Scaffold Missing Top-Level Verb Groups

## Goal

Add empty parent commands for all verb groups that will be needed by later resource PRs. This removes `root.go` as a merge hotspot — future PRs register subcommands within the parent's file rather than editing `root.go`.

## Context

The CLI currently has four verb groups: `get`, `create`, `update`, `delete` (plus the standalone `download` and `whoami` commands). The full CLI plan requires many more verb groups for lifecycle, association, analysis, and administrative flows. Scaffolding them now means later PRs only add `cmd.AddCommand(...)` to the relevant parent file, not to `root.go`.

**Prerequisite:** PR 1 must land first (CUD parent commands split out of `user.go`).

## Verb groups to scaffold

Each verb group gets a dedicated file in `internal/cli/` with a `newXxxCmd()` constructor that returns a `*cobra.Command`. The parent command has no `RunE` — it only serves as a grouping node for future subcommands.

| Verb | File | `Use` | `Short` | Notes |
|------|------|-------|---------|-------|
| `start` | `start.go` | `start` | `Start a resource lifecycle` | For `start launch`, `start item` |
| `finish` | `finish.go` | `finish` | `Finish a resource lifecycle` | For `finish launch`, `finish item` |
| `assign` | `assign.go` | `assign` | `Assign a relationship between resources` | For group↔project, group↔user, project↔user |
| `unassign` | `unassign.go` | `unassign` | `Remove a relationship between resources` | Inverse of `assign` |
| `link` | `link.go` | `link` | `Link external resources` | For item↔external issue |
| `unlink` | `unlink.go` | `unlink` | `Unlink external resources` | Inverse of `link` |
| `analyze` | `analyze.go` | `analyze` | `Run analysis on resources` | For `analyze launch` |
| `compare` | `compare.go` | `compare` | `Compare resources` | For `compare launch` |
| `merge` | `merge_cmd.go` | `merge` | `Merge resources` | For `merge launch`; file named `merge_cmd.go` to avoid confusion |
| `import` | `import_cmd.go` | `import` | `Import resources` | File named `import_cmd.go` because `import` is a Go keyword; function named `newImportCmd` |
| `export` | `export.go` | `export` | `Export resources` | For `export project`, `export user` |
| `exec` | `exec.go` | `exec` | `Execute a command against a resource` | For `exec plugin` |
| `auth` | `auth.go` | `auth` | `Manage authentication and credentials` | For API keys, passwords, invitations |
| `config` | `config_cmd.go` | `config` | `View and manage configuration` | File named `config_cmd.go` to avoid conflict with existing `config.go` (Viper setup) |
| `generate` | `generate.go` | `generate` | `Generate resources` | For `generate demo-data` |

## Implementation

### 1. Create one file per verb group

Each file follows the same minimal pattern. Example for `start.go`:

```go
package cli

import "github.com/spf13/cobra"

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start a resource lifecycle",
	}
}
```

Repeat for all 15 verb groups using the `Use` and `Short` values from the table above.

### 2. Register all new parents in `root.go`

Add `cmd.AddCommand(...)` calls in `newRootCmd()` for each new verb group. Place them after the existing commands, grouped logically:

```go
// Existing commands
cmd.AddCommand(newGetCmd())
cmd.AddCommand(newCreateCmd())
cmd.AddCommand(newUpdateCmd())
cmd.AddCommand(newDeleteCmd())
cmd.AddCommand(newDownloadCmd())
cmd.AddCommand(newWhoamiCmd())

// Lifecycle
cmd.AddCommand(newStartCmd())
cmd.AddCommand(newFinishCmd())

// Associations
cmd.AddCommand(newAssignCmd())
cmd.AddCommand(newUnassignCmd())
cmd.AddCommand(newLinkCmd())
cmd.AddCommand(newUnlinkCmd())

// Analysis and data
cmd.AddCommand(newAnalyzeCmd())
cmd.AddCommand(newCompareCmd())
cmd.AddCommand(newMergeCmd())
cmd.AddCommand(newImportCmd())
cmd.AddCommand(newExportCmd())

// Administration
cmd.AddCommand(newExecCmd())
cmd.AddCommand(newAuthCmd())
cmd.AddCommand(newConfigCmd())
cmd.AddCommand(newGenerateCmd())
```

### 3. Add basic tests

Create a single `verb_scaffolds_test.go` file (or individual test files per verb) that verifies each scaffolded command exists with the expected `Use` string. A table-driven test is preferred:

```go
package cli //nolint:testpackage

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestScaffoldedVerbCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		builder func() *cobra.Command
		use     string
	}{
		{"start", newStartCmd, "start"},
		{"finish", newFinishCmd, "finish"},
		// ... all 15 ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := tt.builder()
			assert.Equal(t, tt.use, cmd.Use)
			assert.NotEmpty(t, cmd.Short)
		})
	}
}
```

Also add a test that verifies the new verbs are actually registered on the root command. This catches a missed `cmd.AddCommand(...)` in `root.go`:

```go
func TestRootCmd_HasScaffoldedVerbs(t *testing.T) {
	t.Parallel()

	root := newRootCmd()
	names := commandNames(root)

	for _, verb := range []string{
		"start", "finish", "assign", "unassign",
		"link", "unlink", "analyze", "compare",
		"merge", "import", "export", "exec",
		"auth", "config", "generate",
	} {
		assert.Contains(t, names, verb, "root should contain %q", verb)
	}
}
```

## Design notes

- **Empty parents are visible in `rp --help`.** This is intentional for this PR. The next PR (PR 3) introduces conditional registration that hides parents with no subcommands. Keeping this PR simple — just add the scaffolds.
- **File naming for Go keyword conflicts.** `import` is a Go keyword, so the file and function use `import_cmd.go` / `newImportCmd`. Same for `config_cmd.go` to avoid confusion with the existing `config.go` that holds Viper logic.
- **No `Long` descriptions yet.** The `Long` field can be filled in when subcommands are added and the verb's scope becomes concrete.
- **`merge_cmd.go` naming.** This avoids potential ambiguity with other merge-related concepts.
- **`config` is provisional.** The exact grouping for settings commands (`config` vs `project-setting` vs something else) is still an open UX decision. Scaffolding the parent now is not a final commitment — it can be renamed or removed before any subcommands land.

## What NOT to do

- **Do not add any subcommands** to the scaffolded parents. They are intentionally empty.
- **Do not modify existing command files** (`get.go`, `create.go`, `update.go`, `delete.go`). Those already exist from PR 1.
- **Do not add resource-specific logic or flags.**

## Validation

```bash
make build
make test
make lint
```

These Makefile targets cover the multi-module layout. Since this PR only touches the root module's CLI code, integration tests are not affected.

Verify all new commands appear in `rp --help` output.

## Definition of done

- [ ] 15 new files created in `internal/cli/`, one per verb group
- [ ] Each file contains a single `newXxxCmd()` function returning a `*cobra.Command`
- [ ] All 15 parents registered in `root.go`
- [ ] Test(s) verify each scaffolded command has correct `Use` and non-empty `Short`
- [ ] Lint and tests pass
- [ ] This file moved to the `completed` directory
