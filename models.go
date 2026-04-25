package devtui

import (
	"context"
	"github.com/tinywasm/unixid"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// GetLogsArgs defines arguments for the app_get_logs tool.
// ormc:formonly
type GetLogsArgs struct {
	Section string `input:"text"`
}

// ActionArgs defines arguments for the tinywasm/action method.
// ormc:formonly
type ActionArgs struct {
	Key   string `input:"text"`
	Value string `input:"text" json:",omitempty"`
}

// DevTUI mantiene el estado de la aplicación
type DevTUI struct {
	*TuiConfig
	*tuiStyle

	apiKey string // stored from TuiConfig.APIKey

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

	isShuttingDown atomic.Bool
	sseCancel      context.CancelFunc // cancels SSE HTTP request context
	sseWg          sync.WaitGroup     // tracks SSE goroutine
}

type TuiConfig struct {
	AppName    string // app name eg: "MyApp"
	AppVersion string // app version eg: "v1.0.0"
	Debug      bool   // NEW: Enable debug mode for unfiltered logs
	TestMode   bool   // Set to true for synchronous execution in tests
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

	ClientMode bool   // true if it should listen to SSE
	ClientURL  string // e.g. http://localhost:3030/logs
	APIKey     string // Bearer token for secured daemon; set by app, empty = open/local
}
