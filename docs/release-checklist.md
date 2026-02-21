# Release Checklist

Use this checklist before publishing a new release. Every item should pass
before tagging.

## 1. Unit tests

- [ ] `go test ./...` passes (root module)
- [ ] `cd pkg && go test ./...` passes (pkg module)
- [ ] `cd test/integration && go test ./internal/... -count=1` passes (test helpers)

Or use the Makefile shorthand:

```sh
make test
```

## 2. Lint

- [ ] `golangci-lint-v2 run --fix` reports no remaining issues (root module)

Or:

```sh
make lint
```

## 3. Integration tests

- [ ] `make test-integration` passes with a clean Docker environment
- [ ] All contract tests pass (label: `contract`)
- [ ] All workflow tests pass (label: `workflow`)
- [ ] Coverage profile is generated and reviewed

## 4. Contract and compatibility review

- [ ] `docs/contracts/automation-contracts.md` is current with the codebase
- [ ] `docs/compatibility.md` reflects the tested Report Portal version
- [ ] No contracted JSON field names have been renamed or removed
- [ ] No `omitempty` tags have been added to or removed from contracted fields
- [ ] Error predicates match the documented HTTP status codes
- [ ] Error output format (`{"error": "..."}`) is unchanged

## 5. Documentation

- [ ] `README.md` reflects current command set and usage
- [ ] `ROADMAP.md` is updated if scope has changed
- [ ] All prompt files for included work are moved to `prompts/completed/`

## 6. Build verification

- [ ] `go build ./cmd/rp` produces a working binary
- [ ] Binary runs `rp --help` without error
- [ ] Binary runs `rp whoami` successfully against a live instance

## 7. Go module hygiene

- [ ] `go mod tidy` introduces no changes (root, pkg, test/integration)
- [ ] No unintended dependency upgrades are included
- [ ] `go vet ./...` reports no issues
