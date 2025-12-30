# Step 4: Enhance terminal_logs MCP Tool

## Objective

Update the `terminal_logs` MCP tool with:
1. Section listing with handler names: `BUILD [WASM, SERVER, ASSETS]`
2. Handler filtering: `handlers: ["WASM", "SERVER"]`
3. Dynamic mode switching: `mode: "compact" | "history"`
4. Auto tab sync: Switch developer's active tab to match LLM's focus

## File: `devtui/mcp.go`

### Update GetMCPToolsMetadata

Add new parameters to the tool:

```go
func (d *DevTUI) GetMCPToolsMetadata() []MCPToolMetadata {
	sectionTitles := d.getSectionTitlesWithHandlers() // NEW: includes handlers

	description := "Get logs from a DevTUI terminal section. Available sections: "
	for i, title := range sectionTitles {
		if i > 0 {
			description += ", "
		}
		description += title
	}
	description += ". Use 'handlers' to filter, 'mode' to switch between compact/history."

	return []MCPToolMetadata{
		{
			Name:        MCPToolName,
			Description: description,
			Parameters: []MCPParameterMetadata{
				{
					Name:        "section",
					Description: "Section title to get logs from. Leave empty to list available sections.",
					Required:    false,
					Type:        "string",
				},
				{
					Name:        "handlers",
					Description: "Filter logs by handler names (e.g., ['WASM', 'SERVER']). Empty = all handlers.",
					Required:    false,
					Type:        "array",
				},
				{
					Name:        "mode",
					Description: "Log display mode: 'compact' (default, clean) or 'history' (full debug). Changes are applied dynamically.",
					Required:    false,
					Type:        "string",
					EnumValues:  []string{"compact", "history"},
					Default:     "compact",
				},
			},
			Execute: d.mcpGetSectionLogs,
		},
	}
}
```

### Add getSectionTitlesWithHandlers Method

```go
// getSectionTitlesWithHandlers returns section titles with their handlers
// Format: "SECTION_NAME [HANDLER1, HANDLER2]"
func (d *DevTUI) getSectionTitlesWithHandlers() []string {
	titles := make([]string, len(d.TabSections))
	for i, section := range d.TabSections {
		handlerNames := d.getSectionHandlerNames(section)
		if len(handlerNames) > 0 {
			titles[i] = Fmt("%s [%s]", section.title, Convert(handlerNames).Join(", ").String())
		} else {
			titles[i] = section.title
		}
	}
	return titles
}

// getSectionHandlerNames returns all handler names in a section
func (d *DevTUI) getSectionHandlerNames(section *tabSection) []string {
	section.mu.RLock()
	defer section.mu.RUnlock()

	names := []string{}
	for _, h := range section.writingHandlers {
		names = append(names, h.Name())
	}
	return names
}
```

### Update mcpGetSectionLogs Method

```go
func (d *DevTUI) mcpGetSectionLogs(args map[string]any, progress chan<- any) {
	sectionName, _ := args["section"].(string)
	handlersArg, _ := args["handlers"].([]any)
	mode, _ := args["mode"].(string)

	// Convert handlers to []string
	var handlers []string
	for _, h := range handlersArg {
		if s, ok := h.(string); ok {
			handlers = append(handlers, s)
		}
	}

	// If no section specified, list available sections
	if sectionName == "" {
		var result string
		result = "Available sections:\n"
		for _, section := range d.TabSections {
			handlerNames := d.getSectionHandlerNames(section)
			if len(handlerNames) > 0 {
				result += Fmt("- %s [%s]\n", section.title, Convert(handlerNames).Join(", ").String())
			} else {
				result += Fmt("- %s\n", section.title)
			}
		}
		progress <- result
		return
	}

	// Find the requested section
	var targetSection *tabSection
	var targetIndex int
	for i, section := range d.TabSections {
		if section.title == sectionName {
			targetSection = section
			targetIndex = i
			break
		}
	}

	if targetSection == nil {
		progress <- Fmt("Error: Section '%s' not found.", sectionName)
		return
	}

	// AUTO SYNC: Switch developer's tab to match LLM's focus
	d.activeTab = targetIndex
	d.RefreshUI()

	// Apply mode if specified
	if mode != "" {
		compact := (mode == "compact")
		d.setHandlersCompactMode(targetSection, handlers, compact)
	}

	// Get logs with optional handler filter
	logs := d.getSectionLogsPlainFiltered(targetSection, handlers)
	if logs == "" {
		progress <- Fmt("Section '%s' has no logs yet.", sectionName)
		return
	}

	progress <- logs
}
```

### Add Helper Methods

```go
// setHandlersCompactMode sets the compact mode for specified handlers (or all if empty)
func (d *DevTUI) setHandlersCompactMode(section *tabSection, handlers []string, compact bool) {
	section.mu.RLock()
	defer section.mu.RUnlock()

	for _, h := range section.writingHandlers {
		if len(handlers) == 0 || contains(handlers, h.Name()) {
			h.SetCompactMode(compact)
		}
	}
}

// getSectionLogsPlainFiltered returns logs filtered by handler names
func (d *DevTUI) getSectionLogsPlainFiltered(section *tabSection, handlers []string) string {
	section.mu.RLock()
	tabContent := make([]tabContent, len(section.tabContents))
	copy(tabContent, section.tabContents)
	section.mu.RUnlock()

	if len(tabContent) == 0 {
		return ""
	}

	var lines []string

	for _, content := range tabContent {
		// Filter by handler if specified
		if len(handlers) > 0 && !contains(handlers, content.RawHandlerName) {
			continue
		}
		formattedMsg := d.formatMessage(content, false)
		lines = append(lines, formattedMsg)
	}

	return Convert(lines).Join("\n").String()
}

// contains checks if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
```

## Verification

```bash
go build ./...
go test ./... -v
```

Test MCP tool manually:
```bash
# List sections with handlers
curl -s -X POST http://localhost:3030/mcp -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"terminal_logs","arguments":{"section":""}},"id":1}'

# Get filtered logs
curl -s -X POST http://localhost:3030/mcp -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"terminal_logs","arguments":{"section":"BUILD","handlers":["WASM"]}},"id":2}'

# Switch to history mode
curl -s -X POST http://localhost:3030/mcp -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"terminal_logs","arguments":{"section":"BUILD","mode":"history"}},"id":3}'
```

## Completion Checklist

- [ ] Added `handlers` and `mode` parameters to tool metadata
- [ ] Added `getSectionTitlesWithHandlers()` method
- [ ] Added `getSectionHandlerNames()` method
- [ ] Added `setHandlersCompactMode()` method
- [ ] Added `getSectionLogsPlainFiltered()` method
- [ ] Updated `mcpGetSectionLogs()` with new logic
- [ ] Added auto tab sync (`d.activeTab = targetIndex`)
- [ ] Code compiles without errors
- [ ] All tests pass
