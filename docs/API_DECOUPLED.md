# DevTUI API Decoupling - Complete Refactoring Plan

## Executive Summary

**Objective**: Create a fully decoupled interface for DevTUI to enable clean testing and zero UI dependencies in consumer applications like `godev`.

**Problem**: Current architecture tightly couples `godev` with DevTUI. GODEV imports DevTUI directly, making tests expensive (too many tokens for LLMs), difficult to maintain, and polluted with unnecessary UI implementation details.

**Solution**: 
1. Single universal registration method `AddHandler(handler any, timeout time.Duration, color string)` with internal type casting
2. **CRITICAL**: GODEV must NOT import DevTUI at all - UI passed as interface parameter in `Start()`
3. All UI interaction through minimal interfaces defined in GODEV
4. DevTUI initialization ONLY in `main.go`

---

## Core Design Principles

1. **Zero UI Imports in GODEV**: GODEV package NEVER imports DevTUI
2. **UI as Parameter**: UI instance passed to `Start()` from `main.go`
3. **Single Entry Point**: ONE method `AddHandler()` for ALL handler types
4. **No Return Value**: `AddHandler()` returns nothing - enforces true decoupling
5. **Internal Type Casting**: DevTUI handles all interface detection internally
6. **No Backward Compatibility**: Complete breaking change - clean slate
7. **Minimal Test Interface**: Simple interface for mocking in consumer tests
8. **UI Initialization Only in main.go**: DevTUI created and passed from entry point

---

## PART 1: DevTUI Library Refactoring

### 1.1 New Universal Registration Method

**File**: `/home/cesar/Dev/Pkg/Mine/devtui/handlerRegistration.go`

```go
package devtui

import "time"

// AddHandler is the ONLY method to register handlers of any type.
// It accepts any handler interface and internally detects the type.
// Does NOT return anything - enforces complete decoupling.
//
// Supported handler interfaces (from interfaces.go):
//   - HandlerDisplay: Static/dynamic content display
//   - HandlerEdit: Interactive text input fields
//   - HandlerExecution: Action buttons
//   - HandlerInteractive: Combined display + interaction
//   - HandlerLogger: Basic line-by-line logging (via MessageTracker detection)
//
// Optional interfaces (detected automatically):
//   - MessageTracker: Enables message update tracking
//   - ShortcutProvider: Registers global keyboard shortcuts
//
// Parameters:
//   - handler: ANY handler implementing one of the supported interfaces
//   - timeout: Operation timeout (used for Edit/Execution/Interactive handlers, ignored for Display)
//   - color: Hex color for handler messages (e.g., "#1e40af", empty string for default)
//
// Example:
//   tab.AddHandler(myEditHandler, 2*time.Second, "#3b82f6")
//   tab.AddHandler(myDisplayHandler, 0, "") // timeout ignored for display
//   tab.AddHandler(myExecutionHandler, 5*time.Second, "#10b981")
func (ts *tabSection) AddHandler(handler any, timeout time.Duration, color string) {
	// Type detection and routing
	switch h := handler.(type) {
	
	case HandlerDisplay:
		ts.registerDisplayHandler(h, color)
		
	case HandlerEdit:
		ts.registerEditHandler(h, timeout, color)
		
	case HandlerExecution:
		ts.registerExecutionHandler(h, timeout, color)
		
	case HandlerInteractive:
		ts.registerInteractiveHandler(h, timeout, color)
		
	case HandlerLogger:
		// Logger detection: check for MessageTracker to determine tracking capability
		_, hasTracking := handler.(MessageTracker)
		ts.registerLoggerHandler(h, color, hasTracking)
		
	default:
		// Invalid handler type - log error or panic
		if ts.tui != nil && ts.tui.Logger != nil {
			ts.tui.Logger("ERROR: Unknown handler type provided to AddHandler:", handler)
		}
	}
}

// Internal registration methods (private)

func (ts *tabSection) registerDisplayHandler(handler HandlerDisplay, color string) {
	anyH := newDisplayHandler(handler, color)
	f := &field{
		handler:    anyH,
		parentTab:  ts,
		asyncState: &internalAsyncState{},
	}
	ts.addFields(f)
}

func (ts *tabSection) registerEditHandler(handler HandlerEdit, timeout time.Duration, color string) {
	var tracker MessageTracker
	if t, ok := handler.(MessageTracker); ok {
		tracker = t
	}

	anyH := newEditHandler(handler, timeout, tracker, color)
	f := &field{
		handler:    anyH,
		parentTab:  ts,
		asyncState: &internalAsyncState{},
	}
	ts.addFields(f)

	// Check for shortcut support
	ts.registerShortcutsIfSupported(handler, len(ts.fieldHandlers)-1)
}

func (ts *tabSection) registerExecutionHandler(handler HandlerExecution, timeout time.Duration, color string) {
	anyH := newExecutionHandler(handler, timeout, color)
	f := &field{
		handler:    anyH,
		parentTab:  ts,
		asyncState: &internalAsyncState{},
	}
	ts.addFields(f)
}

func (ts *tabSection) registerInteractiveHandler(handler HandlerInteractive, timeout time.Duration, color string) {
	var tracker MessageTracker
	if t, ok := handler.(MessageTracker); ok {
		tracker = t
	}

	anyH := newInteractiveHandler(handler, timeout, tracker, color)
	f := &field{
		handler:    anyH,
		parentTab:  ts,
		asyncState: &internalAsyncState{},
	}
	ts.addFields(f)
}

func (ts *tabSection) registerLoggerHandler(handler HandlerLogger, color string, hasTracking bool) {
	var anyH *anyHandler
	
	if hasTracking {
		// Handler implements MessageTracker
		tracker := handler.(MessageTracker)
		anyH = newWriterTrackerHandler(handler, tracker, color)
	} else {
		// Basic logger without tracking
		anyH = newWriterHandler(handler, color)
	}

	// Register in writing handlers list
	ts.mu.Lock()
	ts.writingHandlers = append(ts.writingHandlers, anyH)
	ts.mu.Unlock()
}
```

