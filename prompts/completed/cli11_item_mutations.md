# PR 11: `item` Mutations

## Goal

Extend the existing `rp item` command with `update` and `delete` subcommands while leaving the existing `list` and `get` intact.

## Context

This PR runs in the Phase 4 parallel fan-out. PRs 1–5 have landed. After PR 1, the item resource has been refactored to noun-first with `rp item list` and `rp item get <id-or-uuid>`.

Items are project-scoped. Both `update` and `delete` take a **numeric item ID** (not UUID). The update payload structure mirrors `UpdateLaunchRQ` closely — both have `Attributes`, `Description`, and one additional field (launch has `Mode`, item has `Status`).

### Prerequisites

- PRs 1–5 have landed
- `internal/cli/item.go` already defines `newItemCmd()` with `list` and `get` subcommands

### Client API surface

`TestItemScope` (obtained via `client.Project(projectName).Items()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `Update` | `Update(ctx, itemID int64, types.UpdateTestItemRQ)` | `*types.OperationCompletionRS` |
| `Delete` | `Delete(ctx, itemID int64)` | `*types.OperationCompletionRS` |

### Relevant types

```go
type UpdateTestItemRQ struct {
    Attributes  *[]ItemAttributeResource `json:"attributes,omitempty"`
    Description *string                  `json:"description,omitempty"`
    Status      *string                  `json:"status,omitempty"`
}

type ItemAttributeResource struct {
    Key   *string `json:"key,omitempty"`
    Value string  `json:"value"`
}

type OperationCompletionRS struct {
    Message *string `json:"message,omitempty"`
}
```

## Implementation plan

### 1. Extend `internal/cli/item.go`

Add two new command constructors and register them in the existing `newItemCmd()` parent. **Do not modify the existing `list` and `get` subcommands or any existing helpers** (filter flags, `parseOptionalInt64Flag`, etc.).

Add to the existing `newItemCmd()`:
```go
cmd.AddCommand(newItemUpdateCmd())
cmd.AddCommand(newItemDeleteCmd())
```

#### `rp item update <id>`

- Exactly one positional argument: numeric item ID
- Parse using `parseIDArg(args[0], "item")`
- **Dual input mode:** flags OR `--from-file`
  - `--from-file <path>` — reads the full `UpdateTestItemRQ` from a JSON file. When provided, flags are ignored.
  - `--description` (string, optional) — updated description
  - `--status` (string, optional) — updated status (e.g., `"PASSED"`, `"FAILED"`, `"SKIPPED"`, etc.)
  - `--attribute` (string array, repeatable) — attributes in `key:value` format
- When `--from-file` is absent, at least one flag must be provided.
- Build `types.UpdateTestItemRQ`:
  - `Description`: set from `--description` if provided (use `optionalStringFlag`)
  - `Status`: set from `--status` if provided (use `optionalStringFlag`). Normalize to uppercase.
  - `Attributes`: built from `--attribute` flags, converted to `[]ItemAttributeResource`
- Call `scope.Update(ctx, itemID, request)`
- Table/JSON output: `OperationCompletionRS` via shared message helpers

#### `rp item delete <id>`

- Exactly one positional argument: numeric item ID
- Parse using `parseIDArg(args[0], "item")`
- No additional flags
- Call `scope.Delete(ctx, itemID)`
- Table/JSON output: `OperationCompletionRS` via shared message helpers

### 2. Attribute building

Reuse the `parseAttribute` function from `launch.go` (it's in the same `cli` package). Build a helper to convert `--attribute` flag values to `[]ItemAttributeResource`:

```go
func buildItemAttributeResources(attributes []string) ([]types.ItemAttributeResource, error) {
    resources := make([]types.ItemAttributeResource, 0, len(attributes))
    for _, attr := range attributes {
        key, value, found := parseAttribute(attr)
        if !found {
            return nil, fmt.Errorf("invalid attribute format %q (expected key:value)", attr)
        }

        resources = append(resources, types.ItemAttributeResource{
            Key:   &key,
            Value: value,
        })
    }

    return resources, nil
}
```

**Note:** If PR 10 (launch mutations) has already extracted a shared `buildAttributeResources` helper, use that instead of creating a duplicate. Check the codebase first. If no shared helper exists yet, create this one locally and it can be consolidated in the Phase 5 fan-in (PR 14).

### 3. Status normalization

The `--status` flag accepts case-insensitive input and passes the uppercase value:

```go
func resolveItemStatus(cmd *cobra.Command) *string {
    if !cmd.Flags().Changed("status") {
        return nil
    }

    raw, _ := cmd.Flags().GetString("status")
    normalized := strings.ToUpper(strings.TrimSpace(raw))

    return &normalized
}
```

Do NOT validate status values client-side. The set of valid statuses may vary by deployment, and the API will reject invalid values. Pass the normalized string through.

### 4. `--from-file` vs flags logic

Same pattern as launch update:

```go
if cmd.Flags().Changed("from-file") {
    path, _ := cmd.Flags().GetString("from-file")
    request, err = readRequestFromFile[types.UpdateTestItemRQ](path)
} else {
    // Build request from individual flags
    // Validate at least one flag is set
}
```

### 5. No changes to root.go

The `newItemCmd()` function is already registered in `root.go` from PR 1.

## Test requirements

### Unit tests (add to `internal/cli/item_test.go` or a new section)

- **ID parsing:** Verify that `rp item update abc` returns an error
- **Update flag validation:**
  - No flags and no `--from-file` → error
  - `--description` alone is valid
  - `--status` alone is valid
  - `--attribute` alone is valid
  - `--status` normalization: `"passed"` → `"PASSED"`, `"Failed"` → `"FAILED"`
- **Attribute building:**
  - Valid `key:value` pairs produce correct `ItemAttributeResource`
  - Invalid format errors
- **`--from-file` mode:**
  - Valid JSON produces correct `UpdateTestItemRQ`
  - File takes precedence when flags are also set
- **Delete:** Verify delete takes exactly 1 positional argument
- **Command wiring:** Verify `newItemCmd()` now includes update and delete alongside list and get

## Edge cases and constraints

- **Extend, do not rewrite:** Do NOT modify existing functions (`listItems`, `getSingleItem`, `itemRow`, item filter flags, etc.).
- **Numeric ID only:** Both `update` and `delete` take a numeric item ID, not a UUID.
- **`parseAttribute` is in `launch.go`:** The function is in the same `cli` package, so it's directly accessible. Do not duplicate it.
- **`Status` is a freeform string:** Unlike launch `Mode` which has a closed enum (`DEBUG`/`DEFAULT`), item `Status` is typed as `*string` in the API. Do not validate against a fixed list. Just normalize to uppercase.
- **`--from-file` takes precedence:** Same behavior as launch update.
- **Project flag is already on the parent:** The `-p`/`--project` flag is inherited from `newItemCmd()`.
- **`test-item` alias:** The item parent command has `Aliases: []string{"test-item"}`. The new subcommands are accessible as both `rp item update` and `rp test-item update` automatically.
- **Bulk delete is a stretch goal:** This PR only implements single-item delete.
- **Parallel with PR 10 (launch mutations):** Both PRs add the same pattern (flag-based update with `--from-file` alternative, attribute building). If one lands first, the other should reuse any shared helpers that were extracted. If they run truly in parallel, both will create local helpers that PR 14 (Phase 5 fan-in) can consolidate.

## Example commands and expected output

```bash
# Update description
$ rp item update 100 --description "Updated test item description" -p my_project
MESSAGE
Test item with ID = '100' has been successfully updated.

# Update status
$ rp item update 100 --status PASSED -p my_project
MESSAGE
Test item with ID = '100' has been successfully updated.

# Update with attributes
$ rp item update 100 --attribute "env:prod" --attribute "feature:login" -p my_project
MESSAGE
Test item with ID = '100' has been successfully updated.

# Update from file
$ rp item update 100 --from-file item_update.json -p my_project
MESSAGE
Test item with ID = '100' has been successfully updated.

# Delete
$ rp item delete 100 -p my_project
MESSAGE
Test item with ID = '100' has been successfully deleted.
```

Example `item_update.json`:
```json
{
  "description": "Regression test for login flow",
  "status": "PASSED",
  "attributes": [
    {"key": "env", "value": "staging"},
    {"key": "browser", "value": "chrome"}
  ]
}
```

## Validation

Run before finalizing:

```
make lint
make test
```

## Definition of done

- [ ] `rp item update <id>` with flags (`--description`, `--status`, `--attribute`) and `--from-file`
- [ ] `rp item delete <id>` implemented
- [ ] Existing `list` and `get` subcommands unchanged
- [ ] Status normalized to uppercase (no client-side validation of valid values)
- [ ] Attribute building via `parseAttribute` (reuse or parallel implementation)
- [ ] `--from-file` takes precedence over individual flags
- [ ] Unit tests cover flag validation, status normalization, attribute building, file loading, and command wiring
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
