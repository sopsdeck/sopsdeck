//go:build windows

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"unicode/utf16"

	"github.com/charmbracelet/x/xpty"
	"golang.org/x/sys/windows"
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

func relayPTYInput(pty io.Writer, input io.Reader) error {
	file, ok := input.(*os.File)
	if !ok {
		_, err := io.Copy(pty, input)
		return err
	}

	chars := make([]uint16, 1024)
	for {
		var read uint32
		if err := windows.ReadConsole(windows.Handle(file.Fd()), &chars[0], uint32(len(chars)), &read, nil); err != nil {
			return err
		}
		if read == 0 {
			continue
		}
		p := []byte(string(utf16.Decode(chars[:read])))
		written, err := pty.Write(p)
		if err != nil {
			return err
		}
		if written != len(p) {
			return io.ErrShortWrite
		}
	}
}

func closePTYParentSlave(xpty.Pty) {}

func interactiveResizeSignals() (<-chan os.Signal, func()) {
	return nil, func() {}
}

func isPTYClosedError(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) ||
		errors.Is(err, syscall.ERROR_BROKEN_PIPE) || errors.Is(err, windows.ERROR_INVALID_HANDLE)
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
