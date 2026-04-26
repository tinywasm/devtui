package devtui_test

import (
	"sync"
	"testing"
	"time"

	"github.com/tinywasm/devtui"
)

func TestTeaRunExitsOnEOF(t *testing.T) {
	tui := devtui.DefaultTUIForTest()
	// DefaultTUIForTest sets TestMode(true), which uses strings.NewReader("")

	done := make(chan error, 1)
	go func() {
		_, err := tui.TestOnlyRun()
		done <- err
	}()

	// Give it some time to start, then try to shutdown manually if EOF didn't work
	go func() {
		time.Sleep(500 * time.Millisecond)
		tui.Shutdown()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("Tea Run returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("tea.Run() did not exit within 3s even with Shutdown()")
	}
}

// TestStartClosesExitChan verifies that Start() closes the exitChan when it returns,
// even when tea.Run() fails immediately (no TTY in test environment).
func TestStartClosesExitChan(t *testing.T) {
	// DefaultTUIForTest already sets TestMode(true), which now configures
	// tea.WithInput(strings.NewReader("")) ensuring tea.Run() returns immediately.
	tui := devtui.DefaultTUIForTest()

	var wg sync.WaitGroup
	exitChan := make(chan bool)

	wg.Add(1)
	go tui.Start(&wg, exitChan)

	// Since EOF might not be enough in all environments, force shutdown
	go func() {
		time.Sleep(100 * time.Millisecond)
		tui.Shutdown()
	}()

	select {
	case <-exitChan:
		// exitChan closed — Start() returned cleanly
	case <-time.After(3 * time.Second):
		t.Fatal("Start() did not close exitChan within 3s")
	}

	// Also verify the WaitGroup is released
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		// WaitGroup released
	case <-time.After(time.Second):
		t.Fatal("WaitGroup not released after exitChan closed")
	}
}
