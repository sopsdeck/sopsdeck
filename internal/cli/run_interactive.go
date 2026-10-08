package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/charmbracelet/x/xpty"
	"golang.org/x/term"
)

type capturedStderr struct {
	io.Writer
	terminal *os.File
}

func (w *capturedStderr) TerminalFile() *os.File { return w.terminal }

type ptyReadError struct{ err error }

func (e ptyReadError) Error() string { return e.err.Error() }
func (e ptyReadError) Unwrap() error { return e.err }

func terminalFile(w io.Writer) *os.File {
	var file *os.File
	switch wrapped := w.(type) {
	case *os.File:
		file = wrapped
	case interface{ TerminalFile() *os.File }:
		file = wrapped.TerminalFile()
	}
	if file == nil || !term.IsTerminal(int(file.Fd())) {
		return nil
	}
	return file
}

func (r childRunner) needsInteractivePTY() bool {
	if r.redact == nil || len(r.redact.pairs) == 0 {
		return false
	}
	in, ok := r.stdin.(*os.File)
	if !ok || !term.IsTerminal(int(in.Fd())) {
		return false
	}
	out, ok := r.stdout.(*os.File)
	if !ok || !term.IsTerminal(int(out.Fd())) {
		return false
	}
	return terminalFile(r.stderr) != nil
}

func (r childRunner) runInteractive(argv []string) int {
	in := r.stdin.(*os.File)
	out := r.stdout.(*os.File)
	width, height, err := term.GetSize(int(out.Fd()))
	if err != nil || width < 1 || height < 1 {
		width, height = 80, 24
	}
	pty, err := xpty.NewPty(width, height)
	if err != nil {
		fmt.Fprintf(r.stderr, "run: could not create interactive terminal: %v\n", err)
		return 1
	}
	ptyClosed := false
	defer func() {
		if !ptyClosed {
			_ = pty.Close()
		}
	}()

	state, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		fmt.Fprintf(r.stderr, "run: could not prepare interactive terminal: %v\n", err)
		return 1
	}
	terminalRestored := false
	restoreTerminal := func() error {
		if !terminalRestored {
			err := term.Restore(int(in.Fd()), state)
			terminalRestored = true
			return err
		}
		return nil
	}
	defer restoreTerminal()

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = r.env
	configureInteractiveChildCommand(cmd)
	if err := pty.Start(cmd); err != nil {
		restoreErr := restoreTerminal()
		fmt.Fprintf(r.stderr, "run: %v\n", err)
		if restoreErr != nil {
			fmt.Fprintf(r.stderr, "run: could not restore interactive terminal: %v\n", restoreErr)
		}
		return 1
	}
	closePTYParentSlave(pty)

	output := r.outputWriter(terminalFile(r.stderr))
	outputDone := make(chan error, 1)
	go func() { outputDone <- relayPTYOutput(pty, output) }()
	go func() { _ = relayPTYInput(pty, in) }()

	done := make(chan error, 1)
	go func() { done <- xpty.WaitProcess(context.Background(), cmd) }()
	resizeSignals, stopResize := interactiveResizeSignals()
	defer stopResize()

	var waitErr, outputErr, outputFailure error
	waiting := true
	for waiting {
		select {
		case sig, ok := <-r.signals:
			if ok && sig != nil {
				forwardChildSignal(cmd, sig)
			}
		case <-resizeSignals:
			if w, h, sizeErr := term.GetSize(int(out.Fd())); sizeErr == nil && w > 0 && h > 0 {
				_ = pty.Resize(w, h)
			}
		case outputErr = <-outputDone:
			outputDone = nil
			if outputErr != nil && !isPTYClosedError(outputErr) {
				var readErr ptyReadError
				if !errors.As(outputErr, &readErr) || cmd.ProcessState == nil {
					outputFailure = outputErr
					_ = cancelChildCommand(cmd)
				}
			}
		case waitErr = <-done:
			waiting = false
		}
	}

	// Closing the PTY ends the output relay even if a descendant kept a slave
	// handle open. The Unix parent slave was closed after start so its master can
	// drain normally when the last child exits.
	closeErr := pty.Close()
	ptyClosed = true
	if outputDone != nil {
		outputErr = <-outputDone
	}
	if outputFailure == nil && outputErr != nil && !isPTYClosedError(outputErr) {
		var readErr ptyReadError
		if !errors.As(outputErr, &readErr) {
			outputFailure = outputErr
		}
	}
	if outputFailure != nil {
		restoreErr := restoreTerminal()
		fmt.Fprintf(r.stderr, "run: could not relay interactive command output: %v\n", outputFailure)
		if restoreErr != nil {
			fmt.Fprintf(r.stderr, "run: could not restore interactive terminal: %v\n", restoreErr)
		}
		return 1
	}
	if f, ok := output.(interface{ Flush() error }); ok {
		if err := f.Flush(); err != nil {
			restoreErr := restoreTerminal()
			fmt.Fprintln(r.stderr, "run: could not write redacted command output")
			if restoreErr != nil {
				fmt.Fprintf(r.stderr, "run: could not restore interactive terminal: %v\n", restoreErr)
			}
			return 1
		}
	}
	if closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
		restoreErr := restoreTerminal()
		fmt.Fprintf(r.stderr, "run: could not close interactive terminal: %v\n", closeErr)
		if restoreErr != nil {
			fmt.Fprintf(r.stderr, "run: could not restore interactive terminal: %v\n", restoreErr)
		}
		return 1
	}
	if err := restoreTerminal(); err != nil {
		fmt.Fprintf(r.stderr, "run: could not restore interactive terminal: %v\n", err)
		return 1
	}
	if waitErr == nil {
		return 0
	}
	return childExitCode(waitErr, r.stderr)
}

func relayPTYOutput(input io.Reader, output io.Writer) error {
	buf := make([]byte, 32*1024)
	for {
		n, readErr := input.Read(buf)
		if n > 0 {
			written, writeErr := output.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			if written != n {
				return io.ErrShortWrite
			}
		}
		if readErr != nil {
			return ptyReadError{err: readErr}
		}
	}
}
