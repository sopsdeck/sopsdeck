package cli

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

func copyToClipboard(text string) error {
	commands := clipboardCommands()
	var lastErr error
	for _, command := range commands {
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		return fmt.Errorf("no clipboard command is available")
	}
	return fmt.Errorf("native clipboard: %w", lastErr)
}

func clipboardCommands() [][]string {
	switch runtime.GOOS {
	case "darwin":
		return [][]string{{"pbcopy"}}
	case "windows":
		return [][]string{{"clip"}}
	default:
		return [][]string{{"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}}
	}
}

func cmdCopy(args []string, stdin io.Reader, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: sopsdeck copy")
		return 1
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "copy: %v\n", err)
		return 1
	}
	if err := copyToClipboard(string(data)); err != nil {
		fmt.Fprintf(stderr, "copy: %v\n", err)
		return 1
	}
	return 0
}
