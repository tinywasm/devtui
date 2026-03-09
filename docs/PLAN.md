# Plan: tinywasm/devtui — Implement Missing Tests for JSON-RPC Migration

← Requires: `CHECK_PLAN.md` codebase modifications (already executed)

## Development Rules
- **SRP:** Every file must have a single, well-defined purpose.
- **Max 500 lines per file.**
- **No global state.** Use DI via interfaces.
- **Standard library only** in test assertions.
- **Test runner:** `gotest`. **Publish:** `gopush`.
- **Language:** Plans in English, chat in Spanish.
- **No code changes** until the user says "ejecuta" or "ok".

---

## Problem Summary

The source code modifications to migrate REST calls to JSON-RPC 2.0 in `devtui` (as described in `CHECK_PLAN.md`) have been successfully implemented. `go.mod` is correctly updated to `mcp v0.0.17`, and `client_mode_test.go` covers SSE authentication and `Ctrl+C` behavior. 

However, **six specific test cases** outlined in the original Test Strategy are currently missing and need to be implemented to achieve full coverage.

---

## Missing Tests to Implement

You need to add the following test cases to `devtui/client_mode_test.go` or another appropriate test file:

| Test Name | Validates | Description / Expected Behavior |
|-----------|-----------|---------------------------------|
| `TestFetchState_CallsJSONRPCState` | JSON-RPC state fetch | Calling `fetchAndReconstructState()` triggers `tinywasm/state` on the mocked `/mcp` backend and correctly parses the returned `[]StateEntry`. |
| `TestHandleLogEvent_StateRefreshSignal_FetchesState` | State signal triggers refresh | A log event with `HandlerType: 0` inside SSE data correctly triggers a call to `fetchAndReconstructState()`. |
| `TestHandleLogEvent_NormalEntry_NoStateRefresh` | Normal logs are isolated | Data with `HandlerType != 0` is treated as a normal log entry and does not trigger any JSON-RPC calls. |
| `TestPostAction_SendsJSONRPCAction` | Action dispatch via JSON-RPC | `postAction` method correctly formats a JSON-RPC body with the `shortcut` and `value` inside the params array/map. |
| `TestTuiConfig_APIKey_StoredInDevTUI` | Config parsing | Passing `APIKey` in `TuiConfig` successfully maps and is retrievable via `DevTUI.apiKey`. |
| `TestSSEConnect_NoAPIKey_NoAuthHeader` | SSE connection auth behavior | Connecting to SSE with an empty `APIKey` does not attach an `Authorization` header to the HTTP Request. |

---

## Execution Steps

### Step 1 — Verify existing codebase
Ensure `mcp.Client` is correctly mocked in your test files to count executions and validate payloads. 

### Step 2 — Implement tests in `client_mode_test.go` or other matching files
Add the 6 test cases described above. 

### Step 3 — Run tests
```bash
gotest
```
Ensure 100% of tests pass without race conditions.

### Step 4 — Publish
```bash
gopush 'test: implement missing JSON-RPC and SSE auth test cases for devtui'
```
