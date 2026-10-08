package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/term"
)

func fakeTestExecutable(t *testing.T, name, helper string) string {
	t.Helper()

	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(binDir, name)
	if err := os.WriteFile(path, data, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOPSDECK_TEST_HELPER", helper)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return path
}

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[len(os.Args)-1] == "sopsdeck-test-interactive-child" {
		stdinTTY := term.IsTerminal(int(os.Stdin.Fd()))
		stdoutTTY := term.IsTerminal(int(os.Stdout.Fd()))
		if !stdinTTY || !stdoutTTY {
			fmt.Fprintf(os.Stdout, "TTY=false stdin=%v stdout=%v\n", stdinTTY, stdoutTTY)
			os.Exit(17)
		}
		fmt.Fprint(os.Stdout, "TTY prompt: ")
		answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(18)
		}
		fmt.Fprintf(os.Stdout, "\nHELLO=%s RESPONSE=%s\nTAIL=%s:END\n", os.Getenv("HELLO"), strings.TrimSpace(answer), strings.Repeat("x", 64*1024))
		os.Exit(0)
	}

	switch os.Getenv("SOPSDECK_TEST_HELPER") {
	case "interactive-cli":
		var args []string
		if err := json.Unmarshal([]byte(os.Getenv("SOPSDECK_TEST_CLI_ARGS")), &args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(Main(args, os.Stdin, os.Stdout, os.Stderr, os.Getenv))
	case "wait-interrupt":
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		fmt.Fprintln(os.Stdout, "READY")
		<-signals
		signal.Stop(signals)
		os.Exit(0)
	case "gh-args":
		if err := os.WriteFile(os.Getenv("SOPSDECK_GH_ARGS"), []byte(strings.Join(os.Args[1:], "\n")+"\n"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	case "gh-token":
		if len(os.Args) == 3 && os.Args[1] == "auth" && os.Args[2] == "token" {
			fmt.Fprintln(os.Stdout, "gho_from_gh")
			os.Exit(0)
		}
		os.Exit(1)
	case "sops":
		for _, arg := range os.Args[1:] {
			fmt.Fprintln(os.Stdout, arg)
		}
		_, _ = io.Copy(os.Stdout, os.Stdin)
		fmt.Fprintln(os.Stderr, "upstream error")
		os.Exit(23)
	}
	os.Exit(m.Run())
}
