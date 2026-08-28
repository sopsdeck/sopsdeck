package cli

import (
	"fmt"
	"io"
	"os/exec"
)

func cmdHistory(args []string, stdout, stderr io.Writer) int {
	file := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "history: -f requires a file")
				return 1
			}
			file = args[i]
		default:
			fmt.Fprintln(stderr, "usage: sopsdeck history -f FILE")
			return 1
		}
	}
	if file == "" {
		fmt.Fprintln(stderr, "usage: sopsdeck history -f FILE")
		return 1
	}
	dir, rel, err := gitTrackedRel(file)
	if err != nil {
		fmt.Fprintf(stderr, "history: %v\n", err)
		return 1
	}
	cmd := exec.Command("git", "log", "--follow", "--pretty=%h %s", "--", rel)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(stderr, "history: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(out); err != nil {
		fmt.Fprintf(stderr, "history: %v\n", err)
		return 1
	}
	return 0
}