---

### 1.2 Rename NewLogger to AddLogger (Consistency)

**Change**: Rename `NewLogger()` to `AddLogger()` for consistency with `AddHandler()`.

**File**: `/home/cesar/Dev/Pkg/Mine/devtui/handlerRegistration.go`

**RENAME METHOD**:

```go
// AddLogger creates a logger function with the given name and tracking capability
// enableTracking: true = can update existing lines, false = always creates new lines
//
// Example:
//   log := tab.AddLogger("BuildProcess", true, "#1e40af")
//   log("Starting build...")
//   log("Compiling", 42, "files")
func (ts *tabSection) AddLogger(name string, enableTracking bool, color string) func(message ...any) {
	if enableTracking {
		handler := &simpleWriterTrackerHandler{name: name}
		return ts.registerLoggerFunc(handler, color)
	} else {
		handler := &simpleWriterHandler{name: name}
		return ts.registerLoggerFunc(handler, color)
	}
}
```

**Why AddLogger instead of NewLogger?**
- ✅ **Consistent naming** - matches `AddHandler()` pattern
- ✅ Both methods add/register something to the tab section
- ✅ Clear API - all registration methods start with `Add`
- ✅ Function return type - simple and direct usage

---

### 1.3 DevTUI Implements Interfaces (Consumer Defines Them)

**IMPORTANT**: DevTUI does NOT define the interfaces. Consumer applications (like GODEV) define their own interfaces, and DevTUI simply implements them through its existing methods.

**File**: `/home/cesar/Dev/Pkg/Mine/devtui/init.go` (add compile-time verification)

