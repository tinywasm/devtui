# Step 2: Update anyHandler for Dynamic Modes

## Objective

Add `currentCompactMode` field to `anyHandler` to enable dynamic switching between compact and history modes.

## File: `devtui/anyHandler.go`

### Add New Field to Struct

After the existing `compactID` field (formerly `lastOpID`), add:

```go
type anyHandler struct {
	handlerType handlerType
	timeout     time.Duration
	compactID   string        // Renamed from lastOpID
	mu          sync.RWMutex

	origHandler any
	handlerColor string

	// NEW: Dynamic mode control
	currentCompactMode bool // true = compact (reuse lines), false = history (new lines)

	// Function pointers...
}
```

### Modify GetCompactID Method

Update the `GetCompactID` method to respect the `currentCompactMode` flag:

```go
func (a *anyHandler) GetCompactID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// If compact mode is disabled, always return empty (forces new line)
	if !a.currentCompactMode {
		return ""
	}

	if a.getCompactIDFunc != nil {
		return a.getCompactIDFunc()
	}
	return a.compactID
}
```

### Add SetCompactMode Method

Add a new public method to toggle the mode:

```go
// SetCompactMode toggles between compact (true) and history (false) modes.
// In compact mode, messages reuse the same line for updates.
// In history mode, every message creates a new line (full debug history).
func (a *anyHandler) SetCompactMode(compact bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.currentCompactMode = compact
}

// IsCompactMode returns the current mode state.
func (a *anyHandler) IsCompactMode() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentCompactMode
}
```

### Update Factory Methods

Each factory method must initialize `currentCompactMode` based on the initial `compact` parameter.

Example for `NewWriterTrackerHandler` (now `NewCompactWriterHandler`):

```go
func NewCompactWriterHandler(h interface {
	Name() string
	GetCompactID() string
	SetCompactID(string)
}, color string) *anyHandler {
	return &anyHandler{
		handlerType:        handlerTypeTrackerWriter, // Consider renaming to handlerTypeCompactWriter
		nameFunc:           h.Name,
		getCompactIDFunc:   h.GetCompactID,
		setCompactIDFunc:   h.SetCompactID,
		handlerColor:       color,
		currentCompactMode: true, // NEW: Initialize to compact mode
	}
}
```

For `NewWriterHandler` (non-compact):

```go
func NewWriterHandler(h HandlerLogger, color string) *anyHandler {
	return &anyHandler{
		handlerType:        handlerTypeWriter,
		nameFunc:           h.Name,
		getCompactIDFunc:   func() string { return "" },
		setCompactIDFunc:   func(string) {},
		handlerColor:       color,
		currentCompactMode: false, // NEW: History mode by default
	}
}
```

## Verification

```bash
go build ./...
go test ./... -v
```

## Completion Checklist

- [ ] Added `currentCompactMode bool` field to `anyHandler`
- [ ] Modified `GetCompactID()` to check `currentCompactMode`
- [ ] Added `SetCompactMode(bool)` method
- [ ] Added `IsCompactMode() bool` method
- [ ] Updated all factory methods to initialize `currentCompactMode`
- [ ] Code compiles without errors
- [ ] All tests pass
