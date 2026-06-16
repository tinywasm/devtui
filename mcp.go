package devtui

import (
	"github.com/tinywasm/mcp"
)

// GetMCPTools returns MCP tools provided by DevTUI.
// This method is called by mcpserve to discover tools.
func (d *DevTUI) GetMCPTools() []mcp.Tool {
	return nil
}

// GetHandlerStates returns nil — DevTUI is a client, not a state server.
func (d *DevTUI) GetHandlerStates() []byte { return nil }

// DispatchAction returns false — actions are forwarded to the daemon, not dispatched locally.
func (d *DevTUI) DispatchAction(_, _ string) bool { return false }

// Name implements Loggable interface for MCP integration
func (d *DevTUI) Name() string {
	return "DEVTUI"
}

// SetLog implements Loggable interface for MCP integration
// This allows mcpserve to inject a capturing logger
func (d *DevTUI) SetLog(log func(message ...any)) {
	// Store in separate field to avoid interfering with TUI's Logger
	d.mcpLogger = log
}
