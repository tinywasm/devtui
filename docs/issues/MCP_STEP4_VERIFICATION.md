# Step 4: Verification and Testing

## Objective

Verify that the MCP integration works correctly and all existing tests pass.

## Prerequisites

Complete [Step 3](MCP_STEP3_APP_INTEGRATION.md) first.

## Verification Steps

### 1. Run All DevTUI Tests

```bash
cd /home/cesar/Dev/Pkg/tinywasm/devtui && go test ./... -v
```

Expected: All tests pass

### 2. Run All App Tests  

```bash
cd /home/cesar/Dev/Pkg/tinywasm/app && go test ./... -v
```

Expected: All tests pass

### 3. Verify No ANSI Codes in Plain Output

Create a simple test in `devtui/mcp_test.go`:

```go
package devtui

import (
	"strings"
	"testing"
	"time"
)

func TestGetSectionLogsPlainNoANSI(t *testing.T) {
	exitChan := make(chan bool)
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
		Color:    DefaultPalette(),
		Logger:   func(messages ...any) {},
	})
	tui.SetTestMode(true)

	// Create a test section
	tab := tui.NewTabSection("TEST", "Test Section")
	tabSection := tab.(*tabSection)

	// Add some test content with different message types
	tabSection.addNewContent(Msg.Info, "Info message")
	tabSection.addNewContent(Msg.Error, "Error message")
	tabSection.addNewContent(Msg.Success, "Success message")

	// Wait for messages to be processed
	time.Sleep(100 * time.Millisecond)

	// Get plain logs
	logs := tui.getSectionLogsPlain(tabSection)

	// Verify no ANSI escape codes
	if strings.Contains(logs, "\x1b[") {
		t.Errorf("Plain logs contain ANSI escape codes:\n%s", logs)
	}

	// Verify content is present
	if !strings.Contains(logs, "Info message") {
		t.Error("Plain logs missing 'Info message'")
	}
	if !strings.Contains(logs, "Error message") {
		t.Error("Plain logs missing 'Error message'")
	}
	if !strings.Contains(logs, "Success message") {
		t.Error("Plain logs missing 'Success message'")
	}

	close(exitChan)
}

func TestGetMCPToolsMetadata(t *testing.T) {
	exitChan := make(chan bool)
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
		Color:    DefaultPalette(),
		Logger:   func(messages ...any) {},
	})
	tui.SetTestMode(true)

	// Create sections
	tui.NewTabSection("BUILD", "Build Section")
	tui.NewTabSection("DEPLOY", "Deploy Section")

	// Get MCP tools metadata
	tools := tui.GetMCPToolsMetadata()

	// Verify tool exists
	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if tool.Name != "devtui_get_section_logs" {
		t.Errorf("Expected tool name 'devtui_get_section_logs', got '%s'", tool.Name)
	}

	// Verify parameter has enum values
	if len(tool.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(tool.Parameters))
	}

	param := tool.Parameters[0]
	if param.Name != "section" {
		t.Errorf("Expected parameter name 'section', got '%s'", param.Name)
	}

	// Should have section titles as enum values (SHORTCUTS + BUILD + DEPLOY = 3)
	if len(param.EnumValues) < 3 {
		t.Errorf("Expected at least 3 enum values, got %d: %v", len(param.EnumValues), param.EnumValues)
	}

	close(exitChan)
}

func TestMCPGetSectionLogsListsSections(t *testing.T) {
	exitChan := make(chan bool)
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
		Color:    DefaultPalette(),
		Logger:   func(messages ...any) {},
	})
	tui.SetTestMode(true)

	tui.NewTabSection("BUILD", "Build Section")
	tui.NewTabSection("DEPLOY", "Deploy Section")

	// Call tool with empty section (should list sections)
	progress := make(chan any, 10)
	tui.mcpGetSectionLogs(map[string]any{"section": ""}, progress)

	// Get result
	result := <-progress
	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("Expected string result, got %T", result)
	}

	if !strings.Contains(resultStr, "Available sections:") {
		t.Errorf("Result should list available sections:\n%s", resultStr)
	}
	if !strings.Contains(resultStr, "BUILD") {
		t.Errorf("Result should contain BUILD:\n%s", resultStr)
	}
	if !strings.Contains(resultStr, "DEPLOY") {
		t.Errorf("Result should contain DEPLOY:\n%s", resultStr)
	}

	close(exitChan)
}
```

Run the new test:
```bash
cd /home/cesar/Dev/Pkg/tinywasm/devtui && go test -v -run TestGetSectionLogsPlainNoANSI
cd /home/cesar/Dev/Pkg/tinywasm/devtui && go test -v -run TestGetMCPToolsMetadata
cd /home/cesar/Dev/Pkg/tinywasm/devtui && go test -v -run TestMCPGetSectionLogsListsSections
```

### 4. Manual Verification (Optional)

If you want to manually test the MCP integration:

1. Start the app:
   ```bash
   cd /path/to/your/go/project && tinywasm
   ```

2. Connect to MCP endpoint:
   ```bash
   curl -X POST http://localhost:3030/mcp \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
   ```

3. Verify `devtui_get_section_logs` appears in tools list

4. Call the tool:
   ```bash
   curl -X POST http://localhost:3030/mcp \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"devtui_get_section_logs","arguments":{"section":""}},"id":2}'
   ```

## Completion Checklist

- [ ] All devtui tests pass
- [ ] All app tests pass
- [ ] New MCP tests pass (TestGetSectionLogsPlainNoANSI, TestGetMCPToolsMetadata, TestMCPGetSectionLogsListsSections)
- [ ] No ANSI escape codes in plain output
- [ ] Tool appears in MCP tools list (optional manual verification)

## Final Implementation Summary

After completing all steps:

1. **devtui/print.go**: `formatMessage(msg, styled bool)` supports styled/unstyled output
2. **devtui/view.go**: `ContentViewPlain()` for plain text output
3. **devtui/mcp.go**: MCP tool metadata and implementation
4. **devtui/mcp_test.go**: Tests for MCP functionality
5. **app/start.go**: DevTUI passed as tool handler to mcpserve

The `devtui_get_section_logs` tool is now available to LLMs via MCP!
