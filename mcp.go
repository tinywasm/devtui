package devtui

import (
	. "github.com/tinywasm/fmt"
)

const (
	MCPToolName = "app_get_logs"
)

// ToolExecutor defines how a tool should be executed
type ToolExecutor func(args map[string]any)

// Name implements Loggable interface for MCP integration
func (d *DevTUI) Name() string {
	return "DEVTUI"
}

// SetLog implements Loggable interface for MCP integration
// This allows mcpserve to inject a capturing logger
func (d *DevTUI) SetLog(log func(message ...any)) {
	// Store in separate field to avoid interfering with TUI's Logger
	d.mcpLogger = log
}

// MCPToolMetadata provides MCP tool configuration metadata.
// Fields must match mcpserve.ToolMetadata for reflection compatibility.
// DevTUI does NOT import mcpserve to maintain decoupling.
type MCPToolMetadata struct {
	Name        string
	Description string
	Parameters  []MCPParameterMetadata
	Execute     ToolExecutor // Changed from 2-param to 1-param signature to match client pattern
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
	// Get available section titles for enum and description
	sectionTitles := d.getSectionTitles()

	// Build dynamic description with available sections
	description := "Get real-time application logs and status from development environment sections. " +
		"Returns current state of compilation, server, assets, browser, and other active components. " +
		"Available sections: "
	for i, title := range sectionTitles {
		if i > 0 {
			description += ", "
		}
		description += "'" + title + "'"
	}
	description += ". Pass empty section parameter to list sections with descriptions."

	return []MCPToolMetadata{
		{
			Name:        MCPToolName,
			Description: description,
			Parameters: []MCPParameterMetadata{
				{
					Name:        "section",
					Description: "Section name to get logs from (e.g., BUILD, DEPLOY). Leave empty to list all available sections.",
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

// mcpGetSectionLogs implements the terminal_logs tool
func (d *DevTUI) mcpGetSectionLogs(args map[string]any) {
	sectionName, _ := args["section"].(string)

	// If no section specified, list available sections
	if sectionName == "" {
		var result string
		result = "Available sections:\n"
		for _, section := range d.TabSections {
			result += Sprintf("- %s\n", section.title)
		}
		if d.mcpLogger != nil {
			d.mcpLogger(result)
		}
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
		if d.mcpLogger != nil {
			d.mcpLogger(Sprintf("Error: Section '%s' not found. Available sections: %v", sectionName, d.getSectionTitles()))
		}
		return
	}

	// Get logs in plain format
	logs := d.getSectionLogsPlain(targetSection)
	if logs == "" {
		if d.mcpLogger != nil {
			d.mcpLogger(Sprintf("Section '%s' has no logs yet.", sectionName))
		}
		return
	}

	if d.mcpLogger != nil {
		d.mcpLogger(logs)
	}
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
