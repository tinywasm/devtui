# Refactor Plan for `HandlerEdit` and `HandlerExecution` Interfaces

## Overview
This document outlines the planned changes to refactor the `HandlerEdit` and `HandlerExecution` interfaces in the `devtui` package. The primary goal is to update the method signatures to allow variadic arguments for progress messages, improving flexibility and usability.

## Changes
1. **HandlerEdit Interface**
   - Update the `Change` method signature:
     ```go
     Change(newValue string, progress func(msgs ...any))
     ```

2. **HandlerExecution Interface**
   - Update the `Execute` method signature:
     ```go
     Execute(progress func(msgs ...any))
     ```

## Affected Files and Methods

### Interfaces
- `interfaces.go`
  - Update method signatures in `HandlerEdit` and `HandlerExecution` interfaces.

### Implementations
#### `Change` Method
- `field.go`: `anyHandler.Change`
- `new_api_test.go`: `testEditHandler.Change`
- `example/demo/main.go`: `DatabaseHandler.Change`
- `handler_value_update_test.go`: `ThreadSafePortTestHandler.Change`
- `operation_id_reuse_test.go`: `TestOperationIDHandler.Change`, `TestNewOperationHandler.Change`
- `handler_test.go`: Multiple handlers (`TestEditableHandler`, `TestNonEditableHandler`, etc.)

#### `Execute` Method
- `new_api_test.go`: `testRunHandler.Execute`
- `execution_footer_bug_test.go`: `ExecHandler.Execute`
- `README.md`: `BackupHandler.Execute`
- `race_condition_test.go`: `RaceConditionHandler.Execute`
- `example/demo/main.go`: `BackupHandler.Execute`
- `handler_test.go`: `TestNonEditableHandler.Execute`

### Documentation
- `README.md`: Update examples and method signatures for `Change` and `Execute`.

### Tests
- Update all test cases to reflect the new method signatures and ensure compatibility.

### Progress Message Handling (New Instruction)
- For all progress callbacks (e.g., in `field.go`), always join and translate all variadic arguments into a single string using the `Translate` function from the `tinystring` package before sending or displaying the message.
- Example usage:
  ```go
  import "github.com/tinywasm/fmt"
  msg := tinystring.Translate(msgs...)
  ```
- This ensures that all progress messages are properly composed, translated, and displayed as a single string, as described in the `TRANSLATE.md` documentation.
- Update all relevant progress callback implementations to use this approach, replacing any logic that only uses the first argument or assumes a single string.