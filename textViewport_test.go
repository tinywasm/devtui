package devtui

import (
	"testing"
)

func TestTextViewport_CalculateVisibleWindow(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		cursor         int
		viewWidth      int
		expectedText   string
		expectedCursor int
		expectedStart  int
	}{
		{
			name:           "Empty text",
			text:           "",
			cursor:         0,
			viewWidth:      10,
			expectedText:   "",
			expectedCursor: 0,
			expectedStart:  0,
		},
		{
			name:           "Short text fits perfectly",
			text:           "hello",
			cursor:         5,
			viewWidth:      10,
			expectedText:   "hello",
			expectedCursor: 5,
			expectedStart:  0,
		},
		{
			name:           "Long text, cursor at start",
			text:           "this is a very long text that does not fit",
			cursor:         0,
			viewWidth:      10,
			expectedText:   "this is a ",
			expectedCursor: 0,
			expectedStart:  0,
		},
		{
			name:           "Long text, cursor at limit (no scroll)",
			text:           "0123456789ABCDEF",
			cursor:         10,
			viewWidth:      10,
			expectedText:   "0123456789",
			expectedCursor: 10,
			expectedStart:  0,
		},
		{
			name:           "Long text, cursor exceeds limit (scroll right)",
			text:           "0123456789ABCDEF",
			cursor:         11,
			viewWidth:      10,
			expectedText:   "123456789A",
			expectedCursor: 10,
			expectedStart:  1,
		},
		{
			name:           "Long text, cursor at end",
			text:           "0123456789ABCDEF",
			cursor:         16,
			viewWidth:      5,
			expectedText:   "BCDEF",
			expectedCursor: 5,
			expectedStart:  11,
		},
		{
			name:           "Long text, cursor moves left (scroll left)",
			text:           "0123456789ABCDEF",
			cursor:         3,
			viewWidth:      5,
			expectedText:   "34567",
			expectedCursor: 0,
			expectedStart:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tv := &TextViewport{}
			// To test scroll left, we need to be already scrolled right
			if tt.name == "Long text, cursor moves left (scroll left)" {
				tv.viewStart = 10
			}

			visible, cursorInView := tv.CalculateVisibleWindow(tt.text, tt.cursor, tt.viewWidth)

			if visible != tt.expectedText {
				t.Errorf("expected text %q, got %q", tt.expectedText, visible)
			}
			if cursorInView != tt.expectedCursor {
				t.Errorf("expected cursor in view %d, got %d", tt.expectedCursor, cursorInView)
			}
			if tv.viewStart != tt.expectedStart {
				t.Errorf("expected viewStart %d, got %d", tt.expectedStart, tv.viewStart)
			}
		})
	}
}

func TestTextViewport_SequentialMovement(t *testing.T) {
	tv := &TextViewport{}
	text := "0123456789"
	width := 5

	// 1. Start at pos 0
	visible, cursor := tv.CalculateVisibleWindow(text, 0, width)
	if visible != "01234" || cursor != 0 {
		t.Errorf("Step 1 failed: %q, %d", visible, cursor)
	}

	// 2. Move right to index 4 (still visible at the edge)
	visible, cursor = tv.CalculateVisibleWindow(text, 4, width)
	if visible != "01234" || cursor != 4 || tv.viewStart != 0 {
		t.Errorf("Step 2 failed: %q, %d, start %d", visible, cursor, tv.viewStart)
	}

	// 3. Move right to index 5 (touching edge, should NOT scroll yet)
	visible, cursor = tv.CalculateVisibleWindow(text, 5, width)
	if visible != "01234" || cursor != 5 || tv.viewStart != 0 {
		t.Errorf("Step 3 failed: %q, %d, start %d", visible, cursor, tv.viewStart)
	}

	// 4. Move right to index 6 (exceeding edge, should scroll)
	visible, cursor = tv.CalculateVisibleWindow(text, 6, width)
	if visible != "12345" || cursor != 5 || tv.viewStart != 1 {
		t.Errorf("Step 4 failed: %q, %d, start %d", visible, cursor, tv.viewStart)
	}

	// 5. Move left to index 1 (still visible at the left edge)
	visible, cursor = tv.CalculateVisibleWindow(text, 1, width)
	if visible != "12345" || cursor != 0 || tv.viewStart != 1 {
		t.Errorf("Step 5 failed: %q, %d, start %d", visible, cursor, tv.viewStart)
	}

	// 6. Move left to index 0 (should scroll)
	visible, cursor = tv.CalculateVisibleWindow(text, 0, width)
	if visible != "01234" || cursor != 0 || tv.viewStart != 0 {
		t.Errorf("Step 6 failed: %q, %d, start %d", visible, cursor, tv.viewStart)
	}
}
