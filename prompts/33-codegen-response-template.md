# Prompt 33: Codegen Response Template

## Goal

Add a custom oapi-codegen template that generates `GetBody()`, `GetJSONDefault()`, and `GetSuccess()` accessor methods on every generated response struct. This PR is purely additive — no existing client code changes.

## Motivation

Every scope method in `pkg/client` manually destructures generated response structs: extracting `response.Body`, `response.JSONDefault`, and `response.JSON200` (or `JSON201`) to pass into `handleResponse`. These field names vary per struct but follow predictable conventions. By generating uniform accessor methods through a custom template, subsequent PRs can introduce generic helpers that accept any response struct through an interface, collapsing the response-handling boilerplate without weakening the typed wrapper API.

This is the first of a 9-PR series:

| PR | Prompt | Scope |
|----|--------|-------|
| 1 | **33 (this)** | Custom oapi-codegen template |
| 2 | 34 | `typedResponse[T]` / `voidResponse` interfaces and `execute` / `executeVoid` helpers |
| 3 | 35 | Migrate launches + logs (~26 methods + 3 inline rewrites) |
| 4 | 36 | Migrate items, activity, and filters (~36 methods) |
| 5 | 37 | Migrate users, groups, and project links (~40 methods) |
| 6 | 38 | Migrate projects, dashboards, and widgets (~36 methods) |
| 7 | 39 | Migrate settings and integrations (~31 methods) |
| 8 | 40 | Migrate BTS, plugins, files, and misc (~21 methods) |
| 9 | 41 | Remove `handleResponse`, `handleVoidResponse`, `apiResponse`, and dead test code |

The split is intentionally smaller than the original plan. The migration is mechanically simple, but review quality drops quickly once a PR spans too many resource families.

## Background: oapi-codegen custom templates

oapi-codegen v2.5.1 supports template overrides via `output-options.user-template-files` in the YAML config. The relevant template is `client-with-responses.tmpl`, which generates the `*Response` structs and their `Status()` / `StatusCode()` methods. We override it to append additional accessor methods after the existing generation blocks.

Template data available per operation (via `getResponseTypeDefinitions`):

- `ResponseTypeDefinition.TypeName` — the Go field name, e.g. `JSON200`, `JSON201`, `JSONDefault`, `ApplicationXReportportalLaunchV2JSON200`
- `ResponseTypeDefinition.ResponseName` — the status code string: `"200"`, `"201"`, `"default"`, etc.
- `ResponseTypeDefinition.Schema.TypeDecl` — the Go type declaration including package alias, e.g. `externalRef0.GroupPage`

## Implementation plan

### 1) Extract the default template

Copy the `client-with-responses.tmpl` from the oapi-codegen v2.5.1 source (`pkg/codegen/templates/client-with-responses.tmpl`) into `pkg/internal/openapi/templates/client-with-responses.tmpl`. This becomes the base that we extend.

At the top of the copied template, add a short provenance comment that records:

- the upstream source path
- the upstream oapi-codegen version (`v2.5.1`)
- a note that future generator upgrades should diff the local override against the upstream template before carrying changes forward

Keep the local diff focused on the accessor methods. Do not opportunistically reformat or otherwise rewrite the upstream template.

### 2) Append generated accessor methods

At the bottom of the per-operation block (after the existing `StatusCode()` method), add three method generators:

#### `GetBody() []byte`

Always generated for every response struct:

```gotemplate
func (r {{genResponseTypeName $opid | ucFirst}}) GetBody() []byte {
    return r.Body
}
```

#### `GetJSONDefault() *<ErrorType>`

Generated when the response type definitions contain an entry with `ResponseName == "default"`. The return type comes from the schema so the import alias (`externalRef0.ErrorRS` → `types.ErrorRS`) is correct:

```gotemplate
{{range $responseTypeDefinitions}}
{{if eq .ResponseName "default"}}
func (r {{genResponseTypeName $opid | ucFirst}}) GetJSONDefault() *{{.Schema.TypeDecl}} {
    return r.{{.TypeName}}
}
{{end}}
{{end}}
```

Since the overlay adds a default error response to every operation, this will be generated for all ~100 response structs.

#### `GetSuccess() *<SuccessType>`

Generated **only** when there is exactly one 2xx response type definition. Use template variable tracking to detect the single-success case and skip v2 responses (which have multiple 2xx content types):

```gotemplate
{{$successField := ""}}
{{$successTypeDecl := ""}}
{{$multipleSuccess := false}}
{{range $responseTypeDefinitions}}
{{if or (eq .ResponseName "200") (eq .ResponseName "201") (eq .ResponseName "202") (eq .ResponseName "204")}}
    {{if eq $successField ""}}
        {{$successField = .TypeName}}
        {{$successTypeDecl = .Schema.TypeDecl}}
    {{else}}
        {{$multipleSuccess = true}}
    {{end}}
{{end}}
{{end}}

{{if and (ne $successField "") (not $multipleSuccess)}}
func (r {{genResponseTypeName $opid | ucFirst}}) GetSuccess() *{{$successTypeDecl}} {
    return r.{{$successField}}
}
{{end}}
```

### 3) Update `api.config.yaml`

Add the template override:

```yaml
output-options:
  skip-fmt: false
  skip-prune: true
  nullable-type: false
  name-normalizer: ToCamelCaseWithDigits
  overlay:
    path: overlay.openapi.yaml
  user-template-files:
    client-with-responses.tmpl: templates/client-with-responses.tmpl
```

### 4) Regenerate and verify

Run `make generate`, then confirm:

- All ~100 response structs have `GetBody()` and `GetJSONDefault()`
- Standard response structs (single 2xx) also have `GetSuccess()`
- V2 response structs (`GetLaunchByUuidOldTimestampResponse`, `MergeLaunchesOldUuid1Response`, `GetTestItemByUuidV2Response`) do NOT have `GetSuccess()`
- Void response structs (`DeleteGroupResponse`, etc.) do NOT have `GetSuccess()`
- `go build ./...` succeeds
- `make test` passes unchanged

## Edge cases to verify in generated output

| Response shape | `GetSuccess` generated? | Example struct |
|---|---|---|
| Single `JSON200` | Yes | `GetGroupsResponse` |
| Single `JSON201` | Yes | `StartLaunch1Response` |
| No 2xx JSON body (void) | No | `DeleteGroupResponse` |
| Multiple 2xx content types (v2) | No | `GetLaunchByUuidOldTimestampResponse` |
| Binary response (no JSON) | No | `GetLaunchReportResponse` |

## What this PR does NOT change

- No `pkg/client` code is modified
- `handleResponse`, `handleVoidResponse`, `handleV2Response` remain unchanged
- No public API changes
- All existing tests pass unmodified

## Files touched

| File | Change |
|---|---|
| `pkg/internal/openapi/templates/client-with-responses.tmpl` | New — custom template with provenance comment + accessor methods |
| `pkg/internal/openapi/api.config.yaml` | Add `user-template-files` |
| `pkg/internal/api/api.gen.go` | Regenerated — new accessor methods |
