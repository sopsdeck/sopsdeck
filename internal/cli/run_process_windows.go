//go:build windows

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/charmbracelet/x/xpty"
)

func configureChildCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func configureInteractiveChildCommand(cmd *exec.Cmd) {
	// ConPTY translates Ctrl-C input into a console control event. Starting the
	// attached process group with CREATE_NEW_PROCESS_GROUP would suppress that
	// event, so let the pseudo-console deliver it to its foreground process.
	cmd.SysProcAttr = &syscall.SysProcAttr{}
}

func closePTYParentSlave(xpty.Pty) {}

func interactiveResizeSignals() (<-chan os.Signal, func()) {
	return nil, func() {}
}

func isPTYClosedError(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) || errors.Is(err, syscall.ERROR_BROKEN_PIPE)
}

func cancelChildCommand(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}
	forwardChildSignal(cmd, os.Interrupt)
	return cmd.Process.Kill()
}

func forwardChildSignal(cmd *exec.Cmd, _ os.Signal) {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GenerateConsoleCtrlEvent")
	if ok, _, _ := proc.Call(syscall.CTRL_BREAK_EVENT, uintptr(cmd.Process.Pid)); ok == 0 {
		_ = cmd.Process.Kill()
	}
}

func childExitCode(err error, stderr io.Writer) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	fmt.Fprintf(stderr, "run: %v\n", err)
	return 1
}
