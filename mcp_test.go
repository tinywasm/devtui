package devtui

import (
	"strings"
	"testing"
	"time"

	. "github.com/tinywasm/fmt"
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
	if tool.Name != MCPToolName {
		t.Errorf("Expected tool name '%s', got '%s'", MCPToolName, tool.Name)
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
