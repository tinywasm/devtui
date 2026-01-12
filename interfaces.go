package devtui

// HandlerDisplay defines the interface for read-only information display handlers.
// These handlers show static or dynamic content without user interaction.
type HandlerDisplay interface {
	Name() string    // Full text to display in footer (handler responsible for content) eg. "System Status Information Display"
	Content() string // Display content (e.g., "help\n1-..\n2-...", "executing deploy wait...")
}

// HandlerEdit defines the interface for interactive fields that accept user input.
// These handlers allow users to modify values through text input.
type HandlerEdit interface {
	Name() string           // Identifier for logging: "ServerPort", "DatabaseURL"
	Label() string          // Field label (e.g., "Server Port", "Host Configuration")
	Value() string          // Current/initial value (e.g., "8080", "localhost")
	Change(newValue string) // Handle user input + content display via log
}

// HandlerExecution defines the interface for action buttons that execute operations.
// These handlers trigger business logic when activated by the user.
type HandlerExecution interface {
	Name() string  // Identifier for logging: "DeployProd", "BuildProject"
	Label() string // Button label (e.g., "Deploy to Production", "Build Project")
	Execute()      // Execute action + content display via log
}

// HandlerInteractive defines the interface for interactive content handlers.
// These handlers combine content display with user interaction capabilities.
// All content display is handled through progress() for consistency.
type HandlerInteractive interface {
	Name() string           // Identifier for logging: "ChatBot", "ConfigWizard"
	Label() string          // Field label (updates dynamically)
	Value() string          // Current input value
	Change(newValue string) // Handle user input + content display via log
	WaitingForUser() bool   // Should edit mode be auto-activated?
}

// ShortcutProvider defines the optional interface for handlers that provide global shortcuts.
// HandlerEdit implementations can implement this interface to enable global shortcut keys.
type ShortcutProvider interface {
	Shortcuts() []map[string]string // Returns ordered list of single-entry maps with shortcut->description, preserving registration order
}

// Cancelable defines the optional interface for handlers that want to be notified when the user cancels.
// Interactive handlers can implement this to clean up or reset their state when ESC is pressed.
type Cancelable interface {
	Cancel() // Called when user presses ESC to exit interactive mode
}

// Loggable defines optional logging capability for handlers.
// Handlers implementing this receive a logger function from DevTUI
// when registered via AddHandler.
//
// The log function provided by DevTUI:
// - Is never nil (safe to call immediately)
// - Automatically tracks messages by handler Name()
// - Stores full history internally
// - Displays only most recent log in terminal (clean view)
//
// Example implementation:
//
//	type WasmClient struct {
//	    log func(message ...any)
//	}
//
//	func NewWasmClient() *WasmClient {
//	    return &WasmClient{
//	        log: func(message ...any) {}, // no-op until SetLog called
//	    }
//	}
//
//	func (w *WasmClient) Name() string { return "WASM" }
//
//	func (w *WasmClient) SetLog(logger func(message ...any)) {
//	    w.log = logger
//	}
//
//	func (w *WasmClient) Compile() {
//	    w.log("Compiling...")
//	}
type Loggable interface {
	Name() string
	SetLog(logger func(message ...any))
}

// StreamingLoggable enables handlers to display ALL log messages
// instead of the default "last message only" behavior.
type StreamingLoggable interface {
	Loggable
	AlwaysShowAllLogs() bool // Return true to show all messages
}

const (
	LogOpen  = "[..." // Start or update same line with auto-animation
	LogClose = "...]" // Update same line and stop auto-animation
)
