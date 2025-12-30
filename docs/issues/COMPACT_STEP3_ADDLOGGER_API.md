# Step 3: Update AddLogger API

## Objective

Rename the `enableTracking` parameter to `compact` in `AddLogger` and update all related internal methods.

## File: `devtui/handlerRegistration.go`

### Update AddLogger Signature

Change from:
```go
func (t *DevTUI) AddLogger(name string, enableTracking bool, color string, tabSection any) func(message ...any)
```

To:
```go
// AddLogger creates a logger function with the given name and compact capability.
// compact: true = reuse lines for updates (clean terminal), false = always new lines (history mode)
//
// Parameters:
//   - name: Logger identifier for message display
//   - compact: Enable compact mode (vs always new lines)
//   - color: Hex color for logger messages (e.g., "#1e40af", empty string for default)
//   - tabSection: The tab section returned by NewTabSection (as any for decoupling)
//
// Returns:
//   - Variadic logging function: log("message", values...)
//
// Example:
//
//	tab := tui.NewTabSection("BUILD", "Compiler")
//	log := tui.AddLogger("BuildProcess", true, "#1e40af", tab)
//	log("Starting build...")
//	log("Compiling", 42, "files")
func (t *DevTUI) AddLogger(name string, compact bool, color string, tabSection any) func(message ...any) {
	ts := t.validateTabSection(tabSection, "AddLogger")
	return ts.addLogger(name, compact, color)
}
```

### Update Internal addLogger Method

Change from:
```go
func (ts *tabSection) addLogger(name string, enableTracking bool, color string) func(message ...any)
```

To:
```go
func (ts *tabSection) addLogger(name string, compact bool, color string) func(message ...any) {
	if compact {
		handler := &simpleCompactHandler{name: name}
		return ts.registerLoggerFunc(handler, color)
	} else {
		handler := &simpleWriterHandler{name: name}
		return ts.registerLoggerFunc(handler, color)
	}
}
```

### Rename Internal Handler Structs

Rename `simpleWriterTrackerHandler` to `simpleCompactHandler`:

```go
// simpleCompactHandler supports compact mode (line reuse)
type simpleCompactHandler struct {
	name      string
	compactID string
}

func (w *simpleCompactHandler) Name() string {
	return w.name
}

func (w *simpleCompactHandler) GetCompactID() string {
	return w.compactID
}

func (w *simpleCompactHandler) SetCompactID(id string) {
	w.compactID = id
}
```

Keep `simpleWriterHandler` as-is (it has no tracking/compact capability).

### Update registerLoggerHandler

If this method still references `hasTracking`, rename to `hasCompact`:

```go
func (ts *tabSection) registerLoggerHandler(handler HandlerLogger, color string, hasCompact bool) {
	var anyH *anyHandler

	if hasCompact {
		if compactHandler, ok := handler.(interface {
			Name() string
			GetCompactID() string
			SetCompactID(string)
		}); ok {
			anyH = NewCompactWriterHandler(compactHandler, color)
		} else {
			anyH = NewWriterHandler(handler, color)
		}
	} else {
		anyH = NewWriterHandler(handler, color)
	}

	ts.mu.Lock()
	ts.writingHandlers = append(ts.writingHandlers, anyH)
	ts.mu.Unlock()
}
```

## Verification

```bash
go build ./...
go test ./... -v
```

## Completion Checklist

- [ ] Renamed `enableTracking` to `compact` in `AddLogger` signature
- [ ] Updated docstring to use "compact" terminology
- [ ] Renamed `simpleWriterTrackerHandler` to `simpleCompactHandler`
- [ ] Renamed `lastOperationID` field to `compactID` in struct
- [ ] Updated internal `addLogger` method
- [ ] Updated `registerLoggerHandler` if needed
- [ ] Code compiles without errors
- [ ] All tests pass
