# DevTUI - Complete Purpose and Functionality Description

## What is DevTUI?

DevTUI is a **message presentation and formatting system** for terminal interfaces, built on top of [bubbletea](https://github.com/charmbracelet/bubbletea) and [bubbles](https://github.com/charmbracelet/bubbles). 

**DevTUI is NOT a validation system, error handler, or business logic manager.**

**DevTUI IS a display layer** that:
- **Receives messages** from your handlers via `progress()` callbacks or `SetLog()`
- **Formats and organizes** those messages in a clean terminal interface  
- **Manages the visual presentation** - tabs, navigation, scrolling, colors
- **Provides structure** for development tools through a unified handler architecture

You inject handlers that contain your business logic, and DevTUI simply displays whatever information they send through `progress()`. If your handler fails, succeeds, or needs to show status - it's the handler's responsibility to send the appropriate message. DevTUI just shows it.

## Problem it Solves

### The Original Problem
During development of Go-to-WASM compilation tools with:
- File change detection
- DevBrowser hot reload
- CSS/JS file minification
- Complex configuration for full-stack Go applications

**The problems were:**
- Too many scattered logs everywhere
- Complex and confusing visual information
- Difficult to understand what was really happening
- The view layer grew too much and became unmanageable

### The Solution: DevTUI
A TUI that **acts as a pure presentation layer**, enabling:
- **Organized message display** in the same terminal space (no infinite log accumulation)
- **Automatic message formatting** and reordering (always showing what happened last)
- **Clean navigation interface** maintaining focus without UI clutter
- **Simple handler injection** where handlers send messages via `progress()` or a provided `log()` function, and DevTUI displays them automatically.

**Key Principle**: DevTUI doesn't validate, process, or judge your data. It's a dumb display system that shows whatever your handlers tell it to show.

## Ideal Use Cases

### Minimalist Development Tools
- **Real-time compilers** (Go-to-WASM, etc.)
- **File monitors** with automatic actions
- **Development dashboards** with live metrics
- **Configuration interfaces** for complex pipelines
- **Build tools** with multiple steps
- **Asset minifiers** with visual progress

### Integration with Development Environments
- **VS Code**: Integrated terminal with limited space
- **Text editors**: Auxiliary panel for monitoring
- **CI/CD pipelines**: Real-time monitoring interface
- **Docker workflows**: Container configuration and status

## Architecture and Key Concepts

### What are "Handlers"?
Handlers are **business logic components** that:
- **Contain your application logic** (compilation, configuration, deployment, etc.)
- **Decide their own state** and validation rules
- **Send information to DevTUI** via `progress()` callbacks
- **Handle their own errors** by sending appropriate messages

**They are NOT UI widgets** - they are **functionality containers** that use DevTUI as their display layer. DevTUI automatically provides the UI structure (tabs, navigation, formatting) based on handler type.

**Critical**: DevTUI doesn't care if your handler succeeds or fails. It only cares about displaying the messages your handler sends.

### Tab System and Organization
- **Thematic tabs**: Group related handlers (Config, Build, Logs, etc.)
- **One active element**: Only one handler is shown at a time (maintains focus)
- **Arrow navigation**: Left/Right to switch between handlers, Tab/Shift+Tab to switch between tabs
- **Informative footer**: Context of the active handler
- **Automatic ShortcutsHandler**: "SHORTCUTS" tab automatically loaded at position 0 with navigation help

### Unified Logging and Clean Terminal
**Traditional problem**: Logs accumulate infinitely creating visual noise.

**Unified solution**: 
- **Automatic Tracking**: DevTUI matches messages to the handler's `Name()`.
- **Clean Display**: Only the **most recent log entry** per handler is shown in the terminal.
- **Full History**: DevTUI preserves the complete history internally for MCP tools and debugging.
- **Simplicity**: No need for `AddLogger` or `MessageTracker` interfaces. Just implement `Loggable`.
- **Thread-safe**: Handlers call their internal `log()` safely from any goroutine.

## Comparison with Other TUI Libraries

### vs bubbletea + bubbles (base)
- **DevTUI**: Pre-configured abstraction with specific patterns and integrated viewport
- **bubbletea/bubbles**: General framework, requires implementing all UI logic

### vs tview, termui, gocui
- **DevTUI**: Focus on injectable handlers for development
- **Others**: General widgets for complete applications

### Unique Advantage: Functional Minimalism
- **Unified registration**: Single `AddHandler()` method for all capabilities
- **Loggable**: Automatically recognized for name-based logging
- **Last log only**: Enforced clean view by default
- **History preserved**: Full context available via external tools (MCP)
- **Zero coupling**: Consumer packages don't depend on DevTUI

## Handler Types and Their Purposes

### 1. HandlerDisplay (2 methods)
**Purpose**: Read-only information that displays immediately
**Cases**: System status, metrics, help (like ShortcutsHandler), current configuration

### 2. HandlerEdit (4 methods)
**Purpose**: Interactive input fields with validation
**Cases**: Port configuration, URLs, file paths, compilation parameters

### 3. HandlerExecution (3 methods)
**Purpose**: Action buttons with optional progress callbacks
**Cases**: Compile, deploy, clear cache, restart services, backups

### 4. Loggable (2 methods)
**Purpose**: Automatic logging with name-based tracking
**Cases**: Compilation logs, server output, system events, long-running processes

## What DevTUI Does NOT Do

**DevTUI is explicitly NOT responsible for:**

- ❌ **Validating user input** - Your handlers validate their own data
- ❌ **Managing errors** - Your handlers decide how to handle their failures  
- ❌ **Business logic** - Your handlers contain all the application logic
- ❌ **Data persistence** - Your handlers manage their own state
- ❌ **Network operations** - Your handlers handle their own I/O
- ❌ **File operations** - Your handlers manage their own file access
- ❌ **Complex decision making** - Your handlers make all the decisions

**DevTUI only cares about:**
- ✅ **Displaying messages** that handlers send via `progress()`
- ✅ **Formatting the terminal interface** (colors, layout, navigation)
- ✅ **Managing UI state** (active tab, scroll position, focus)
- ✅ **Providing structure** through minimal handler interfaces

**Example of responsibility separation:**
```go
// Handler is responsible for ALL business logic
func (h *DatabaseHandler) Change(newValue any, progress func(string)) {
    // Handler validates input
    if !h.validateConnectionString(newValue.(string)) {
        progress("Error: Invalid connection format")
        return // Handler decides not to change state
    }
    
    // Handler tests connection
    if !h.testConnection(newValue.(string)) {
        progress("Error: Cannot connect to database")
        return // Handler decides not to change state
    }
    
    // Handler updates its own state
    h.connectionString = newValue.(string)
    progress("Database connection updated successfully")
}

// DevTUI simply displays whatever message the handler sends
// It doesn't know or care if the operation succeeded or failed
```

## When NOT to use DevTUI

- **Complex end-user applications** (use tview, bubbletea directly)
- **Multi-window GUIs** (DevTUI is single-window)
- **Highly customized interfaces** (DevTUI prioritizes consistency)
- **Web or desktop applications** (DevTUI is terminal-specific)

## Technical Benefits

### For the Developer
- **Pure presentation layer**: No mixing of business logic with display concerns
- **Handler autonomy**: Each handler manages its own state and validation
- **Simple message interface**: Just send strings via `progress()` - DevTUI handles the rest
- **Fast implementation**: Minimal interfaces (1-4 methods per handler)
- **Zero error handling**: DevTUI displays whatever you send - no error management complexity
- **Reusability**: Portable handlers between projects (business logic stays separate)
- **Testing**: Test your handler logic independently of display layer

### For the End User
- **Consistent experience**: Standard navigation across all tools
- **Organized information**: No visual saturation, viewport with automatic scroll
- **Real-time feedback**: Progress callbacks and message tracking
- **Efficient space**: Maximum terminal utilization
- **Integrated help**: Automatic ShortcutsHandler with navigation commands

## Relationship with GoDEV App

DevTUI is the **main interface** of GoDEV App, a Go development tool that includes:
- Go-to-WASM compilation
- DevBrowser hot reload
- Asset minification
- Dependency management
- File monitoring

GoDEV demonstrated the need to separate the view layer, giving rise to DevTUI as an independent project.
