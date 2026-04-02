package devtui

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/form/input"
	"github.com/tinywasm/mcp"
)

const (
	MCPToolName = "app_get_logs"
)

// GetMCPTools returns MCP tools provided by DevTUI.
// This method is called by mcpserve to discover tools.
func (d *DevTUI) GetMCPTools() []mcp.Tool {
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

	// Update dynamic schema with section titles
	args := new(GetLogsArgs)
	fields := args.Schema()
	for i := range fields {
		if fields[i].Name == "section" {
			s := input.Select()
			opts := make([]fmt.KeyValue, len(sectionTitles))
			for j, title := range sectionTitles {
				opts[j] = fmt.KeyValue{Key: title, Value: title}
			}
			if setter, ok := s.(interface{ SetOptions(...fmt.KeyValue) }); ok {
				setter.SetOptions(opts...)
			}
			fields[i].Widget = s
			break
		}
	}

	// Manual JSON encoding for []Field as json.Encode only accepts Fielder
	schema := "["
	for i, f := range fields {
		if i > 0 {
			schema += ","
		}
		schema += Sprintf(`{"name":"%s","type":%d`, f.Name, f.Type)
		if f.Widget != nil {
			schema += Sprintf(`,"widget":{"type":"%s"}`, f.Widget.Type())
			if s, ok := f.Widget.(interface{ GetOptions() []fmt.KeyValue }); ok {
				opts := s.GetOptions()
				if len(opts) > 0 {
					schema += `,"options":[`
					for j, opt := range opts {
						if j > 0 {
							schema += ","
						}
						schema += Sprintf(`{"key":"%s","value":"%s"}`, opt.Key, opt.Value)
					}
					schema += "]"
				}
			}
		}
		schema += "}"
	}
	schema += "]"

	return []mcp.Tool{
		{
			Name:        MCPToolName,
			Description: description,
			InputSchema: schema,
			Resource:    "logs",
			Action:      'r',
			Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
				var args GetLogsArgs
				if err := req.Bind(&args); err != nil {
					return nil, err
				}
				return d.mcpGetSectionLogs(args), nil
			},
		},
	}
}

// GetHandlerStates returns nil — DevTUI is a client, not a state server.
func (d *DevTUI) GetHandlerStates() []byte { return nil }

// DispatchAction returns false — actions are forwarded to the daemon, not dispatched locally.
func (d *DevTUI) DispatchAction(_, _ string) bool { return false }

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

// getSectionTitles returns all registered section titles
func (d *DevTUI) getSectionTitles() []string {
	titles := make([]string, len(d.TabSections))
	for i, section := range d.TabSections {
		titles[i] = section.Title
	}
	return titles
}

// mcpGetSectionLogs implements the terminal_logs tool
func (d *DevTUI) mcpGetSectionLogs(args GetLogsArgs) *mcp.Result {
	sectionName := args.Section

	// If no section specified, list available sections
	if sectionName == "" {
		var result string
		result = "Available sections:\n"
		for _, section := range d.TabSections {
			result += Sprintf("- %s\n", section.Title)
		}
		return mcp.Text(result)
	}

	// Find the requested section
	var targetSection *tabSection
	for _, section := range d.TabSections {
		if section.Title == sectionName {
			targetSection = section
			break
		}
	}

	if targetSection == nil {
		return mcp.Text(Sprintf("Error: Section '%s' not found. Available sections: %v", sectionName, d.getSectionTitles()))
	}

	// Get logs in plain format
	logs := d.getSectionLogsPlain(targetSection)
	if logs == "" {
		return mcp.Text(Sprintf("Section '%s' has no logs yet.", sectionName))
	}

	return mcp.Text(logs)
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
	fieldHandlers := section.FieldHandlers
	if len(fieldHandlers) > 0 && section.IndexActiveEditField < len(fieldHandlers) {
		activeField := fieldHandlers[section.IndexActiveEditField]
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
