package devtui

import (
	"bufio"
	"context"
	"os"
	"sync"
	"time"

	"github.com/tinywasm/fmt"
	tinytime "github.com/tinywasm/time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tinywasm/unixid"
)

// channelMsg es un tipo especial para mensajes del canal
type channelMsg tabContent

// Print representa un mensaje de actualización
type tickMsg time.Time

// cursorTickMsg is used for blinking the cursor
type cursorTickMsg time.Time

// cursorTick creates a command that sends a message every 500ms
func (h *DevTUI) cursorTick() tea.Cmd {
	return tea.Every(500*time.Millisecond, func(t time.Time) tea.Msg {
		return cursorTickMsg(t)
	})
}

// NewTUI creates a new DevTUI instance and initializes it.
//
// Usage Example:
//
//	config := &TuiConfig{
//	    AppName: "MyApp",
//	    Color: nil, // or your *ColorPalette
//	    Logger: func(err any) { os.Stdout.WriteString(fmt.Sprintf("%v\n", err)) },
//	}
//	tui := NewTUI(config)
func NewTUI(c *TuiConfig) *DevTUI {
	if c.AppName == "" {
		c.AppName = "DevTUI"
	}

	// Initialize the unique ID generator first
	id, err := unixid.NewUnixID()
	if err != nil {
		if c.Logger != nil {
			c.Logger("Critical: Error initializing unixid:", err, "- timestamp generation will use fallback")
		}
		// id will remain nil, but createTabContent method will handle this gracefully now
	}

	_, noopCancel := context.WithCancel(context.Background())

	tui := &DevTUI{
		TuiConfig:        c,
		apiKey:           c.APIKey,
		focused:          true, // assume the app is focused
		TabSections:      []*tabSection{},
		activeTab:        0, // Will be adjusted in Start() method
		tabContentsChan:  make(chan tabContent, 1000),
		currentTime:      tinytime.FormatTime(tinytime.Now()),
		tuiStyle:         newTuiStyle(c.Color),
		id:               id,                    // Set the ID here
		shortcutRegistry: newShortcutRegistry(), // NEW: Initialize shortcut registry
		testMode:         c.TestMode,
		sseCancel:        noopCancel,
	}

	// FIXED: Removed manual content sending to prevent duplication
	// HandlerDisplay automatically shows Content() when field is selected
	// No need for manual sendMessageWithHandler() call

	tui.tea = tea.NewProgram(tui,
		tea.WithAltScreen(), // use the full size of the terminal in its "alternate screen buffer"
		// Mouse support disabled to enable terminal text selection
	)

	return tui
}

// Init initializes the terminal UI application.
func (h *DevTUI) Init() tea.Cmd {
	// Start SSE client here (sections must be registered before replay messages arrive)
	if h.ClientMode && h.ClientURL != "" {
		ctx, cancel := context.WithCancel(context.Background())
		h.sseCancel = cancel // set before goroutine starts — no race
		h.sseWg.Add(1)
		go h.startSSEClient(h.ClientURL, ctx)
	}
	return tea.Batch(
		h.listenToMessages(),
		h.tickEverySecond(),
		h.cursorTick(), // Start blinking
	)
}

// Start initializes and runs the terminal UI application.
//
// It accepts optional variadic arguments of any type. If a *sync.WaitGroup
// is provided among these arguments, Start will call its Done() method
// before returning.
//
// The method runs the UI using the internal tea engine, and handles any
// errors that may occur during execution. If an error occurs, it will be
// displayed on the console and the application will wait for user input
// before exiting.
//
// Parameters:
//   - args ...any: Optional arguments. Can include a *sync.WaitGroup for synchronization.
func (h *DevTUI) Start(args ...any) {
	// Check if a WaitGroup was passed
	for _, arg := range args {
		if wg, ok := arg.(*sync.WaitGroup); ok {
			defer wg.Done()
			break
		}
	}

	// Add SHORTCUTS tab last, after all user tabs are registered
	// Only add if it doesn't already exist (idempotency)
	shortcutsExists := false
	for _, tab := range h.TabSections {
		if tab.Title == "SHORTCUTS" {
			shortcutsExists = true
			break
		}
	}
	if !shortcutsExists {
		createShortcutsTab(h)
	}

	// NEW: Trigger initial content display for interactive handlers
	h.checkAndTriggerInteractiveContent()
	h.notifyTabActive(h.activeTab)

	if _, err := h.tea.Run(); err != nil {
		os.Stdout.WriteString(fmt.Sprintf("Error running DevTUI: %v\n", err))
		os.Stdout.WriteString("\nPress any key to exit...\n")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}

	// Terminal is restored here. Now drain the SSE goroutine.
	done := make(chan struct{})
	go func() {
		h.sseWg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// clean exit
	case <-time.After(2 * time.Second):
		os.Exit(0) // terminal already clean; force exit
	}
}

// shutdownMsg triggers a clean exit through the normal Update() path.
// This ensures the full ClearScreen → ExitAltScreen → Quit sequence runs.
type shutdownMsg struct{}

// Shutdown signals the TUI to stop gracefully.
// Safe to call from any goroutine (OS signal handlers, external callers).
func (h *DevTUI) Shutdown() {
	if h.tea != nil {
		go h.tea.Send(shutdownMsg{})
	}
}

// SetTestMode enables or disables test mode for synchronous behavior in tests.
// This should only be used in test files to make tests deterministic.
func (h *DevTUI) SetTestMode(enabled bool) {
	h.testMode = enabled
}

// isTestMode returns true if the TUI is running in test mode (synchronous execution).
// This is an internal method used by field handlers to determine execution mode.
func (h *DevTUI) isTestMode() bool {
	return h.testMode
}
