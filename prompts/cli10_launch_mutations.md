# PR 10: `launch` Mutations

## Goal

Extend the existing `rp launch` command with `update` and `delete` subcommands while leaving the existing `list` and `get` intact.

## Context

This PR runs in the Phase 4 parallel fan-out. PRs 1–5 have landed. After PR 1, the launch resource has been refactored to noun-first with `rp launch list` and `rp launch get <id-or-uuid>`.

Launches are project-scoped. Both `update` and `delete` take a **numeric launch ID** (not UUID). The update payload is small enough for flags but also supports `--from-file` for bulk attribute updates.

### Prerequisites

- PRs 1–5 have landed
- `internal/cli/launch.go` already defines `newLaunchCmd()` with `list` and `get` subcommands

### Client API surface

`LaunchScope` (obtained via `client.Project(projectName).Launches()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `Update` | `Update(ctx, launchID int64, types.UpdateLaunchRQ)` | `*types.OperationCompletionRS` |
| `Delete` | `Delete(ctx, launchID int64)` | `*types.OperationCompletionRS` |

### Relevant types

```go
type UpdateLaunchRQ struct {
    Attributes  *[]ItemAttributeResource `json:"attributes,omitempty"`
    Description *string                  `json:"description,omitempty"`
    Mode        *UpdateLaunchRQMode      `json:"mode,omitempty"`
}
// UpdateLaunchRQMode: "DEBUG", "DEFAULT"

type ItemAttributeResource struct {
    Key   *string `json:"key,omitempty"`
    Value string  `json:"value"`
}

type OperationCompletionRS struct {
    Message *string `json:"message,omitempty"`
}
```

## Implementation plan

### 1. Extend `internal/cli/launch.go`

Add two new command constructors and register them in the existing `newLaunchCmd()` parent. **Do not modify the existing `list` and `get` subcommands or any existing helpers** (filter flags, `parseAttribute`, etc.).

Add to the existing `newLaunchCmd()`:
```go
cmd.AddCommand(newLaunchUpdateCmd())
cmd.AddCommand(newLaunchDeleteCmd())
```

#### `rp launch update <id>`

- Exactly one positional argument: numeric launch ID
- Parse using `parseIDArg(args[0], "launch")`
- **Dual input mode:** flags OR `--from-file`
  - `--from-file <path>` — reads the full `UpdateLaunchRQ` from a JSON file. When provided, flags are ignored.
  - `--description` (string, optional) — updated description
  - `--mode` (string, optional) — updated mode: `"DEBUG"` or `"DEFAULT"`
  - `--attribute` (string array, repeatable) — attributes in `key:value` format (reuses the existing `parseAttribute` helper in `launch.go`)
- When `--from-file` is absent, at least one flag must be provided. Validate this before making the API call.
- Build `types.UpdateLaunchRQ`:
  - `Description`: set from `--description` if provided (use `optionalStringFlag`)
  - `Mode`: set from `--mode` if provided (cast to `types.UpdateLaunchRQMode`)
  - `Attributes`: built from `--attribute` flags via `parseAttribute`, converted to `[]ItemAttributeResource`
- Call `scope.Update(ctx, launchID, request)`
- Table/JSON output: `OperationCompletionRS` via shared message helpers

#### `rp launch delete <id>`

- Exactly one positional argument: numeric launch ID
- Parse using `parseIDArg(args[0], "launch")`
- No additional flags
- Call `scope.Delete(ctx, launchID)`
- Table/JSON output: `OperationCompletionRS` via shared message helpers

### 2. Attribute building helper

Create a helper to convert `--attribute` flag values to `[]ItemAttributeResource`:

```go
func buildAttributeResources(attributes []string) ([]types.ItemAttributeResource, error) {
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

This reuses the existing `parseAttribute` function already in `launch.go`.

### 3. Mode validation

Validate `--mode` input case-insensitively but pass the uppercase value to the API:

```go
func resolveUpdateLaunchMode(cmd *cobra.Command) (*types.UpdateLaunchRQMode, error) {
    if !cmd.Flags().Changed("mode") {
        return nil, nil
    }

    raw, _ := cmd.Flags().GetString("mode")
    normalized := strings.ToUpper(strings.TrimSpace(raw))

    switch normalized {
    case "DEBUG", "DEFAULT":
        mode := types.UpdateLaunchRQMode(normalized)
        return &mode, nil
    default:
        return nil, fmt.Errorf("invalid mode %q: must be DEBUG or DEFAULT", raw)
    }
}
```

### 4. `--from-file` vs flags logic

In the update handler:

```go
if cmd.Flags().Changed("from-file") {
    path, _ := cmd.Flags().GetString("from-file")
    request, err = readRequestFromFile[types.UpdateLaunchRQ](path)
    // proceed with request
} else {
    // Build request from individual flags
    // Validate at least one flag is set
}
```

### 5. No changes to root.go

The `newLaunchCmd()` function is already registered in `root.go` from PR 1.

## Test requirements

### Unit tests (add to `internal/cli/launch_test.go`)

- **ID parsing:** Verify that `rp launch update abc` returns an error (non-numeric)
- **Update flag validation:**
  - No flags and no `--from-file` → error ("at least one of --description, --mode, --attribute, or --from-file must be provided")
  - `--description` alone is valid
  - `--mode` alone is valid
  - `--attribute` alone is valid
  - `--mode` validation: `"DEBUG"` accepted, `"DEFAULT"` accepted, `"debug"` normalized to `"DEBUG"`, `"INVALID"` rejected
- **Attribute building:**
  - Valid `key:value` pairs produce correct `ItemAttributeResource` (verify `Key` is a pointer)
  - Invalid format (no colon) errors
  - Empty key (`:value`) errors
- **`--from-file` mode:**
  - Valid JSON produces correct `UpdateLaunchRQ`
  - File takes precedence when flags are also set (no error, file wins)
- **Delete:** Verify delete takes exactly 1 positional argument
- **Command wiring:** Verify `newLaunchCmd()` now includes update and delete subcommands alongside list and get

## Edge cases and constraints

- **Extend, do not rewrite:** This PR adds `update` and `delete` to the existing launch command. Do NOT modify existing functions (`listLaunches`, `getSingleLaunch`, `launchRow`, launch filter flags, `parseAttribute`, etc.).
- **Numeric ID only:** Both `update` and `delete` take a numeric launch ID, not a UUID. Use `parseIDArg` for validation.
- **`--attribute` reuses existing `parseAttribute`:** The `parseAttribute(attr string) (string, string, bool)` function already exists in `launch.go` for list filter flags. Reuse it for building `ItemAttributeResource` values.
- **`ItemAttributeResource.Key` is `*string`:** When building from `--attribute key:value`, the key must be wrapped in a pointer. The `Value` field is a non-pointer string.
- **Mode is case-insensitive input, uppercase output:** Accept `debug`, `Debug`, `DEBUG` → always send `"DEBUG"` to the API.
- **`--from-file` takes precedence:** When both `--from-file` and individual flags are provided, `--from-file` wins silently. Do not error on the combination.
- **Project flag is already on the parent:** The `-p`/`--project` flag is a `PersistentFlag` on `newLaunchCmd()`, so `update` and `delete` inherit it automatically.
- **Bulk delete is a stretch goal:** The PR plan defers `DeleteBulk` to a later PR. This PR only implements single-launch delete.

## Example commands and expected output

```bash
# Update description
$ rp launch update 42 --description "Updated launch description" -p my_project
MESSAGE
Launch with ID = '42' has been successfully updated.

# Update mode
$ rp launch update 42 --mode DEBUG -p my_project
MESSAGE
Launch with ID = '42' has been successfully updated.

# Update with attributes
$ rp launch update 42 --attribute "env:staging" --attribute "team:backend" -p my_project
MESSAGE
Launch with ID = '42' has been successfully updated.

# Update from file (full payload)
$ rp launch update 42 --from-file launch_update.json -p my_project
MESSAGE
Launch with ID = '42' has been successfully updated.

# Delete
$ rp launch delete 42 -p my_project
MESSAGE
Launch with ID = '42' has been successfully deleted.
```

Example `launch_update.json`:
```json
{
  "description": "Nightly regression run",
  "mode": "DEFAULT",
  "attributes": [
    {"key": "env", "value": "production"},
    {"key": "build", "value": "1.2.3"}
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

- [ ] `rp launch update <id>` with flags (`--description`, `--mode`, `--attribute`) and `--from-file`
- [ ] `rp launch delete <id>` implemented
- [ ] Existing `list` and `get` subcommands unchanged
- [ ] Mode validation (case-insensitive input, reject invalid values)
- [ ] Attribute building reuses existing `parseAttribute`
- [ ] `--from-file` takes precedence over individual flags
- [ ] Unit tests cover flag validation, mode normalization, attribute building, file loading, and command wiring
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
