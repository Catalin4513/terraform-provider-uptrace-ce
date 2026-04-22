# Remove Provider-Specific Parse Helpers

## Goal

Remove the remaining provider-specific `Parse*` ID helper so org and project resources follow the same convention: parse resource IDs inline at the call site with the concrete width required by the API.

## Scope

- Delete `client.ParseProjectID`.
- Inline `strconv.ParseUint(..., 10, 32)` in `internal/services/project/project_resource.go`.
- Replace helper-only unit tests with resource-level tests that exercise invalid project IDs through the resource methods that now own the parsing.
- Update stale documentation that still mentions deleted parse helpers.

## Non-Goals

- Do not remove generic import parsers from `internal/tfutil`.
- Do not introduce a new shared helper for org/project IDs.
- Do not change monitor, notification channel, or token resource parsing outside the direct impact of deleting `client.ParseProjectID`.

## Design

`org_resource.go` already parses `uint64` org IDs inline with `strconv.ParseUint`. `project_resource.go` should match that pattern and parse `uint32` project IDs inline in `Read`, `Update`, and `Delete`.

The helper deletion slightly shifts test ownership. Instead of testing `client.ParseProjectID` in isolation, unit tests should cover the resource behavior that depends on ID parsing by verifying that invalid state or plan IDs produce diagnostics in the affected lifecycle methods.

## Verification

- Run the focused project unit tests after the red/green cycle.
- Run the broader Go test suite for the touched packages if the focused tests pass cleanly.