```go
// ============================================================================
// INTERFACE COMPLIANCE VERIFICATION
// ============================================================================
// These verify that DevTUI can satisfy consumer-defined interfaces.
// The actual interface definitions live in consumer packages (like godev.TuiInterface).
// This is just a documentation example - actual verification happens when consumer compiles.

// Example of what a consumer interface might look like (NOT defined here):
//
// type TuiInterface interface {
//     NewTabSection(title, description string) TabSectionInterface
//     Start(wg *sync.WaitGroup)
// }
//
// type TabSectionInterface interface {
//     AddHandler(handler any, timeout time.Duration, color string)
//     AddLogger(name string, enableTracking bool, color string) func(message ...any)
// }

// DevTUI already has these methods:
// - NewTabSection(title, description string) *tabSection
// - Start(wg *sync.WaitGroup)
//
// tabSection will have these methods:
// - AddHandler(handler any, timeout time.Duration, color string) [NEW - to be implemented]
// - AddLogger(name string, enableTracking bool, color string) func(message ...any) [RENAMED from NewLogger]
```

**CRITICAL NOTES**:
1. DevTUI does NOT define `TuiInterface` - consumers do
2. DevTUI implements methods that consumers expect
3. Consumer code defines interfaces matching DevTUI's public API
4. This achieves true decoupling - DevTUI doesn't know who uses it
5. NewLogger returns anonymous function - simpler than interface

---

### 1.4 Remove Deprecated Methods

**File**: `/home/cesar/Dev/Pkg/Mine/devtui/handlerRegistration.go`

**DELETE these methods completely:**
```go
// REMOVE:
func (ts *tabSection) AddDisplayHandler(handler HandlerDisplay, color string) *tabSection
func (ts *tabSection) AddEditHandler(handler HandlerEdit, timeout time.Duration, color string) *tabSection
func (ts *tabSection) AddExecutionHandler(handler HandlerExecution, timeout time.Duration, color string) *tabSection
func (ts *tabSection) AddInteractiveHandler(handler HandlerInteractive, timeout time.Duration, color string) *tabSection
```

---

## PART 2: GODEV Application Refactoring

### 2.1 Create UI Interface Abstraction (NO DevTUI Import)

**File**: `/home/cesar/Dev/Pkg/Mine/godev/ui_interface.go` (new file)

```go
package godev

import (
	"sync"
	"time"
)

// ============================================================================
// UI INTERFACES - GODEV defines its own interfaces, NO DevTUI import
// ============================================================================

// TuiInterface defines the minimal UI interface needed by GODEV.
// This interface is implemented by DevTUI but GODEV doesn't know that.
// GODEV never imports DevTUI package.
type TuiInterface interface {
	// NewTabSection creates a new tab section
	NewTabSection(title, description string) TabSectionInterface
	
	// Start initializes and runs the UI
	Start(wg *sync.WaitGroup)
}

// TabSectionInterface defines the minimal tab section interface.
type TabSectionInterface interface {
	// AddHandler registers any type of handler
	AddHandler(handler any, timeout time.Duration, color string)
	
	// AddLogger creates a logger function for this tab section
	// Returns anonymous function for simple usage: log("message")
	AddLogger(name string, enableTracking bool, color string) func(message ...any)
}
```

### 2.2 Update Handler Struct (NO DevTUI Import)

**File**: `/home/cesar/Dev/Pkg/Mine/godev/start.go`

```go
package godev

import (
	"os"
	"sync"
	// NO DevTUI import here!
)

// handler contains application state and dependencies
// CRITICAL: This struct does NOT import DevTUI
type handler struct {
	rootDir   string
	config    *Config
	tui       TuiInterface // Interface defined in GODEV, not DevTUI
	exitChan  chan bool
	
	// Build dependencies
	serverHandler *goserver.ServerHandler
	assetsHandler *AssetMin
	wasmClient   *tinywasm.TinyWasm
	watcher       *devwatch.DevWatch
	browser       *devbrowser.DevBrowser
	
	// Deploy dependencies
	deployCloudflare *goflare.Goflare
	
	// Test hooks
	pendingBrowserReload func() error
}

// Start is called from main.go with UI passed as parameter
// CRITICAL: UI instance created in main.go, passed here as interface
func Start(rootDir string, logger func(messages ...any), ui TuiInterface, exitChan chan bool) {
	h := &handler{
		rootDir:  rootDir,
		tui:      ui, // UI passed from main.go
		exitChan: exitChan,
	}

	ActiveHandler = h

	// Validate directory
	homeDir, _ := os.UserHomeDir()
	if rootDir == homeDir || rootDir == "/" {
		logger("Cannot run godev in user root directory. Please run in a Go project directory")
		return
	}

	// ADD SECTIONS using the passed UI interface
	h.AddSectionBUILD()
	h.AddSectionDEPLOY()

	var wg sync.WaitGroup
	wg.Add(3)

	// Start the UI (passed from main.go)
	go h.tui.Start(&wg)

	// Start server
	go h.serverHandler.StartServer(&wg)

	// Start file watcher
	go h.watcher.FileWatcherStart(&wg)

	wg.Wait()
}
```

