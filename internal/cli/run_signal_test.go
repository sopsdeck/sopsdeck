package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// buildSopsdeckBin compiles the real binary so signal handling can be
// exercised as a genuine subprocess (kill() from outside the process).
func buildSopsdeckBin(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "sopsdeck")
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

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()

	waitFor(t, func() bool {
		raw, err := os.ReadFile(file)
		return err == nil && isEncryptedBytes(raw)
	}, 5*time.Second, "relock after SIGTERM")
}
