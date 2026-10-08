package cli

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/xpty"
)

func TestRunInteractiveRedactedCommandUsesPTY(t *testing.T) {
	age := testdata(t, "age.txt")
	envFile := testdata(t, "hello.env")
	mustUnsetenv(t, "HELLO")
	t.Setenv("SOPS_AGE_KEY_FILE", age)
	t.Setenv("SOPSDECK_TEST_HELPER", "interactive-cli")
	args, err := json.Marshal([]string{
		"run", "-f", envFile, "--",
		mustExecutable(t), "-test.run=^TestHelperProcess$", "--", "sopsdeck-test-interactive-child",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOPSDECK_TEST_CLI_ARGS", string(args))

	pty, err := xpty.NewPty(80, 24)
	if err != nil {
		t.Fatalf("create parent PTY: %v", err)
	}
	defer pty.Close()

	cmd := exec.Command(mustExecutable(t), "-test.run=^TestHelperProcess$")
	cmd.Env = os.Environ()
	if err := pty.Start(cmd); err != nil {
		t.Fatalf("start sopsdeck under parent PTY: %v", err)
	}

	chunks := make(chan []byte, 256)
	readDone := make(chan error, 1)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := pty.Read(buf)
			if n > 0 {
				chunks <- append([]byte(nil), buf[:n]...)
			}
			if err != nil {
				readDone <- err
				return
			}
		}
	}()

	var output strings.Builder
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for !strings.Contains(output.String(), "TTY prompt:") && !strings.Contains(output.String(), "TTY=false") {
		select {
		case chunk := <-chunks:
			output.Write(chunk)
		case err := <-readDone:
			t.Fatalf("parent PTY closed before interactive prompt (read error %v), output=%q", err, outputSummary(output.String()))
		case <-deadline.C:
			_ = cmd.Process.Kill()
			_ = xpty.WaitProcess(context.Background(), cmd)
			t.Fatalf("timed out waiting for interactive prompt, output=%q", outputSummary(output.String()))
		}
	}
	if strings.Contains(output.String(), "TTY=false") {
		_ = xpty.WaitProcess(context.Background(), cmd)
		t.Fatalf("redacted child did not receive a terminal: %q", outputSummary(output.String()))
	}

	answer := "answer\n"
	if runtime.GOOS == "windows" {
		answer = "answer\r"
	}
	if _, err := io.WriteString(pty, answer); err != nil {
		t.Fatalf("answer interactive prompt: %v", err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := xpty.WaitProcess(waitCtx, cmd); err != nil {
		t.Fatalf("sopsdeck interactive run failed: %v, output=%q", err, outputSummary(output.String()))
	}

	tail := "TAIL=" + strings.Repeat("x", 64*1024) + ":END"
	for !strings.Contains(output.String(), "HELLO=[sopsdeck:HELLO]") || !strings.Contains(output.String(), "RESPONSE=answer") || !strings.Contains(output.String(), tail) {
		select {
		case chunk := <-chunks:
			output.Write(chunk)
		case err := <-readDone:
			t.Fatalf("parent PTY closed before final output (read error %v), output=%q", err, outputSummary(output.String()))
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out waiting for final output: %q", outputSummary(output.String()))
		}
	}
	_ = pty.Close()

	for {
		select {
		case chunk := <-chunks:
			output.Write(chunk)
		case err := <-readDone:
			if err == nil || err == io.EOF || strings.Contains(strings.ToLower(err.Error()), "input/output error") || strings.Contains(strings.ToLower(err.Error()), "closed") {
				goto readComplete
			}
			t.Fatalf("read parent PTY: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out draining interactive output: %q", outputSummary(output.String()))
		}
	}

readComplete:
	got := output.String()
	if !strings.Contains(got, "TTY prompt:") || !strings.Contains(got, "RESPONSE=answer") {
		t.Fatalf("interactive prompt/response missing: %q", got)
	}
	if !strings.Contains(got, "HELLO=[sopsdeck:HELLO]") || strings.Contains(got, "HELLO=world") {
		t.Fatalf("secret was not redacted from interactive output: %q", got)
	}
	if !strings.Contains(got, tail) {
		t.Fatal("interactive PTY truncated the child's final output")
	}
}

func outputSummary(output string) string {
	if len(output) <= 512 {
		return output
	}
	return output[:256] + "…" + output[len(output)-256:]
}

func mustExecutable(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return path
}
