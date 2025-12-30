package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/tinywasm/devtui"

	example "github.com/tinywasm/devtui/example"
)

// SimpleLogger implements devtui.Loggable
type SimpleLogger struct {
	name string
	log  func(...any)
}

func (l *SimpleLogger) Name() string          { return l.name }
func (l *SimpleLogger) SetLog(f func(...any)) { l.log = f }
func (l *SimpleLogger) Log(m string, a ...any) {
	if l.log != nil {
		l.log(fmt.Sprintf(m, a...))
	}
}

func main() {
	tui := devtui.NewTUI(&devtui.TuiConfig{
		AppName:  "Demo",
		ExitChan: make(chan bool),
		Color:    devtui.DefaultPalette(),
		Logger: func(messages ...any) {
			fmt.Println(messages...) // Replace with actual logging implementation
		},
	})

	// Method chaining with optional timeout configuration
	// New API dramatically simplifies handler implementation

	// Dashboard tab with DisplayHandlers (read-only information)
	dashboard := tui.NewTabSection("Dashboard", "System Overview")
	tui.AddHandler(&example.StatusHandler{}, 0, "", dashboard)

	// Configuration tab with EditHandlers (interactive fields)
	config := tui.NewTabSection("Config", "System Configuration")
	tui.AddHandler(&example.DatabaseHandler{ConnectionString: "postgres://localhost:5432/mydb"}, 2*time.Second, "", config)
	tui.AddHandler(&example.BackupHandler{}, 5*time.Second, "", config)

	// NEW: Chat tab with InteractiveHandler - Demonstrates interactive content management
	chat := tui.NewTabSection("Chat", "AI Chat Assistant")
	chatHandler := &example.SimpleChatHandler{
		Messages:           make([]example.ChatMessage, 0),
		WaitingForUserFlag: false, // Start showing content, not waiting for input
		IsProcessing:       false, // Not processing initially
	}
	tui.AddHandler(chatHandler, 3*time.Second, "", chat)

	// Logging tab with Writers
	logs := tui.NewTabSection("Logs", "System Logs")

	// Unified logging via Loggable handlers
	systemLog := &SimpleLogger{name: "SystemLog"}
	opLog := &SimpleLogger{name: "OpLog"}

	tui.AddHandler(systemLog, 0, "", logs)
	tui.AddHandler(opLog, 0, "", logs)

	systemLog.Log("System initialized")
	systemLog.Log("API demo started")
	systemLog.Log("Chat interface enabled")

	// Generate multiple log entries to test scrolling (30 total)
	go func() {
		for i := 1; i <= 30; i++ {
			time.Sleep(3 * time.Second) // Simulate processing delay
			systemLog.Log("System log entry #%d - Processing data batch", i)
		}
	}()

	opLog.Log("Operation tracking enabled")

	// Generate more tracking entries to test Page Up/Page Down navigation
	go func() {
		for i := 1; i <= 50; i++ {
			time.Sleep(3 * time.Second) // Simulate processing delay
			opLog.Log("Operation #%d - Background task completed successfully", i)
		}
	}()

	// Different timeout configurations:
	// - Synchronous (default): .Register() or timeout = 0
	// - Asynchronous with timeout: .WithTimeout(duration)
	// - Example timeouts: 100*time.Millisecond, 2*time.Second, 1*time.Minute
	// - Tip: Keep timeouts reasonable (2-10 seconds) for good UX

	// Handler Types Summary:
	// • HandlerDisplay: Name() + Content() - Shows immediate content
	// • HandlerEdit: Name() + Label() + Value() + Change() - Interactive fields
	// • HandlerExecution: Name() + Label() + Execute() - Action buttons
	// • HandlerInteractive: Name() + Label() + Value() + Change() + WaitingForUser() - Interactive content
	// • Loggable: Name() + SetLog() - Automatic logging and tracking

	var wg sync.WaitGroup
	wg.Add(1)
	go tui.Start(&wg)
	wg.Wait()
}