### 2.3 Update main.go to Create and Pass UI

**File**: `/home/cesar/Dev/Pkg/Mine/godev/cmd/godev/main.go`

**CRITICAL**: This is the ONLY file that imports DevTUI

```go
package main

import (
	"log"
	"os"

	"github.com/tinywasm/godev"
	"github.com/tinywasm/devtui" // ONLY import DevTUI in main.go
)

func main() {
	// Initialize root directory
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting current working directory:", err)
		return
	}

	exitChan := make(chan bool)

	// Create a Logger instance
	logger := godev.NewLogger()

	// Create DevTUI instance (ONLY in main.go)
	ui := devtui.NewTUI(&devtui.TuiConfig{
		AppName:  "GODEV",
		ExitChan: exitChan,
		Color:    devtui.DefaultPalette(),
		Logger:   func(messages ...any) { logger.Logger(messages...) },
	})

	// Pass UI as interface to Start - GODEV doesn't know it's DevTUI
	godev.Start(rootDir, logger.Logger, ui, exitChan)
}
```

### 2.4 Update section-build.go (Uses Interface, NO DevTUI Import)

**File**: `/home/cesar/Dev/Pkg/Mine/godev/section-build.go`

**BEFORE** (tight coupling):
```go
package godev

import (
	// ... other imports
	"github.com/tinywasm/devtui" // ❌ BAD: Direct DevTUI import
)

func (h *handler) AddSectionBUILD() {
	sectionBuild := h.tui.NewTabSection("BUILD", "Building and Compiling")
	
	// Multiple method calls with different signatures
	wasmLogger := sectionBuild.NewLogger("WASM", false, colorPurpleMedium)
	// ... etc
}
```

**AFTER** (decoupled via interface):
```go
package godev

import (
	// ... other imports
	// ✅ NO DevTUI import!
)

func (h *handler) AddSectionBUILD() {
	// h.tui is TuiInterface defined in GODEV, not DevTUI
	sectionBuild := h.tui.NewTabSection("BUILD", "Building and Compiling")
	
	// Logger creation returns func(message ...any)
	wasmLogger := sectionBuild.AddLogger("WASM", false, colorPurpleMedium)
	serverLogger := sectionBuild.AddLogger("SERVER", false, colorBlueMedium)
	assetsLogger := sectionBuild.AddLogger("ASSETS", false, colorGreenMedium)
	watchLogger := sectionBuild.AddLogger("WATCH", false, colorYellowMedium)
	configLogger := sectionBuild.AddLogger("CONFIG", true, colorTealMedium)
	browserLogger := sectionBuild.AddLogger("BROWSER", true, colorPinkMedium)

	// Use loggers - simple function calls
	h.config = NewConfig(h.rootDir, configLogger)
	
	// Initialize handlers - these use the loggers above
	// Loggers are anonymous functions, implementation unknown to GODEV
	
	// If there are any direct handler registrations, update them:
	// OLD: tab.AddEditHandler(handler, timeout, color)
	// NEW: tab.AddHandler(handler, timeout, color)
}
```

### 2.5 Update section-deploy.go Similarly

**File**: `/home/cesar/Dev/Pkg/Mine/godev/section-deploy.go`

