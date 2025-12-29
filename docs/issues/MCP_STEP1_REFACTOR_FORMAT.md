# Step 1: Refactor formatMessage for Styled/Unstyled Output

## Objective

Modify the message formatting system to support both styled (terminal) and unstyled (MCP) output without code duplication.

## Current Implementation

In `print.go`:

```go
// formatMessage formatea un mensaje según su tipo
func (t *DevTUI) formatMessage(msg tabContent) string {
    // Check if message comes from a readonly field handler (HandlerDisplay)
    if msg.handlerName != "" && t.isReadOnlyHandler(msg.handlerName) {
        return msg.Content
    }

    // Apply message type styling to content (unified for all handler types)
    styledContent := t.applyMessageTypeStyle(msg.Content, msg.Type)  // <-- STYLING HERE

    // Generate timestamp (unified for all handler types that need it)
    timeStr := t.generateTimestamp(msg.Timestamp)  // <-- STYLING HERE

    // ... rest of formatting
    handlerName := t.formatHandlerName(msg.handlerName, msg.handlerColor)  // <-- STYLING HERE
    return Fmt("%s %s%s", timeStr, handlerName, styledContent)
}
```

The styling is applied at three points:
1. `applyMessageTypeStyle()` - Colors content based on message type (error=red, etc.)
2. `generateTimestamp()` - Uses `t.timeStyle.Render()`
3. `formatHandlerName()` - Uses lipgloss styles for colored handler name

## Required Changes

### 1. Add `styled` parameter to `formatMessage`

Modify `formatMessage` to accept a boolean parameter:

```go
// formatMessage formats a message according to its type.
// When styled is false, no ANSI escape codes are added (for MCP/LLM output).
func (t *DevTUI) formatMessage(msg tabContent, styled bool) string {
    // Check if message comes from a readonly field handler (HandlerDisplay)
    if msg.handlerName != "" && t.isReadOnlyHandler(msg.handlerName) {
        return msg.Content
    }

    var content string
    var timeStr string
    var handlerName string

    if styled {
        content = t.applyMessageTypeStyle(msg.Content, msg.Type)
        timeStr = t.generateTimestamp(msg.Timestamp)
        handlerName = t.formatHandlerName(msg.handlerName, msg.handlerColor)
    } else {
        content = msg.Content
        timeStr = t.generateTimestampPlain(msg.Timestamp)
        handlerName = t.formatHandlerNamePlain(msg.handlerName)
    }

    // Check if message comes from interactive handler
    if msg.handlerName != "" && t.isInteractiveHandler(msg.handlerName) {
        return Fmt("%s %s", timeStr, content)
    }

    return Fmt("%s %s%s", timeStr, handlerName, content)
}
```

### 2. Add plain formatting helper methods

Add these new methods in `print.go`:

```go
// generateTimestampPlain returns timestamp without styling
func (t *DevTUI) generateTimestampPlain(timestamp string) string {
    if t.timeProvider != nil && timestamp != "" {
        return t.timeProvider.FormatTime(timestamp)
    }
    return "--:--:--"
}

// formatHandlerNamePlain returns handler name without styling (just padded)
func (t *DevTUI) formatHandlerNamePlain(handlerName string) string {
    if handlerName == "" {
        return ""
    }
    // handlerName already comes padded from createTabContent
    return handlerName + " "
}
```

### 3. Update all callers of formatMessage

Search for all calls to `formatMessage` and update them:

```go
// In view.go ContentView():
formattedMsg := h.formatMessage(content, true)  // styled = true for terminal

// In any future MCP code:
formattedMsg := h.formatMessage(content, false)  // styled = false for LLM
```

## Files to Modify

| File | Changes |
|------|---------|
| `print.go` | Add `styled bool` param to `formatMessage`, add `generateTimestampPlain()`, add `formatHandlerNamePlain()` |
| `view.go` | Update `ContentView()` to pass `styled: true` |

## Verification

1. Run all existing tests:
   ```bash
   cd /home/cesar/Dev/Pkg/tinywasm/devtui && go test ./... -v
   ```

2. All tests must pass - the behavior should be identical when `styled=true`

## Completion Checklist

- [ ] Modified `formatMessage(msg tabContent, styled bool)` signature
- [ ] Added `generateTimestampPlain()` method
- [ ] Added `formatHandlerNamePlain()` method  
- [ ] Updated `ContentView()` to pass `styled: true`
- [ ] All tests pass
