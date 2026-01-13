package devtui

import "github.com/tinywasm/fmt"

const HandlerNameWidth = 15

// padHandlerName pads the handler name to a fixed width, centering it.
// If the name is longer than width, it truncates it.
func padHandlerName(name string, width int) string {
	if len(name) >= width {
		return name[:width]
	}
	padding := width - len(name)
	leftPad := padding / 2
	rightPad := padding - leftPad
	return fmt.Convert(" ").Repeat(leftPad).String() + name + fmt.Convert(" ").Repeat(rightPad).String()
}