Apply same pattern as section-build.go:
- NO DevTUI import
- Use `h.tui` as `TuiInterface` (defined in GODEV)
- Use `AddHandler()` for any handler registrations
- Logger creation returns `func(message ...any)` - simple functions

---

## PART 3: Testing Strategy (NO DevTUI Import)

### 3.1 Mock TUI Interface for Tests

**File**: `/home/cesar/Dev/Pkg/Mine/godev/ui_mock_test.go` (new file)

**CRITICAL**: Mock implementation in GODEV, NO DevTUI import needed!

```go
package godev

import (
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// MOCK IMPLEMENTATIONS - NO DevTUI import needed!
// ============================================================================

// mockTui implements TuiInterface for testing
// This is defined in GODEV, not DevTUI
type mockTui struct {
	sections []*mockTabSection
}

func (m *mockTui) NewTabSection(title, description string) TabSectionInterface {
	section := &mockTabSection{
		title:       title,
		description: description,
		handlers:    []any{},
		loggers:     []*mockLogger{},
	}
	m.sections = append(m.sections, section)
	return section
}

func (m *mockTui) Start(wg *sync.WaitGroup) {
	// Mock implementation - do nothing
	if wg != nil {
		wg.Done()
	}
}

// mockTabSection implements TabSectionInterface for testing
type mockTabSection struct {
	title       string
	description string
	handlers    []any
	loggers     []*mockLogger
}

func (m *mockTabSection) AddHandler(handler any, timeout time.Duration, color string) {
	m.handlers = append(m.handlers, handler)
}

func (m *mockTabSection) AddLogger(name string, enableTracking bool, color string) func(message ...any) {
	logger := &mockLogger{
		name:     name,
		tracking: enableTracking,
		messages: []string{},
	}
	m.loggers = append(m.loggers, logger)
	
	// Return anonymous function that captures the logger
	return func(message ...any) {
		var msg string
		for i, m := range message {
			if i > 0 {
				msg += " "
			}
			msg += fmt.Sprint(m)
		}
		logger.messages = append(logger.messages, msg)
	}
}

// mockLogger stores logged messages for testing
type mockLogger struct {
	name     string
	tracking bool
	messages []string
}

// ============================================================================
// TEST EXAMPLE - Clean test with NO DevTUI knowledge
// ============================================================================

func TestAddSectionBUILD(t *testing.T) {
	// Create mock TUI - NO DevTUI import needed!
	mockTUI := &mockTui{}
	
	// Create handler with mock - completely decoupled
	h := &handler{
		rootDir:  "/test/dir",
		tui:      mockTUI,  // TuiInterface defined in GODEV
		exitChan: make(chan bool),
	}
	
	// Execute - no UI pollution!
	h.AddSectionBUILD()
	
	// Verify - no DevTUI internals needed!
	if len(mockTUI.sections) != 1 {
		t.Errorf("Expected 1 section, got %d", len(mockTUI.sections))
	}
	
	section := mockTUI.sections[0]
	if section.title != "BUILD" {
		t.Errorf("Expected title 'BUILD', got '%s'", section.title)
	}
	
	// Verify loggers were created
	if len(section.loggers) < 6 { // WASM, SERVER, ASSETS, WATCH, CONFIG, BROWSER
		t.Errorf("Expected 6 loggers, got %d", len(section.loggers))
	}
	
	// Get logger function and test it
	logFunc := section.AddLogger("TestLogger", false, "")
	logFunc("Test message")
	
	// Verify message was logged
	testLogger := section.loggers[len(section.loggers)-1] // Last created logger
	if len(testLogger.messages) != 1 {
		t.Error("Expected message to be logged")
	}
	if testLogger.messages[0] != "Test message" {
		t.Errorf("Expected 'Test message', got '%s'", testLogger.messages[0])
	}
	
	// Test logger tracking capability
	trackerLogger := section.loggers[4] // CONFIG logger has tracking
	if !trackerLogger.tracking {
		t.Error("Expected CONFIG logger to have tracking enabled")
	}
}

// Helper function to create mock UI for tests
func NewMockTUI() TuiInterface {
	return &mockTui{
		sections: []*mockTabSection{},
	}
}
```

