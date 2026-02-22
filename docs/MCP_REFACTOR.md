# DevTUI Refactoring Plan (Headless logs & SSE Client)

## Development Rules
<!-- START_SECTION:CORE_PRINCIPLES -->
- **Single Responsibility Principle (SRP):** Every file (CSS, Go, JS) must have a single, well-defined purpose. This must be reflected in both the file's content and its naming convention.

- **Mandatory Dependency Injection (DI):**
    - **No Global State:** Avoid direct system calls (OS, Network) in logic.
    - **Interfaces:** Define interfaces for external dependencies (`Downloader`, `ProcessManager`).
    - **Composition:** Main structs must hold these interfaces.
    - **Injection:** `cmd/<app_name>/main.go` is the ONLY place where "Real" implementations are injected.

- **Framework-less Development:** For HTML/Web projects, use only the **Standard Library**. No external frameworks or libraries are allowed.

- **Strict File Structure:**
    - **Flat Hierarchy:** Go libraries must avoid subdirectories. Keep files in the root.
    - **Max 500 lines:** Files exceeding 500 lines MUST be subdivided and renamed by domain.
    - **Test Organization:** If >5 test files exist in root, move **ALL** tests to `tests/`.
<!-- END_SECTION:CORE_PRINCIPLES -->

## 1. Objective
Refactor `tinywasm/devtui` to support two modes of execution: Headless (generating logs but not rendering) and Client (reading logs from SSE and rendering).

**IMPORTANT RECOVERY PROCEDURE**: Before implementing these changes, you MUST create a git recovery branch (e.g., `git checkout -b refactor-mcp-daemon`).

## 2. Sequence Flow
See [MCP_REFACTOR_FLOW.md](diagrams/MCP_REFACTOR_FLOW.md) for precise execution paths.

## 3. Precise Code Changes

### 3.1. `devtui/init.go` and `DevTUI` Struct
- **Client Mode Flag**: Add `ClientMode bool` (true if it should listen to SSE) and `ClientURL string` (e.g., `http://localhost:3030/logs`) in `TuiConfig`.
- **Relying on Native Headless Logging**: `devtui` already supports displaying only logs without interactive colors. The new logic must capitalize on this. If `Headless == true` AND `ClientMode == false`, let Bubbletea just dump text or push logs straight to an injected `chan` without claiming TTY dominance.
- **SSE Consumer Setup**: When `ClientMode == true` (no `app` backend running locally):
  1. Call `tinywasm/sse.NewClient(c.ClientURL)`.
  2. Implement an SSE Event Listener loop (e.g., `go func() { ... }()`) that listens to the `sse.Client` channel.
  3. Every time a log event arrives, unmarshal it into `tabContent` and push it to the `h.tabContentsChan` channel so `bubbletea` updates automatically.

### 3.2. Sending Keyboard Actions to Server (`mcpserve/action`)
- **Key Interception**: Locate the `Update(msg tea.Msg)` loop inside `devtui` (often in `update.go` or similar main `bubbletea` loop).
- **HTTP POST**: When the user presses shortcut keys like `r` (reload) or `d` (deploy), instead of invoking a direct struct handler from memory (since in client mode we don't have the `app` struct), we must:
  1. Detect if `ClientMode == true`.
  2. Perform an `http.Post` request to `http://localhost:3030/action?key=r`.
  3. The remote MCP Daemon will handle the actual logic.
- **`q` to Quit**: Same as above, but with `key=q`.
- **`Ctrl+C` behavior**: In `ClientMode`, `Ctrl+C` intercepts the Tea message `tea.KeyCtrlC` and simply calls `tea.Quit` (killing the local interface process). It does NOT post a shutdown to the server.

### 3.3. Removing Hardcoded Couplings
- Ensure UI components in `devtui` rely ONLY on `chan tabContent` or similar primitives. They shouldn't require concrete modules (like `devflow.GitClient` or `devwatch.DevWatch` directly) if they are just rendering remote layouts. (If they do, provide dummy/stub interface versions in client mode).

## 4. Diagram-Driven Testing (DDT)
As mandated by the `DEFAULT_LLM_SKILL.md`, the branching logic must be fully tested.
- **DDT Restrictions (Never block the CI)**: Do NOT invoke `tea.NewProgram().Run()` inside an automated test logic because Bubbletea will attempt to seize OS Terminals or hang the CI runner.
- You MUST test the pure event handlers, calling `Update(msg tea.KeyMsg)` programmatically.
- **Goal**: Instantiate `DevTUI` in `ClientMode`, inject a fake `httptest.Server` to act as the MCP API, programmatically send a `tea.KeyMsg('r')` simulate a keystroke, and verify that DevTUI successfully fired an HTTP POST to `/action?key=r`.
