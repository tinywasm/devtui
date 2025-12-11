# RefreshUI - Decoupled UI Updates

## Overview

The `RefreshUI()` method provides a public interface for external tools to notify devtui that the UI needs to be refreshed, without creating tight coupling between the TUI and external components.

## Design Goals

1. **Decoupling**: External tools don't need to import devtui types
2. **Efficiency**: Only refreshes the currently active tab
3. **Thread-Safety**: Can be safely called from any goroutine
4. **Simplicity**: Single method call with no parameters

## Usage

### Basic Usage

```go
// In your external tool
tui.RefreshUI()
```

### Interface-Based Pattern (Recommended)

Define a minimal interface in your external tool to avoid importing devtui:

```go
// In your external tool package
type UIRefresher interface {
    RefreshUI()
}

// Receive TUI as interface
func MyExternalTool(ui UIRefresher) {
    // Do some work...
    
    // Notify UI to refresh
    ui.RefreshUI()
}
```

### Example with Handler

```go
// In your handler implementation
type MyHandler struct {
    ui interface {
        RefreshUI()
    }
}

func (h *MyHandler) OnStateChange() {
    // Update internal state
    h.updateInternalState()
    
    // Notify TUI to refresh the current tab
    h.ui.RefreshUI()
}
```

## When to Use

Call `RefreshUI()` when:

- ✅ External tool state changes (e.g., compilation status)
- ✅ Handler completes an operation
- ✅ Display content needs immediate update
- ✅ Field values change programmatically

Don't call it for:

- ❌ Every log message (handled automatically)
- ❌ User keyboard input (handled by TUI)
- ❌ Channel messages (handled automatically)

## Implementation Details

### How It Works

1. External tool calls `tui.RefreshUI()`
2. Method checks if TUI is ready and running
3. Sends a `refreshTabMsg` to the tea.Program
4. TUI's Update() method handles the message
5. Viewport is updated with current tab content
6. Screen is re-rendered

### Thread Safety

The method is thread-safe because:
- Uses tea.Program's Send() which is thread-safe
- No direct state mutation
- Proper mutex protection in underlying methods

### Performance

- Lightweight: Only sends a message, doesn't block
- Efficient: Only updates current tab, not all tabs
- Non-blocking: Returns immediately

## Complete Example

```go
package main

import (
    "github.com/tinywasm/devtui"
    "time"
)

// External tool with minimal coupling
type CompilerTool struct {
    ui interface {
        RefreshUI()
    }
    status string
}

func (c *CompilerTool) Compile() {
    c.status = "Compiling..."
    c.ui.RefreshUI()
    
    time.Sleep(2 * time.Second)
    
    c.status = "✓ Compilation successful"
    c.ui.RefreshUI()
}

func main() {
    config := &devtui.TuiConfig{
        AppName:  "MyApp",
        ExitChan: make(chan bool),
    }
    
    tui := devtui.NewTUI(config)
    tab := tui.NewTabSection("BUILD", "Build section")
    
    // Create tool with UI reference
    compiler := &CompilerTool{ui: tui}
    
    go func() {
        time.Sleep(1 * time.Second)
        compiler.Compile()
    }()
    
    tui.Start()
}
```

## Testing

See `refresh_tab_test.go` for comprehensive tests including:

- Basic refresh functionality
- Graceful handling before TUI is ready
- Concurrent refresh calls from multiple goroutines
- Integration with external tools

## Alternative Approaches Considered

1. **Direct viewport access**: Would require exporting internal types ❌
2. **Callback registration**: More complex API ❌
3. **Channel-based updates**: Requires managing channels ❌
4. **Global refresh**: Would refresh all tabs (inefficient) ❌

The chosen approach (single public method) provides the best balance of simplicity, efficiency, and decoupling.

## Related

- `updateViewport()` - Internal method that performs the actual update
- `ContentView()` - Generates the content for the current tab
- `Update()` - Main event loop that handles refresh messages
