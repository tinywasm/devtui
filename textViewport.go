package devtui

// TextViewport manages horizontal scrolling for text that exceeds visible width
type TextViewport struct {
	viewStart int // First visible character (rune index)
}

// CalculateVisibleWindow returns the visible portion of text based on cursor and width.
// It also adjusts the viewStart to ensure the cursor is always visible.
func (tv *TextViewport) CalculateVisibleWindow(text string, cursor, viewWidth int) (string, int) {
	// Convert text to runes to handle multi-byte characters correctly
	runes := []rune(text)
	textLen := len(runes)

	// 1. Ensure viewWidth is valid
	if viewWidth <= 0 {
		return "", 0
	}

	// 2. Adjust View for Cursor (Behavior C: Scroll when cursor touches edges)
	tv.AdjustViewForCursor(textLen, cursor, viewWidth)

	// 3. Extract visible runes
	end := tv.viewStart + viewWidth
	if end > textLen {
		end = textLen
	}

	visibleRunes := runes[tv.viewStart:end]
	cursorInView := cursor - tv.viewStart

	return string(visibleRunes), cursorInView
}

// AdjustViewForCursor adjusts viewStart to keep cursor within the visible window [viewStart, viewStart + viewWidth]
func (tv *TextViewport) AdjustViewForCursor(textLen, cursor, viewWidth int) {
	if viewWidth <= 0 {
		return
	}

	// If cursor is before viewStart, scroll left
	if cursor < tv.viewStart {
		tv.viewStart = cursor
	}

	// If cursor is beyond viewWidth (at or after the edge), scroll right
	// Note: cursor position can be == textLen (end of string), that position must be visible too
	if cursor > tv.viewStart+viewWidth {
		tv.viewStart = cursor - viewWidth
	}

	// Final boundary checks
	if tv.viewStart < 0 {
		tv.viewStart = 0
	}
	if tv.viewStart > textLen {
		tv.viewStart = textLen
	}

	// If text fits entirely, reset viewStart
	if textLen <= viewWidth {
		tv.viewStart = 0
	}
}

// TruncateWithViewport is a wrapper around fmt.Convert(text).Truncate(...) but using our logic
func (tv *TextViewport) TruncateWithViewport(text string, cursor, viewWidth int) (string, int) {
	if text == "" {
		return "", 0
	}
	return tv.CalculateVisibleWindow(text, cursor, viewWidth)
}
