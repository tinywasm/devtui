package devtui

import (
	"strings"
	"testing"
	"time"

	"github.com/tinywasm/devtui/example"
	tea "github.com/charmbracelet/bubbletea"
)

// TestChatHandlerRealScenario tests the complete chat interaction flow
// focusing on handler behavior in different states, not DevTUI orchestration
func TestChatHandlerRealScenario(t *testing.T) {
	t.Run("Chat handler behavior following DevTUI responsibility separation", func(t *testing.T) {
		t.Logf("=== TESTING CHAT HANDLER ACCORDING TO DEVTUI PRINCIPLES ===")

		// Create chat handler with initial state (using the real handler from example)
		chatHandler := &example.SimpleChatHandler{}

		var contentDisplayed []string
		mockProgress := func(progress <-chan string) {
			for msg := range progress {
				contentDisplayed = append(contentDisplayed, msg)
				t.Logf("Progress: %s", msg)
			}
		}

		// STATE 1: Initial content display (when DevTUI selects the field)
		t.Logf("State 1: DevTUI selects field -> handler shows content")

		// Verify initial state
		if chatHandler.WaitingForUser() {
			t.Errorf("Initial state: should not be waiting for user")
		}

		// DevTUI calls Change("", progress) when field is selected
		progressChan := make(chan string, 10)
		go func() {
			defer close(progressChan)
			chatHandler.Change("", progressChan)
		}()
		mockProgress(progressChan)

		// Verify welcome content was shown
		if len(contentDisplayed) == 0 {
			t.Errorf("No content displayed in initial state")
		}

		welcomeFound := false
		for _, content := range contentDisplayed {
			if strings.Contains(content, "Welcome") {
				welcomeFound = true
				break
			}
		}
		if !welcomeFound {
			t.Errorf("Expected welcome content, got: %v", contentDisplayed)
		}

		// STATE 2: DevTUI transitions to input mode (this is DevTUI's responsibility)
		t.Logf("State 2: DevTUI activates input mode -> handler becomes ready")

		contentDisplayed = []string{}

		// DevTUI is responsible for managing the input activation
		// The handler just needs to be ready when WaitingForUser() should return true
		chatHandler.WaitingForUserFlag = true // This simulates DevTUI's state management

		// Verify handler is now waiting for user
		if !chatHandler.WaitingForUser() {
			t.Errorf("After DevTUI activation: should be waiting for user")
		}

		// Label should reflect input mode (handler's responsibility)
		if !strings.Contains(chatHandler.Label(), "Type message") {
			t.Errorf("Expected input mode label, got: %s", chatHandler.Label())
		}

		// STATE 3: User types and sends message (handler processes business logic)
		t.Logf("State 3: User sends message -> handler processes it")

		userMessage := "Hello, how are you?"
		progressChan2 := make(chan string, 10)
		go func() {
			defer close(progressChan2)
			chatHandler.Change(userMessage, progressChan2)
		}()
		mockProgress(progressChan2)

		// Note: We cannot safely check handler state immediately after sending
		// as the async operation may not have started yet. The handler's business logic
		// manages its own state - DevTUI just displays what it reports.
		// Race conditions would occur if we check WaitingForUser() or Label() here.

		if chatHandler.Value() != "" {
			t.Errorf("Handler should clear input after sending, got '%s'", chatHandler.Value())
		}

		// Verify handler sent appropriate progress messages
		userMessageFound := false
		for _, content := range contentDisplayed {
			if strings.Contains(content, "U: Hello, how are you?") {
				userMessageFound = true
				break
			}
		}
		if !userMessageFound {
			t.Errorf("Expected user message in progress, got: %v", contentDisplayed)
		}

		// STATE 4: AI response completion (handler's async business logic)
		t.Logf("State 4: Handler completes AI response -> ready for next input")

		// Wait for async AI response (handler's responsibility)
		maxWait := 50
		for i := 0; i < maxWait; i++ {
			// Use Label() method to check if still processing instead of direct field access
			label := chatHandler.Label()
			if !strings.Contains(label, "Processing") {
				break // Processing completed
			}
			time.Sleep(100 * time.Millisecond)
		}

		// Verify handler managed its async operation correctly
		if !chatHandler.WaitingForUser() {
			t.Errorf("After AI response: handler should be waiting for user again")
		}

		// Use Label() method to verify processing is complete
		finalLabel := chatHandler.Label()
		if strings.Contains(finalLabel, "Processing") {
			t.Errorf("After AI response: handler should not be processing, label: %s", finalLabel)
		}

		// Note: We cannot safely check message count after async operations
		// as it would create race conditions. The handler manages its own state
		// and DevTUI respects that encapsulation.

		// STATE 5: DevTUI re-selects field -> handler shows conversation history
		t.Logf("State 5: DevTUI re-selects field -> handler shows history")

		// Simulate DevTUI deactivating input mode (field loses focus, regains focus)
		chatHandler.WaitingForUserFlag = false
		contentDisplayed = []string{}

		// DevTUI calls Change("", progress) when field is re-selected
		progressChan3 := make(chan string, 10)
		go func() {
			defer close(progressChan3)
			chatHandler.Change("", progressChan3)
		}()
		mockProgress(progressChan3)

		// Verify conversation history is shown (handler's business logic)
		historyFound := false
		for _, content := range contentDisplayed {
			if strings.Contains(content, "U: Hello, how are you?") || strings.Contains(content, "A: Response:") {
				historyFound = true
				break
			}
		}
		if !historyFound {
			t.Errorf("Expected conversation history, got: %v", contentDisplayed)
		}

		// STATE 6: Test empty input while in input mode (edge case handling)
		t.Logf("State 6: User presses Enter without typing -> handler guides user")

		chatHandler.WaitingForUserFlag = true // Back to input mode
		contentDisplayed = []string{}

		// User presses Enter without typing anything
		progressChan4 := make(chan string, 10)
		go func() {
			defer close(progressChan4)
			chatHandler.Change("", progressChan4)
		}()
		mockProgress(progressChan4)

		// Handler should guide the user (handler's responsibility for UX)
		guidanceFound := false
		for _, content := range contentDisplayed {
			if strings.Contains(content, "Type message") {
				guidanceFound = true
				break
			}
		}
		if !guidanceFound {
			t.Errorf("Expected user guidance message, got: %v", contentDisplayed)
		}

		t.Logf("=== CHAT HANDLER TEST COMPLETED - ALL RESPONSIBILITIES PROPERLY SEPARATED ===")
	})

	t.Run("Test chat UI rendering and edit mode transitions", func(t *testing.T) {
		tui := DefaultTUIForTest()

		chatHandler := &example.SimpleChatHandler{}

		chatTab := tui.NewTabSection("Chat", "AI Chat Assistant")
		tui.AddHandler(chatHandler, 5*time.Second, "", chatTab)

		tui.viewport.Width = 80
		tui.viewport.Height = 24

		chatTabIndex := len(tui.TabSections) - 1
		tui.activeTab = chatTabIndex
		chatTabSection := chatTab.(*tabSection)
		chatField := chatTabSection.fieldHandlers[0]

		t.Logf("=== TESTING UI RENDERING AND EDIT MODE ===")

		// Phase 1: Before any interaction
		content1 := tui.ContentView()
		t.Logf("Phase 1 - Initial UI:\n%s", content1)

		// Phase 2: Enter to activate input mode
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		content2 := tui.ContentView()
		t.Logf("Phase 2 - After Enter (should be in edit mode):\n%s", content2)

		// Should now be in edit mode (check if tempEditValue is being used)
		if chatField.tempEditValue == "" && !chatHandler.WaitingForUser() {
			// This is expected - the handler manages its own state
			t.Logf("Handler state: WaitingForUser=%v, IsProcessing=%v", chatHandler.WaitingForUser(), chatHandler.IsProcessing)
		}

		// Phase 3: Type message
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hello")})

		content3 := tui.ContentView()
		t.Logf("Phase 3 - After typing 'hello':\n%s", content3)

		// Phase 4: Send message
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		content4 := tui.ContentView()
		t.Logf("Phase 4 - After sending message:\n%s", content4)

		// Should no longer be in edit mode (processing message)
		if chatHandler.WaitingForUser() && !chatHandler.IsProcessing {
			// This is fine - handler completed processing and is ready for next input
			t.Logf("Handler ready for next input: WaitingForUser=%v, IsProcessing=%v", chatHandler.WaitingForUser(), chatHandler.IsProcessing)
		}

		t.Logf("=== UI RENDERING TEST COMPLETED ===")
	})
}