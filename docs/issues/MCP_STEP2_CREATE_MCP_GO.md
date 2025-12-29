# Step 2: Create mcp.go with Tool Metadata and Implementation

## Objective

Create `/home/cesar/Dev/Pkg/tinywasm/devtui/mcp.go` that:
1. Defines MCP tool metadata structs (compatible with mcpserve reflection)
2. Implements `GetMCPToolsMetadata()` method on DevTUI
3. Provides the `devtui_get_section_logs` tool logic
4. Adds `ContentViewPlain()` method in view.go

## Prerequisites

Complete [Step 1](MCP_STEP1_REFACTOR_FORMAT.md) first.

## Implementation

### 1. Create `mcp.go`

Create new file `/home/cesar/Dev/Pkg/tinywasm/devtui/mcp.go`:

```go
package devtui

import (
	. "github.com/tinywasm/fmt"
)

// MCPToolMetadata provides MCP tool configuration metadata.
// Fields must match mcpserve.ToolMetadata for reflection compatibility.
// DevTUI does NOT import mcpserve to maintain decoupling.
type MCPToolMetadata struct {
	Name        string
	Description string
	Parameters  []MCPParameterMetadata
	Execute     func(args map[string]any, progress chan<- any)
}

// MCPParameterMetadata describes a tool parameter.
// Fields must match mcpserve.ParameterMetadata for reflection compatibility.
type MCPParameterMetadata struct {
	Name        string
	Description string
	Required    bool
	Type        string // "string", "number", "boolean"
	EnumValues  []string
	Default     any
}

// GetMCPToolsMetadata returns MCP tools provided by DevTUI.
// This method is called via reflection by mcpserve to discover tools.
func (d *DevTUI) GetMCPToolsMetadata() []MCPToolMetadata {
	// Get available section titles for enum
	sectionTitles := d.getSectionTitles()

	return []MCPToolMetadata{
		{
			Name:        "devtui_get_section_logs",
			Description: "Get logs from a specific DevTUI terminal section (tab). Lists available sections when called without parameters or with empty section name.",
			Parameters: []MCPParameterMetadata{
				{
					Name:        "section",
					Description: "Section title to get logs from (e.g., 'BUILD', 'DEPLOY'). Leave empty to list available sections.",
					Required:    false,
					Type:        "string",
					EnumValues:  sectionTitles,
					Default:     "",
				},
			},
			Execute: d.mcpGetSectionLogs,
		},
	}
}

// getSectionTitles returns all registered section titles
func (d *DevTUI) getSectionTitles() []string {
	titles := make([]string, len(d.TabSections))
	for i, section := range d.TabSections {
		titles[i] = section.title
	}
	return titles
}

// mcpGetSectionLogs implements the devtui_get_section_logs tool
func (d *DevTUI) mcpGetSectionLogs(args map[string]any, progress chan<- any) {
	sectionName, _ := args["section"].(string)

	// If no section specified, list available sections
	if sectionName == "" {
		var result string
		result = "Available sections:\n"
		for _, section := range d.TabSections {
			result += Fmt("- %s\n", section.title)
		}
		progress <- result
		return
	}

	// Find the requested section
	var targetSection *tabSection
	for _, section := range d.TabSections {
		if section.title == sectionName {
			targetSection = section
			break
		}
	}

	if targetSection == nil {
		progress <- Fmt("Error: Section '%s' not found. Available sections: %v", sectionName, d.getSectionTitles())
		return
	}

	// Get logs in plain format
	logs := d.getSectionLogsPlain(targetSection)
	if logs == "" {
		progress <- Fmt("Section '%s' has no logs yet.", sectionName)
		return
	}

	progress <- logs
}

// getSectionLogsPlain returns the logs of a section without ANSI styling
func (d *DevTUI) getSectionLogsPlain(section *tabSection) string {
	section.mu.RLock()
	tabContent := make([]tabContent, len(section.tabContents))
	copy(tabContent, section.tabContents)
	section.mu.RUnlock()

	if len(tabContent) == 0 {
		return ""
	}

	var lines []string

	// Add display handler content if available (same as ContentView)
	fieldHandlers := section.fieldHandlers
	if len(fieldHandlers) > 0 && section.indexActiveEditField < len(fieldHandlers) {
		activeField := fieldHandlers[section.indexActiveEditField]
		if activeField.hasContentMethod() {
			displayContent := activeField.getDisplayContent()
			if displayContent != "" {
				lines = append(lines, displayContent)
				if len(tabContent) > 0 {
					lines = append(lines, "")
				}
			}
		}
	}

	// Format messages without styling
	for _, content := range tabContent {
		formattedMsg := d.formatMessage(content, false) // styled = false
		lines = append(lines, formattedMsg)
	}

	return Convert(lines).Join("\n").String()
}
```

### 2. Add ContentViewPlain() to view.go

Add a new method in `view.go` that can be used for testing:

```go
// ContentViewPlain returns the content view without ANSI styling.
// Used for MCP tool output and testing.
func (h *DevTUI) ContentViewPlain() string {
	if len(h.TabSections) == 0 {
		return "No tabs created yet"
	}
	if h.activeTab >= len(h.TabSections) {
		h.activeTab = 0
	}

	section := h.TabSections[h.activeTab]
	return h.getSectionLogsPlain(section)
}
```

## Key Design Decisions

1. **Struct names**: `MCPToolMetadata` and `MCPParameterMetadata` match field names with `mcpserve.ToolMetadata` and `mcpserve.ParameterMetadata` for reflection compatibility

2. **No mcpserve import**: DevTUI defines its own structs. The reflection system in `mcpserve/tools.go` (lines 80-133) converts them automatically

3. **EnumValues**: Section titles are passed as enum values so the LLM knows valid options

4. **Thread safety**: Uses `section.mu.RLock()` when accessing `tabContents`

## Files to Create/Modify

| File | Action | Changes |
|------|--------|---------|
| `mcp.go` | CREATE | Full implementation as shown above |
| `view.go` | MODIFY | Add `ContentViewPlain()` method |

## Verification

1. Run tests:
   ```bash
   cd /home/cesar/Dev/Pkg/tinywasm/devtui && go test ./... -v
   ```

2. Verify code compiles:
   ```bash
   cd /home/cesar/Dev/Pkg/tinywasm/devtui && go build ./...
   ```

## Completion Checklist

- [ ] Created `mcp.go` with all structs and methods
- [ ] Added `ContentViewPlain()` to `view.go`
- [ ] Code compiles without errors
- [ ] All existing tests pass