---

## PART 4: Migration Steps

### Step 1: Update DevTUI Library
1. Implement new `AddHandler()` method with all internal routing in `handlerRegistration.go`
2. Rename `NewLogger()` to `AddLogger()` for consistency
3. Delete old registration methods (AddDisplayHandler, AddEditHandler, etc.)
4. Add compile-time verification examples in `init.go` (documentation only)
5. Run all DevTUI tests to ensure functionality
6. Update DevTUI README with new API examples

### Step 2: Update GODEV Application
1. **Create `ui_interface.go`** with interface definitions (TuiInterface, TabSectionInterface)
2. **Update `start.go`**:
   - Change `Start()` signature to accept `TuiInterface` parameter
   - Update `handler` struct to use `TuiInterface` instead of `*devtui.DevTUI`
   - Remove DevTUI import
3. **Update `main.go`**:
   - Create DevTUI instance in main.go (ONLY place that imports DevTUI)
   - Pass UI as parameter to `Start()`
4. **Update `section-build.go`**:
   - Remove DevTUI import
   - Use `h.tui` as `TuiInterface`
   - Update any handler registrations to use `AddHandler()`
   - Update logger creation to use `AddLogger()` (renamed from `NewLogger()`)
   - Logger usage remains `log("message")` - NO changes needed
5. **Update `section-deploy.go`**: Same as section-build.go
6. **Create `ui_mock_test.go`**: Mock implementations for testing
7. **Write clean tests**: Using mocks with NO DevTUI imports

### Step 3: Verify Complete Decoupling
1. **Verify GODEV has NO DevTUI imports** except in `cmd/godev/main.go`
2. Run `go list -f '{{.Imports}}' ./...` to verify no DevTUI in GODEV package
3. Run all tests to ensure mocks work correctly
4. Verify test files have NO DevTUI imports
5. Measure test code size reduction

### Step 4: Cleanup and Documentation
1. Remove any unused builder patterns in DevTUI
2. Update DevTUI README emphasizing decoupled architecture
3. Update GODEV documentation
4. Add migration guide for other consumers
5. Document interface pattern for other projects

---

## Benefits of This Approach

### For DevTUI Library:
- ✅ **Consistent API** - All registration methods use `Add` prefix
- ✅ Single, consistent pattern - no confusion about which method to use
- ✅ Cleaner codebase - less code to maintain
- ✅ Easier to extend - adding new handler types is straightforward
- ✅ Better encapsulation - internal details hidden from consumers
- ✅ **Consumer-agnostic** - doesn't know or care who uses it

### For GODEV Application:
- ✅ **ZERO coupling** - GODEV package has NO DevTUI import
- ✅ **Consumer defines interfaces** - GODEV controls its own contracts
- ✅ **Testable** - easy to mock TUI for unit tests
- ✅ **No UI pollution in tests** - test only business logic
- ✅ **Cheaper tests** - fewer tokens for LLMs (>50% reduction)
- ✅ **UI only in main.go** - clean separation of concerns
- ✅ **Pluggable UI** - could swap DevTUI for another UI library

### For Developers:
- ✅ Simpler mental model - one way to do things
- ✅ Less documentation to read
- ✅ Faster development - no decision paralysis
- ✅ Easier debugging - clear flow through codebase
- ✅ **True layered architecture** - business logic independent of UI

---

## Breaking Changes

### Removed Methods (DevTUI):
- `AddDisplayHandler()` → Use `AddHandler()`
- `AddEditHandler()` → Use `AddHandler()`
- `AddExecutionHandler()` → Use `AddHandler()`
- `AddInteractiveHandler()` → Use `AddHandler()`

### Renamed Methods (DevTUI):
- `NewLogger()` → `AddLogger()` (for consistency with AddHandler)

### API Changes (DevTUI):
- All registration methods now return nothing (void)
- `NewLogger()` renamed to `AddLogger()` for consistency
- `AddLogger()` returns `func(message ...any)` - simple function
- **Consistent API**: All registration methods start with `Add` prefix

