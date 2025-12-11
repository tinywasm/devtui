package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/tinywasm/devtui"

	example "github.com/tinywasm/devtui/example"
)

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

	// Basic writer (always creates new lines)
	systemLogger := tui.AddLogger("SystemLogWriter", false, "", logs)
	systemLogger("System initialized")
	systemLogger("API demo started")
	systemLogger("Chat interface enabled")

	// Generate multiple log entries to test scrolling (30 total)
	go func() {
		for i := 1; i <= 30; i++ {
			time.Sleep(3 * time.Second) // Simulate processing delay
			systemLogger("System log entry #%d - Processing data batch", i)
		}
	}()

	// Advanced writer (can update existing messages with tracking)
	opWLogger := tui.AddLogger("OperationLogWriter", true, "", logs)
	opWLogger("Operation tracking enabled")

	// Generate more tracking entries to test Page Up/Page Down navigation
	go func() {
		for i := 1; i <= 50; i++ {
			time.Sleep(3 * time.Second) // Simulate processing delay
			opWLogger("Operation #%d - Background task completed successfully", i)
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
	// • HandlerLogger: Name() - Basic logging (new lines)

	var wg sync.WaitGroup
	wg.Add(1)
	go tui.Start(&wg)
	wg.Wait()
}