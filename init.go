package devtui

import (
	"bufio"
	"os"
	"sync"
	"time"

	"github.com/tinywasm/fmt"
	tinytime "github.com/tinywasm/time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tinywasm/unixid"
)

// channelMsg es un tipo especial para mensajes del canal
type channelMsg tabContent

// Print representa un mensaje de actualización
type tickMsg time.Time

// DevTUI mantiene el estado de la aplicación
type DevTUI struct {
	*TuiConfig
	*tuiStyle

	id *unixid.UnixID

	ready    bool
	viewport viewport.Model

	focused bool // is the app focused

	TabSections       []*tabSection // represent sections in the tui
	activeTab         int           // current tab index
	editModeActivated bool          // global flag to edit config

	shortcutRegistry *ShortcutRegistry // NEW: Global shortcut key registry

	currentTime     string
	tabContentsChan chan tabContent
	tea             *tea.Program
	testMode        bool // private: only used in tests to enable synchronous behavior

	// MCP integration: separate logger for MCP tool execution
	mcpLogger func(message ...any) // injected by mcpserve via SetLog()

	cursorVisible bool // for blinking effect
}

// cursorTickMsg is used for blinking the cursor
type cursorTickMsg time.Time

// cursorTick creates a command that sends a message every 500ms
func (h *DevTUI) cursorTick() tea.Cmd {
	return tea.Every(500*time.Millisecond, func(t time.Time) tea.Msg {
		return cursorTickMsg(t)
	})
}

type TuiConfig struct {
	AppName  string    // app name eg: "MyApp"
	ExitChan chan bool //  global chan to close app eg: make(chan bool)
	/*// *ColorPalette style for the TUI
	  // if nil it will use default style:
	type ColorPalette struct {
	 Foreground string // eg: #F4F4F4
	 Background string // eg: #000000
	 Primary  string // eg: #FF6600
	 Secondary   string // eg: #666666
	}*/
	Color *ColorPalette

	Logger func(messages ...any) // function to write log error
}

// NewTUI creates a new DevTUI instance and initializes it.
//
// Usage Example:
//
//	config := &TuiConfig{
//	    AppName: "MyApp",
//	    ExitChan: make(chan bool),
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

	tui := &DevTUI{
		TuiConfig:        c,
		focused:          true, // assume the app is focused
		TabSections:      []*tabSection{},
		activeTab:        0, // Will be adjusted in Start() method
		tabContentsChan:  make(chan tabContent, 1000),
		currentTime:      tinytime.FormatTime(tinytime.Now()),
		tuiStyle:         newTuiStyle(c.Color),
		id:               id,                    // Set the ID here
		shortcutRegistry: newShortcutRegistry(), // NEW: Initialize shortcut registry
	}

	// Always add SHORTCUTS tab first
	createShortcutsTab(tui)

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
	return tea.Batch(
		tea.EnterAltScreen,
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

	// NEW: Trigger initial content display for interactive handlers
	h.checkAndTriggerInteractiveContent()

	if _, err := h.tea.Run(); err != nil {
		os.Stdout.WriteString(fmt.Sprintf("Error running DevTUI: %v\n", err))
		os.Stdout.WriteString("\nPress any key to exit...\n")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
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
