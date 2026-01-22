package devtui

import (
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/tinywasm/fmt"
)

func (t *DevTUI) ReturnFocus() error {

	time.Sleep(100 * time.Millisecond)

	pid := os.Getpid()

	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("xdotool", "search", "--pid", fmt.Convert(pid).String(), "windowactivate")
		return cmd.Run()

	case "darwin":
		cmd := exec.Command("osascript", "-e", fmt.Sprintf(`
            tell application "System Events"
                set frontmost of the first process whose unix id is %d to true
            end tell
        `, pid))
		return cmd.Run()

	case "windows":
		// Usando taskkill para verificar si el proceso existe y obtener su ventana
		cmd := exec.Command("cmd", "/C", fmt.Sprintf("tasklist /FI \"PID eq %d\" /FO CSV /NH", pid))
		return cmd.Run()

	default:
		return fmt.Err("focus unsupported platform")
	}

}

// GetTabSections returns all tab sections as a slice of any for interface compatibility
func (t *DevTUI) GetTabSections() []any {
	result := make([]any, len(t.TabSections))
	for i, section := range t.TabSections {
		result[i] = section
	}
	return result
}
