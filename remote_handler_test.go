//go:build !wasm

package devtui

import (
	"testing"

	"github.com/tinywasm/mcp"
)

// TestRemoteField_RegistersShortcuts verifies shortcuts from StateEntry are registered
func TestRemoteField_RegistersShortcuts(t *testing.T) {
	tui := &DevTUI{
		shortcutRegistry: newShortcutRegistry(),
		TabSections:      []*tabSection{{index: 0, title: "BUILD"}},
	}

	entry := StateEntry{
		TabTitle:    "BUILD",
		HandlerName: "WASM",
		HandlerType: HandlerTypeEdit,
		Shortcut:    "WASM",
		Shortcuts: []map[string]string{
			{"L": "Large"},
			{"M": "Medium"},
			{"S": "Small"},
		},
	}

	section := tui.TabSections[0]
	client := &mcp.Client{}

	f := newRemoteField(entry, client, section, tui)

	if f == nil {
		t.Fatal("newRemoteField returned nil")
	}

	// Verify shortcuts were registered
	for _, m := range entry.Shortcuts {
		for key := range m {
			shortcut, exists := tui.shortcutRegistry.Get(key)
			if !exists {
				t.Errorf("shortcut %q was not registered", key)
				continue
			}
			if shortcut.HandlerName != "WASM" {
				t.Errorf("shortcut %q: expected HandlerName='WASM', got %q", key, shortcut.HandlerName)
			}
			if shortcut.Value != key {
				t.Errorf("shortcut %q: expected Value=%q, got %q", key, key, shortcut.Value)
			}
		}
	}
}

// TestRemoteField_DispatchesWithHandlerName verifies field created from StateEntry uses handler name as key
func TestRemoteField_DispatchesWithHandlerName(t *testing.T) {
	entry := StateEntry{
		TabTitle:    "BUILD",
		HandlerName: "WasmClient",
		HandlerType: HandlerTypeEdit,
		Shortcut:    "WasmClient",
		Label:       "WASM Mode",
		Value:       "M",
	}

	section := &tabSection{index: 0, title: "BUILD"}
	// Pass nil client to avoid dispatch calls; we just verify field construction
	f := newRemoteField(entry, nil, section, nil)

	if f == nil {
		t.Fatal("newRemoteField returned nil")
	}

	// Verify field was created with correct handler type
	if f.handler.handlerType != handlerTypeEdit {
		t.Errorf("expected handlerTypeEdit, got %d", f.handler.handlerType)
	}

	// Verify isRemote flag is set
	if !f.isRemote {
		t.Error("expected isRemote=true")
	}

	// Verify handler name is accessible
	if f.handler.Name() != "WasmClient" {
		t.Errorf("expected handler.Name()='WasmClient', got %q", f.handler.Name())
	}
}
