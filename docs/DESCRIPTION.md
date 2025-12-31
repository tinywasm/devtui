# DevTUI - Detailed Description

DevTUI is a specialized Terminal User Interface library designed for Go development tools (like `godev` and `cdvelop`). It focuses on displaying logs and process statuses in an organized, readable way while providing a decoupled architecture that keeps your business logic clean.

## Core Philosophy

1.  **Decoupling**: DevTUI uses a consumer-driven interface pattern. Your application defines the UI interfaces it needs, and DevTUI implements them. This allows you to write business logic without importing DevTUI directly, making testing easier and dependencies cleaner.
2.  **Clean Terminal**: Most development tools spam the terminal with logs. DevTUI organizes logs by "handlers" (components) and by default shows only the **most recent status message** for each component.
3.  **Universal Registration**: A single `AddHandler` method accepts any supported handler type, simplifying the API surface.

## Architecture

DevTUI is built on the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework.

### The `tabSection` Concept

The interface is organized into **Tab Sections**. Each section represents a major domain of your application (e.g., "Build", "Deploy", "Database").
Inside a section, you register **Handlers**.

### Handlers

A "Handler" is a component that does something and produces logs. DevTUI supports several handler types via interfaces:

1.  **HandlerDisplay**: Read-only information.
2.  **HandlerEdit**: Input fields for configuration.
3.  **HandlerExecution**: Action buttons (e.g., "Deploy").
4.  **HandlerInteractive**: Components that handle user interaction and display dynamic content.
5.  **Loggable**: Any handler can implement `Loggable` to receive a `log` function. Messages sent to this logger are automatically tracked and displayed.

### Universal `AddHandler` API

The `AddHandler` method is the single entry point for registering any component. It uses Go's type system to detect which interfaces your handler implements.

```go
// Universal registration
tui.AddHandler(myHandler, timeout, color, tabSection)
```

## Logging & Tracking

DevTUI's logging system is unique:

-   **Last Message Only**: By default, only the last message from each handler is shown in the main view. This keeps the UI stable and readable.
-   **Full History**: The full log history is preserved internally.
-   **MCP Integration**: DevTUI exposes an MCP (Model Context Protocol) tool called `app_get_logs`. AI agents or external tools can query this tool to retrieve the full log history for debugging, even though the user sees a clean UI.

## Progress System

For long-running operations, handlers should implement the `Loggable` interface.
-   DevTUI provides a `log` function via `SetLog`.
-   Handlers call this function to report progress.
-   The UI updates automatically to show the latest status.
-   This replaces the previous channel-based progress system.

## Keyboard Shortcuts

DevTUI has a built-in global shortcut system.
-   Handlers can implement `Shortcuts() []map[string]string`.
-   These shortcuts are active globally (from any tab).
-   Pressing the key navigates to the handler and triggers its action.

## Testing

The decoupled design makes testing your application trivial. You can mock the UI interface in your tests without needing a real TUI instance.

```go
// Your code depends on this interface
type UI interface {
    NewTabSection(title, description string) any
    AddHandler(handler any, timeout time.Duration, color string, tabSection any)
}
```

This allows you to verify that your application registers the correct handlers and sends the expected logs without running a terminal UI during tests.
