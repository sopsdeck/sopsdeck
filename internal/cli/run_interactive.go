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

type terminalStderr struct {
	io.Writer
	terminal *os.File
}

func (w *terminalStderr) TerminalFile() *os.File { return w.terminal }

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
		if terminalRestored {
			return nil
		}
		terminalRestored = true
		return term.Restore(int(in.Fd()), state)
	}
	defer func() { _ = restoreTerminal() }()
	fail := func(format string, args ...any) int {
		restoreErr := restoreTerminal()
		fmt.Fprintf(r.stderr, format, args...)
		if restoreErr != nil {
			fmt.Fprintf(r.stderr, "run: could not restore interactive terminal: %v\n", restoreErr)
		}
		return 1
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = r.env
	configureInteractiveChildCommand(cmd)
	if err := pty.Start(cmd); err != nil {
		return fail("run: %v\n", err)
	}
	closePTYParentSlave(pty)

	output := r.outputWriter(terminalFile(r.stderr))
	outputDone := make(chan error, 1)
	go func() { outputDone <- relayPTYOutput(pty, output) }()
	go func() { _ = relayPTYInput(pty, in) }()

	done := make(chan error, 1)
	go func() { done <- xpty.WaitProcess(context.Background(), cmd) }()

	result := r.waitInteractive(cmd, pty, out, outputDone, done)

	// Closing the PTY ends the output relay even if a descendant kept a slave
	// handle open. The Unix parent slave was closed after start so its master can
	// drain normally when the last child exits.
	closeErr := pty.Close()
	ptyClosed = true
	if result.outputPending {
		result.outputErr = <-outputDone
	}
	if result.outputFailure == nil && unexpectedPTYOutputError(result.outputErr) {
		result.outputFailure = result.outputErr
	}
	if msg := interactiveFinishError(result.outputFailure, output, closeErr); msg != "" {
		return fail("%s\n", msg)
	}
	if err := restoreTerminal(); err != nil {
		fmt.Fprintf(r.stderr, "run: could not restore interactive terminal: %v\n", err)
		return 1
	}
	if result.waitErr == nil {
		return 0
	}
	return childExitCode(result.waitErr, r.stderr)
}

type interactiveWaitResult struct {
	waitErr       error
	outputErr     error
	outputFailure error
	outputPending bool
}

// waitInteractive forwards signals and resizes until the child exits. An
// output relay failure cancels the child unless it is the expected read error
// after the child has already exited.
func (r childRunner) waitInteractive(cmd *exec.Cmd, pty xpty.Pty, out *os.File, outputDone, done <-chan error) interactiveWaitResult {
	resizeSignals, stopResize := interactiveResizeSignals()
	defer stopResize()

	result := interactiveWaitResult{outputPending: true}
	for {
		select {
		case sig, ok := <-r.signals:
			if ok && sig != nil {
				forwardChildSignal(cmd, sig)
			}
		case <-resizeSignals:
			if w, h, sizeErr := term.GetSize(int(out.Fd())); sizeErr == nil && w > 0 && h > 0 {
				_ = pty.Resize(w, h)
			}
		case result.outputErr = <-outputDone:
			outputDone = nil
			result.outputPending = false
			if result.outputErr == nil || isPTYClosedError(result.outputErr) {
				continue
			}
			if unexpectedPTYOutputError(result.outputErr) || cmd.ProcessState == nil {
				result.outputFailure = result.outputErr
				_ = cancelChildCommand(cmd)
			}
		case result.waitErr = <-done:
			return result
		}
	}
}

// unexpectedPTYOutputError reports output relay errors other than a closed PTY
// or a read error from the PTY master.
func unexpectedPTYOutputError(err error) bool {
	if err == nil || isPTYClosedError(err) {
		return false
	}
	var readErr ptyReadError
	return !errors.As(err, &readErr)
}

func interactiveFinishError(outputFailure error, output io.Writer, closeErr error) string {
	if outputFailure != nil {
		return fmt.Sprintf("run: could not relay interactive command output: %v", outputFailure)
	}
	if f, ok := output.(interface{ Flush() error }); ok {
		if err := f.Flush(); err != nil {
			return "run: could not write redacted command output"
		}
	}
	if closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
		return fmt.Sprintf("run: could not close interactive terminal: %v", closeErr)
	}
	return ""
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
