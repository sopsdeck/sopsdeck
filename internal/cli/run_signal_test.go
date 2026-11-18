package cli

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/xpty"
)

// buildSopsdeckBin compiles the real binary so signal handling can be
// exercised as a genuine subprocess (kill() from outside the process).
func buildSopsdeckBin(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "sopsdeck")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	out, err := exec.Command("go", "build", "-o", bin, "sopsdeck/cmd/sopsdeck").CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func waitFor(t *testing.T, cond func() bool, timeout time.Duration, what string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s did not happen within %s", what, timeout)
}

func TestRunRelocksWhenChildIsKilled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM process-group behavior is Unix-specific")
	}
	bin := buildSopsdeckBin(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "app.config.json")
	writeFixture(t, file, "hello.json")

	// Fake keychain + age identity so the subprocess can decrypt. SOPS only
	// reads age keys from the env, so hand the child the documented bridge:
	// SOPS_AGE_KEY_CMD='sopsdeck identity key'.
	keychain := filepath.Join(dir, "keychain")
	if err := os.MkdirAll(keychain, 0o700); err != nil {
		t.Fatal(err)
	}
	identity, err := os.ReadFile(testdata(t, "age.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keychain, "identity"), identity, 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "run", "-f", file, "--", "sleep", "30")
	cmd.Env = append(os.Environ(),
		"SOPSDECK_KEYCHAIN_DIR="+keychain,
		"SOPSDECK_STATE_DIR="+filepath.Join(dir, "state"),
		"SOPS_AGE_KEY_CMD="+bin+" identity key",
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	// Wait until the child is actually running with the file unlocked.
	waitFor(t, func() bool {
		raw, err := os.ReadFile(file)
		return err == nil && !isEncryptedBytes(raw)
	}, 5*time.Second, "transient unlock")

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()

	waitFor(t, func() bool {
		raw, err := os.ReadFile(file)
		return err == nil && isEncryptedBytes(raw)
	}, 5*time.Second, "relock after SIGTERM")
}

func TestRunRelocksWhenInteractiveChildReceivesInterrupt(t *testing.T) {
	bin := buildSopsdeckBin(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "app.config.json")
	writeFixture(t, file, "hello.json")
	if err := os.Chmod(file, 0o640); err != nil {
		t.Fatal(err)
	}
	initial, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	initialMode := initial.Mode().Perm()
	initialContents, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	age := testdata(t, "age.txt")
	child, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pty, err := xpty.NewPty(80, 24)
	if err != nil {
		t.Fatalf("create parent PTY: %v", err)
	}
	defer pty.Close()

	cmd := exec.Command(bin, "run", "-f", file, "--", child, "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(),
		"SOPS_AGE_KEY_FILE="+age,
		"SOPSDECK_TEST_HELPER=wait-interrupt",
	)
	if err := pty.Start(cmd); err != nil {
		t.Fatalf("start sopsdeck under parent PTY: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	ready := make(chan string, 1)
	readErr := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(pty).ReadString('\n')
		if err != nil {
			readErr <- err
			return
		}
		ready <- line
	}()
	select {
	case line := <-ready:
		if !strings.Contains(line, "READY") {
			t.Fatalf("interactive child did not become ready: %q", line)
		}
	case err := <-readErr:
		t.Fatalf("read interactive child readiness: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("interactive child did not become ready")
	}
	waitFor(t, func() bool {
		raw, err := os.ReadFile(file)
		return err == nil && !isEncryptedBytes(raw)
	}, 5*time.Second, "transient unlock")

	if _, err := io.WriteString(pty, "\x03"); err != nil {
		t.Fatalf("send Ctrl-C to interactive child: %v", err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := xpty.WaitProcess(waitCtx, cmd); err != nil {
		t.Fatalf("sopsdeck did not exit cleanly after child interrupt: %v", err)
	}

	if restored, err := os.ReadFile(file); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(restored, initialContents) {
		t.Fatal("interrupt did not restore original encrypted bytes")
	}
	if info, err := os.Stat(file); err != nil {
		t.Fatal(err)
	} else if got := info.Mode().Perm(); got != initialMode {
		t.Fatalf("restored mode=%#o want %#o", got, initialMode)
	}
	if _, err := os.Stat(transientRunLockPath(file)); !os.IsNotExist(err) {
		t.Fatalf("run lock was not removed: %v", err)
	}
}
