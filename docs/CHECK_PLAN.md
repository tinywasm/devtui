# Plan: Migrate `devtui` to New `tinywasm/mcp` API

This plan outlines the steps to refactor the MCP implementation in `tinywasm/devtui` to comply with the updated `tinywasm/mcp` API, utilizing `ormc` for automatic JSON Schema generation and validation, and reorganizing tests.

## 1. Prerequisites & Tooling
The new API relies on `ormc` for generating schemas and validation logic from Go structs.

- [ ] Install `ormc`:
  ```bash
  go install github.com/tinywasm/orm/cmd/ormc@latest
  ```

## 2. Model Centralization (`models.go`)
Create a new file `tinywasm/devtui/models.go` to store all tool argument structures.

- [ ] Define argument structures in `models.go` (done).

## 3. Implementation Phases

### Phase 1: MCP Protocol Migration (Server & Client)
Update both the tool definitions (Server) and the daemon communication (Client) to the latest protocol which uses `*github.com/tinywasm/context.Context`.

- [ ] **mcp.go**: 
    - Replace `Parameters: []mcp.Parameter{...}` with `InputSchema: new(GetLogsArgs).Schema()`.
    - Update `Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error)`.
    - Use `req.Bind(&args)` to decode arguments.
- [ ] **remote_handler.go**:
    - Update `postAction` to use `*tinywasm/context.Context` for `client.Dispatch`.
- [ ] **sse_client.go**:
    - Update `fetchAndReconstructState` to use `*tinywasm/context.Context` for `client.Call`.

### Phase 2: Test Reorganization
Organize all existing tests into a dedicated `tests/` directory.

- [ ] Create `tinywasm/devtui/tests/` directory.
- [ ] Move all 35+ `*_test.go` files from root to `tests/`.
- [ ] Update test package names to `devtui_test` (External testing).
- [ ] Fix imports and context usage in tests.

## 4. Verification
- [ ] Run `ormc` in root project.
- [ ] Ensure all code compiles correctly.
- [ ] Run all tests from the new location: `go test ./tests/...`