### API Changes (GODEV):
- **CRITICAL**: `Start()` signature changed to accept UI parameter
- **CRITICAL**: GODEV package has NO DevTUI import
- Consumer defines own interfaces (TuiInterface, TabSectionInterface)
- Logger creation uses `AddLogger()` instead of `NewLogger()`
- Logger usage remains `log("message")` - NO changes needed

### Migration Example:

**OLD CODE** (GODEV):
```go
// start.go
package godev
import "github.com/tinywasm/devtui" // ❌ BAD

func Start(rootDir string, logger func(messages ...any), exitChan chan bool) {
	// Creates DevTUI internally
	tui := devtui.NewTUI(...)
	h.tui = tui
}

// section-build.go
tab.AddEditHandler(myHandler, 2*time.Second, "#3b82f6")
log := tab.NewLogger("Builder", true, "#10b981") // ❌ OLD: NewLogger
log("Building project...")
```

**NEW CODE** (GODEV):
```go
// ui_interface.go (NEW FILE)
package godev
// NO DevTUI import!
type TuiInterface interface {
    NewTabSection(title, description string) TabSectionInterface
    Start(wg *sync.WaitGroup)
}
type TabSectionInterface interface {
    AddHandler(handler any, timeout time.Duration, color string)
    AddLogger(name string, enableTracking bool, color string) func(message ...any)
}

// start.go
package godev
// NO DevTUI import!
func Start(rootDir string, logger func(messages ...any), ui TuiInterface, exitChan chan bool) {
	h.tui = ui // UI passed from main.go
}

// main.go (cmd/godev/main.go)
package main
import "github.com/tinywasm/devtui" // ✅ ONLY import here
ui := devtui.NewTUI(...)
godev.Start(rootDir, logger.Logger, ui, exitChan) // Pass UI

// section-build.go
// NO DevTUI import!
tab.AddHandler(myHandler, 2*time.Second, "#3b82f6")
log := tab.AddLogger("Builder", true, "#10b981") // ✅ NEW: AddLogger for consistency
log("Building project...") // ✅ Simple function call, no .Log()
```

