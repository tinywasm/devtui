package devtui

import (
	"strings"
	"testing"
	"time"

	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/mcp"
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
	tools := tui.GetMCPTools()

	// Verify tool exists
	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if tool.Name != MCPToolName {
		t.Errorf("Expected tool name '%s', got '%s'", MCPToolName, tool.Name)
	}

	// Verify InputSchema is not empty
	if tool.InputSchema == "" {
		t.Error("Expected tool.InputSchema to be populated")
	}

	if !strings.Contains(tool.InputSchema, "section") {
		t.Errorf("Expected InputSchema to contain 'section', got: %s", tool.InputSchema)
	}

	close(exitChan)
}

func TestMCPGetSectionLogsListsSections(t *testing.T) {
	exitChan := make(chan bool)

	// Capture logger output
	var loggedMessages []string
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
		Color:    DefaultPalette(),
		Logger: func(messages ...any) {
			for _, msg := range messages {
				if str, ok := msg.(string); ok {
					loggedMessages = append(loggedMessages, str)
				}
			}
		},
	})
	tui.SetTestMode(true)

	// Inject MCP logger (simulates what mcpserve does)
	tui.SetLog(func(messages ...any) {
		for _, msg := range messages {
			if str, ok := msg.(string); ok {
				loggedMessages = append(loggedMessages, str)
			}
		}
	})

	tui.NewTabSection("BUILD", "Build Section")
	tui.NewTabSection("DEPLOY", "Deploy Section")

	// Call tool with empty section (should list sections)
	result := tui.mcpGetSectionLogs(GetLogsArgs{Section: ""})

	// Verify result
	if result == nil {
		t.Fatal("Expected non-nil result from mcpGetSectionLogs")
	}

	resultStr, _ := mcp.GetText(result)
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
