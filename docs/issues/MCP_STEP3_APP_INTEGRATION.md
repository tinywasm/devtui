# Step 3: Integrate DevTUI MCP Tools in tinywasm/app

## Objective

Modify `tinywasm/app/start.go` to pass DevTUI as a tool handler to mcpserve, enabling the `devtui_get_section_logs` tool.

## Prerequisites

Complete [Step 2](MCP_STEP2_CREATE_MCP_GO.md) first.

## Current Implementation

In `start.go` (lines 116-123):

```go
toolHandlers := []any{}
if h.wasmClient != nil {
    toolHandlers = append(toolHandlers, h.wasmClient)
}
if h.browser != nil {
    toolHandlers = append(toolHandlers, h.browser)
}
h.mcp = mcpserve.NewHandler(mcpConfig, toolHandlers, h.tui, h.exitChan)
```

Currently `h.tui` is only passed as `TuiInterface` for UI refresh, but not as a tool handler.

## Required Changes

### 1. Add DevTUI to toolHandlers

Modify `start.go` to include the TUI as a tool handler:

```go
toolHandlers := []any{}
if h.wasmClient != nil {
    toolHandlers = append(toolHandlers, h.wasmClient)
}
if h.browser != nil {
    toolHandlers = append(toolHandlers, h.browser)
}
// Add DevTUI as tool handler so its GetMCPToolsMetadata() is discovered
if h.tui != nil {
    toolHandlers = append(toolHandlers, h.tui)
}
h.mcp = mcpserve.NewHandler(mcpConfig, toolHandlers, h.tui, h.exitChan)
```

## Why This Works

The mcpserve reflection system automatically discovers tools via reflection:

```go
func mcpToolsFromHandler(handler any) ([]ToolMetadata, error) {
    handlerValue := reflect.ValueOf(handler)
    method := handlerValue.MethodByName("GetMCPToolsMetadata")
    // ...
}
```

Since DevTUI now has `GetMCPToolsMetadata()` (added in Step 2), the reflection will find it and register the `devtui_get_section_logs` tool.

## Files to Modify

| File | Changes |
|------|---------|
| `start.go` | Add `h.tui` to `toolHandlers` slice |

## Verification

1. Run app tests:
   ```bash
   cd /home/cesar/Dev/Pkg/tinywasm/app && go test ./... -v
   ```

2. Verify code compiles:
   ```bash
   cd /home/cesar/Dev/Pkg/tinywasm/app && go build ./...
   ```

## Completion Checklist

- [ ] Modified `start.go` to include `h.tui` in `toolHandlers`
- [ ] Code compiles without errors
- [ ] All app tests pass