---

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────┐
│ main.go (cmd/godev/main.go)                                 │
│ - ONLY file that imports DevTUI                             │
│ - Creates concrete DevTUI instance                          │
│ - Passes as TuiInterface to godev.Start()                   │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ GODEV Package (github.com/tinywasm/godev)                    │
│ - NO DevTUI import!                                         │
│ - Defines own interfaces (ui_interface.go):                 │
│   • TuiInterface                                            │
│   • TabSectionInterface                                     │
│ - Start() receives TuiInterface parameter                   │
│ - Business logic uses interfaces only                       │
│ - Loggers are simple functions: log("msg")                  │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ DevTUI Package (github.com/tinywasm/devtui)                  │
│ - Implements methods that match consumer interfaces         │
│ - Does NOT define consumer interfaces                       │
│ - Single AddHandler() method for all types                  │
│ - AddLogger() returns func(message ...any)                  │
│ - Consistent API: all registration with Add prefix          │
│ - Internal type casting and routing                         │
└─────────────────────────────────────────────────────────────┘
```

## Questions to Address Before Implementation

1. **Logger Interface**: Should we keep the variadic `Log(message ...any)` or force single string?
   - **Recommendation**: Keep variadic for backward compatibility and convenience

2. **Error Handling**: Should `AddHandler()` panic on unknown types or silently fail?
   - **Recommendation**: Log error using TUI logger, don't panic (more resilient)

3. **Color Parameter**: Should empty string use default or require explicit color?
   - **Recommendation**: Empty string = use default (current behavior)

4. **Start() Signature**: Should we keep the logger parameter separate from UI?
   - **Recommendation**: YES - logger is for fatal errors before UI creation

5. **Transition Period**: Should we support both APIs temporarily?
   - **Recommendation**: NO - clean break, update all code at once

---

## Implementation Priority

### Phase 1 (Critical - Enables Testing):
1. Create minimal interfaces in GODEV (TuiInterface, TabSectionInterface)
2. Implement AddHandler() method in DevTUI
3. Update GODEV to receive UI as parameter in Start()
4. Update GODEV to use interfaces (no DevTUI import)

### Phase 2 (Cleanup):
5. Remove deprecated methods
6. Update all tests
7. Update documentation

### Phase 3 (Polish):
8. Add examples
9. Create migration guide
10. Version bump and release

---

## Success Criteria

- ✅ Single AddHandler() method handles all handler types
- ✅ No method returns *tabSection (full decoupling)
- ✅ **GODEV package has ZERO DevTUI imports** (except cmd/godev/main.go)
- ✅ **GODEV defines its own interfaces** (TuiInterface, TabSectionInterface)
- ✅ **UI passed as parameter to Start()** from main.go
- ✅ GODEV tests can run without creating real DevTUI instance
- ✅ Test code size reduced by >50%
- ✅ Mock implementation is <100 lines of code
- ✅ All existing functionality preserved
- ✅ Compile-time type safety maintained
- ✅ **Pluggable architecture** - could swap UI implementations

---

## Key Architectural Decision

**CRITICAL POINT**: 
- **GODEV does NOT import DevTUI** (except in main.go)
- **GODEV defines its own interfaces** that describe what it needs
- **DevTUI implements methods** that happen to match those interfaces
- **main.go** is the only place that knows about DevTUI concrete type
- **This enables true decoupling** - business logic knows nothing about UI

This is the **Dependency Inversion Principle** in action:
- High-level module (GODEV) does NOT depend on low-level module (DevTUI)
- Both depend on abstractions (interfaces defined by GODEV)
- UI implementation is injected at runtime (in main.go)

---

## Open Questions

**Por favor, revisa este plan ACTUALIZADO y responde:**

1. ✅ ¿Ahora está claro que GODEV NO importa DevTUI? (excepto main.go)
2. ✅ ¿Entiendes que GODEV define sus propias interfaces?
3. ✅ ¿Está claro que la UI se pasa como parámetro a Start()?
4. ✅ ¿Está claro que AddLogger() retorna función anónima, NO interfaz?
5. ✅ ¿El renombre de NewLogger → AddLogger mantiene consistencia con AddHandler?
6. ¿La estrategia de tener un solo método `AddHandler()` cumple tus expectativas?
7. ¿El enfoque de que `AddHandler()` no retorne nada es aceptable para ti?
8. ¿Las interfaces `TuiInterface` y `TabSectionInterface` son suficientes?

**Aspectos que requieren tu decisión:**

- **Error handling**: ¿Qué hacer cuando `AddHandler()` recibe un tipo desconocido?
  - Opción 1: Log error y continuar (resiliente)
  - Opción 2: Panic (fail-fast)
  - Opción 3: Silencioso (ignorar)
- **Start() signature**: ¿Es correcto `Start(rootDir, logger, ui, exitChan)`?
  - ¿El orden de parámetros es óptimo?
  - ¿Debería ser `Start(ui, rootDir, logger, exitChan)`?

---

## Tracking de Pruebas Fallidas

Para un seguimiento detallado del estado de las pruebas durante esta refactorización, consulta el siguiente documento:
[Lista de Pruebas con Errores de Compilación](./FAIL_TEST.md)

---

## Next Steps

Una vez apruebes este plan:

1. Implementaremos el método `AddHandler()` universal en DevTUI
2. Renombraremos `NewLogger()` a `AddLogger()` en DevTUI
3. Crearemos las interfaces en `godev/ui_interface.go`
4. Actualizaremos `godev/start.go` para recibir UI como parámetro
5. Actualizaremos `godev/cmd/godev/main.go` para crear y pasar UI
6. Eliminaremos imports de DevTUI en todo godev (excepto main.go)
7. Crearemos los mocks para testing en `godev/ui_mock_test.go`
8. Migraremos todos los handlers a usar `AddHandler()`
9. Migraremos todos los loggers a usar `AddLogger()`

**¿Procedo con la implementación o tienes ajustes al plan?**
