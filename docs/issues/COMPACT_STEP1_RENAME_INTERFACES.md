# Step 1: Rename Interfaces - MessageTracker to CompactProvider

## Objective

Rename the `MessageTracker` interface and its methods to use "Compact" terminology.

## File: `devtui/interfaces.go`

### Current Code (lines 50-55)

```go
// MessageTracker provides optional interface for message tracking control.
// Handlers can implement this to control message updates and operation tracking.
type MessageTracker interface {
	GetLastOperationID() string
	SetLastOperationID(id string)
}
```

### New Code

```go
// CompactProvider provides optional interface for compact mode control.
// Handlers implementing this can reuse the same line for updates (compact mode).
// When GetCompactID() returns a non-empty string, messages with that ID are updated in-place.
// When it returns "", a new line is always created (history mode).
type CompactProvider interface {
	GetCompactID() string
	SetCompactID(id string)
}
```

## Files to Update References

After renaming the interface, update all files that reference `MessageTracker`:

### 1. `devtui/handlerRegistration.go`

Search and replace:
- `MessageTracker` → `CompactProvider`
- `GetLastOperationID` → `GetCompactID`
- `SetLastOperationID` → `SetCompactID`

Lines affected: ~147-160, ~175-178, ~194-198

### 2. `devtui/anyHandler.go`

Search and replace:
- `getOpIDFunc` → `getCompactIDFunc`
- `setOpIDFunc` → `setCompactIDFunc`
- `GetLastOperationID` → `GetCompactID`
- `SetLastOperationID` → `SetCompactID`
- `lastOpID` → `compactID`

Lines affected: ~27, ~44-45, ~99-117, factory methods

### 3. `devtui/tabSection.go`

Search and replace:
- `GetLastOperationID` → `GetCompactID`
- `SetLastOperationID` → `SetCompactID`

Lines affected: ~122

### 4. `devtui/print.go`

Search and replace:
- `SetLastOperationID` → `SetCompactID`

Lines affected: ~32

## Verification

```bash
go build ./...
go test ./... -v
```

## Completion Checklist

- [ ] Renamed `MessageTracker` to `CompactProvider` in `interfaces.go`
- [ ] Renamed methods to `GetCompactID`/`SetCompactID`
- [ ] Updated all references in `handlerRegistration.go`
- [ ] Updated all references in `anyHandler.go`
- [ ] Updated all references in `tabSection.go`
- [ ] Updated all references in `print.go`
- [ ] Code compiles without errors
- [ ] All tests pass
