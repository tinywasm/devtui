package devtui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
	. "github.com/tinywasm/fmt"
)

// NEW: sendMessageWithHandler sends a message with handler identification
func (d *DevTUI) sendMessageWithHandler(content string, mt MessageType, tabSection *tabSection, handlerName string, trackingID string, handlerColor string) {
	// trackingID is now the handlerName for automatic tracking
	_, newContent := tabSection.updateOrAddContentWithHandler(mt, content, handlerName, trackingID, handlerColor)

	// Always send to channel to trigger UI update
	// prevent deadlock if channel is full
	select {
	case d.tabContentsChan <- newContent:
	default:
		// channel is full, drop message or handle gracefully
		// maybe trigger a RefreshUI signal instead if possible?
		// for now just don't block
	}
}

// formatMessage formatea un mensaje según su tipo
// When styled is false, no ANSI escape codes are added (for MCP/LLM output).
func (t *DevTUI) formatMessage(msg tabContent, styled bool) string {
	// Check if message comes from a readonly field handler (HandlerDisplay)
	if msg.handlerName != "" && t.isReadOnlyHandler(msg.handlerName) {
		// For readonly fields: no timestamp, cleaner visual content, no special coloring
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

	// short paths
	content = Convert(content).PathShort().String()

	// Check if message comes from interactive handler - clean format with timestamp only
	if msg.handlerName != "" && t.isInteractiveHandler(msg.handlerName) {
		// Interactive handlers: timestamp + content (no handler name for cleaner UX)
		return Fmt("%s %s", timeStr, content)
	}

	// Default format for other handlers (Edit, Execution, Writers)
	// Use already padded handlerName for consistent width
	return Fmt("%s %s%s", timeStr, handlerName, content)
}

// Helper methods to reduce code duplication

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

func (t *DevTUI) applyMessageTypeStyle(content string, msgType MessageType) string {
	switch msgType {
	case Msg.Error:
		return t.errStyle.Render(content)
	case Msg.Warning:
		return t.warnStyle.Render(content)
	case Msg.Info:
		return t.infoStyle.Render(content)
	case Msg.Success:
		return t.successStyle.Render(content)
	default:
		return content
	}
}

func (t *DevTUI) generateTimestamp(timestamp string) string {
	if t.timeProvider != nil && timestamp != "" {
		// FormatTime accepts any (string, int64, etc.) and returns "HH:MM:SS"
		return t.timeStyle.Render(t.timeProvider.FormatTime(timestamp))
	}
	return t.timeStyle.Render("--:--:--")
}

func (t *DevTUI) formatHandlerName(handlerName string, handlerColor string) string {
	if handlerName == "" {
		return ""
	}

	// handlerName already comes padded from createTabContent, no need to pad again

	// Use Primary color if no specific color provided
	color := handlerColor
	if color == "" {
		color = t.Primary // Use palette.Primary as default
	}

	// Create style with handler-specific color as background
	style := lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color(color)).
		Foreground(lipgloss.Color(t.Foreground)) // Use foreground for text contrast

	styledName := style.Render(handlerName)
	// styledName := style.Render(Fmt("[%s]", handlerName))
	return styledName + " "
}

// Helper to detect readonly handlers
func (t *DevTUI) isReadOnlyHandler(handlerName string) bool {
	// Check if handler has empty label (readonly convention)
	for _, tab := range t.TabSections {
		if handler := tab.getWritingHandler(handlerName); handler != nil {
			// Check if it's a display handler (readonly)
			return handler.handlerType == handlerTypeDisplay
		}
	}
	return false
}

// NEW: Helper to detect interactive handlers
func (t *DevTUI) isInteractiveHandler(handlerName string) bool {
	for _, tab := range t.TabSections {
		for _, field := range tab.fieldHandlers {
			if field.handler != nil && field.handler.Name() == handlerName {
				return field.handler.handlerType == handlerTypeInteractive
			}
		}
	}
	return false
}

// createTabContent creates tabContent with unified logic
func (h *DevTUI) createTabContent(content string, mt MessageType, tabSection *tabSection, handlerName string, trackingID string, handlerColor string) tabContent {
	// Timestamp SIEMPRE nuevo usando GetNewID - Handle gracefully if unixid failed to initialize
	var timestamp string
	if h.id != nil {
		timestamp = h.id.GetNewID()
	} else {
		// Log the issue before using fallback
		if h.Logger != nil {
			h.Logger("Warning: unixid not initialized, using fallback timestamp for content: " + content)
		}
		// Graceful fallback when unixid initialization failed
		timestamp = h.timeProvider.FormatTime(time.Now().UnixNano())
	}

	return tabContent{
		Id:             timestamp,
		Timestamp:      timestamp,
		Content:        content,
		Type:           mt,
		tabSection:     tabSection,
		operationID:    nil,
		isProgress:     false,
		isComplete:     false,
		handlerName:    padHandlerName(handlerName, HandlerNameWidth),
		RawHandlerName: handlerName,
		handlerColor:   handlerColor,
	}
}
