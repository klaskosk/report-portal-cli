# Compatibility Notes

This document describes the supported Report Portal version baseline and known
behavior differences that affect the CLI and client library.

## Supported baseline

| Component | Version |
|---|---|
| Report Portal release | **24.1** |
| Service API image | `reportportal/service-api:5.11.0` |
| Service UAT image | `reportportal/service-authorization:5.11.0` |
| REST API version | `/api/v1` |

The integration test suite (`docker-compose.test.yml`) pins all service images
to the versions above. All CLI and client behavior is validated against this
baseline.

## Known version-sensitive behaviors

### Log entry creation (async indexing)

Log entries created via `POST /api/v1/{project}/log/entry` are indexed
asynchronously. After a successful `201 Created` response, the entry may not
appear in listing queries (`GET .../log`) for several hundred milliseconds. The
integration test suite accounts for this with polling (see
`waitForSeedResolution` in `test/integration/helpers_test.go`).

**Guidance**: consumers that create logs and immediately read them back should
poll or add a short delay. This behavior is inherent to Report Portal's
architecture and is expected across versions.

### Attachment binary content processing

Attachments uploaded via multipart log creation require additional async
processing before the `binaryContent.id` field appears in log listing
responses. Processing time varies with server load and can exceed five seconds
in constrained environments.

**Guidance**: poll the log listing endpoint for the `binaryContent` field rather
than assuming it is available immediately after upload. The test suite uses a 30
second polling window with 500ms intervals.

### Launch finish / force-finish terminal status

After calling `PUT .../launch/{id}/finish` or force-finish, the launch status
transitions to one of several terminal values: `PASSED`, `FAILED`, `STOPPED`,
or `INTERRUPTED`. The specific terminal status depends on child item results and
is resolved server-side.

**Guidance**: check for any terminal status rather than expecting a specific one
when verifying that a launch has completed.

### UAT token endpoint routing delay

After the UAT service starts, the Traefik gateway may take several seconds to
register the `/uat/sso/oauth/token` route. Requests during this window receive
connection errors or 404 responses.

**Guidance**: retry token acquisition with a polling interval (the test suite
uses 2 second intervals with a 60 second timeout).

### Project list output format

`rp get project -o json` returns an array of project name strings when listing,
not an array of project objects. This differs from other list commands which
return arrays of resource objects.

**Guidance**: automation consumers should expect `[]string` for project listing
and `object` for single-project detail.

## Fallback and error handling guidance

### Unrecognized HTTP status codes

The client library returns `*UnexpectedStatusError` for any non-2xx response
that does not contain a parseable JSON error body. This commonly occurs when a
reverse proxy or gateway intercedes (returning HTML error pages, for example).

**Guidance**: use `client.HasStatusCode(err, code)` as a general-purpose
predicate. The named predicates (`IsNotFound`, `IsUnauthorized`, etc.) also
match `UnexpectedStatusError` through the shared `statusCoder` interface.

### Missing optional fields

JSON responses may omit optional fields entirely (via `omitempty` struct tags)
rather than returning `null` or zero values. Automation consumers should treat
absent keys as equivalent to the field's zero value.

### Residual risks

- **Untested Report Portal versions**: behavior on versions other than 24.1 /
  5.11.0 is not validated. API changes in newer releases may break assumptions.
- **Analyzer and jobs services**: the integration test stack excludes the
  analyzer, analyzer-train, metrics-gatherer, and jobs services. Behavior that
  depends on these services (e.g., auto-analysis, data cleanup) is not covered.
