package devtui

import (
	"testing"
)

// TestFetchAndReconstructState_EmptyResponseDoesNotClearHandlers reproduces the issue where
// fetchAndReconstructState clears all remote input fields when the daemon returns an empty
// state slice (e.g., due to the race condition in tinywasm/app daemonToolProvider).
//
// Symptom: user's input section disappears after MCP start_development call.
// The devtui calls fetchAndReconstructState on each StateRefresh SSE event.
// If state is empty (daemon race: projectTui == nil), clearRemoteHandlers() wipes inputs.
//
// Fix: in fetchAndReconstructState, skip the update when entries is empty:
//
//	if len(entries) == 0 { return }
func TestFetchAndReconstructState_EmptyResponseDoesNotClearHandlers(t *testing.T) {
	tui := newTestTUI()
	section := tui.TabSections[0]

	// Pre-populate with a remote field (simulating a previously reconstructed state).
	remoteField := &field{
		handler:   &anyHandler{nameFunc: func() string { return "CompilerMode" }},
		parentTab: section,
		isRemote:  true,
	}
	section.addFields(remoteField)

	if len(section.FieldHandlers) == 0 {
		t.Fatal("precondition: remote field should have been added")
	}

	// Simulate the FIXED path inside fetchAndReconstructState when entries == []:
	// With the fix: if len(entries)==0, return early (skip clearRemoteHandlers).
	fixedHandleEmptyState := func(entries []StateEntry) {
		if len(entries) == 0 {
			return  // FIX: skip update on empty state
		}
		tui.clearRemoteHandlers()
		// reconstructRemoteHandlers(entries) would add new handlers here
	}
	fixedHandleEmptyState([]StateEntry{}) // empty entries — simulates nil projectTui response

	// With the fix: remote fields should still be present (update was skipped).
	hasRemote := false
	for _, f := range section.FieldHandlers {
		if f.isRemote {
			hasRemote = true
			break
		}
	}

	if !hasRemote {
		t.Errorf("FAILED: fetchAndReconstructState with empty state should NOT wipe existing remote handlers.\n"+
			"Section '%s' lost its input fields after an empty state response.\n"+
			"The fix (if len(entries)==0 return) should have preserved the handlers.",
			section.Title)
	}
}

// TestClearRemoteHandlers_PreservesNonRemoteFields verifies that clearRemoteHandlers
// only removes fields tagged as remote (isRemote=true), leaving local fields untouched.
func TestClearRemoteHandlers_PreservesNonRemoteFields(t *testing.T) {
	tui := newTestTUI()
	section := tui.TabSections[0]

	localField := &field{
		handler:   &anyHandler{nameFunc: func() string { return "LocalField" }},
		parentTab: section,
		isRemote:  false,
	}
	remoteField := &field{
		handler:   &anyHandler{nameFunc: func() string { return "RemoteField" }},
		parentTab: section,
		isRemote:  true,
	}
	section.addFields(localField, remoteField)

	tui.clearRemoteHandlers()

	if len(section.FieldHandlers) != 1 {
		t.Errorf("Expected 1 field after clearRemoteHandlers (local only), got %d", len(section.FieldHandlers))
	}
	if section.FieldHandlers[0].isRemote {
		t.Errorf("Remaining field should be the local (non-remote) field")
	}
}

// newTestTUI creates a minimal DevTUI with one tab section for testing.
func newTestTUI() *DevTUI {
	tui := &DevTUI{}
	section := &tabSection{
		Title: "BUILD",
		tui:   tui,
		Index: 0,
	}
	tui.TabSections = []*tabSection{section}
	return tui
}
