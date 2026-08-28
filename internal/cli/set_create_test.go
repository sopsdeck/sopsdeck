package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetCreatesEncryptedFileWhenMissing(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)
	t.Setenv("SOPSDECK_KEYCHAIN_DIR", state)
	mustUnsetenv(t, "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE")

	var stdout, stderr bytes.Buffer
	code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("create exit %d stderr=%q", code, stderr.String())
	}
	keyFile := filepath.Join(state, "identity")
	t.Setenv("SOPS_AGE_KEY_FILE", keyFile)

	envFile := filepath.Join(t.TempDir(), "new.env")
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"set", "HELLO", "world", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	raw, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("HELLO=world")) && !bytes.Contains(raw, []byte("ENC[")) {
		t.Fatal("wrote plaintext dotenv; want SOPS ciphertext")
	}
	if !bytes.Contains(raw, []byte("ENC[")) {
		t.Fatalf("file is not SOPS encrypted:\n%s", raw)
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "world" {
		t.Fatalf("get stdout=%q", stdout.String())
	}
}

func TestSetCreatesEmptyEncryptedManagedFileWhenMissing(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)
	t.Setenv("SOPSDECK_KEYCHAIN_DIR", state)
	mustUnsetenv(t, "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE")

	var stdout, stderr bytes.Buffer
	code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("create exit %d stderr=%q", code, stderr.String())
	}
	keyFile := filepath.Join(state, "identity")
	t.Setenv("SOPS_AGE_KEY_FILE", keyFile)

	folder := t.TempDir()
	envFile := filepath.Join(folder, ".env.staging")
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"set", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	raw, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("ENC[")) {
		t.Fatalf("file is not SOPS encrypted:\n%s", raw)
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "-f", envFile, "--output", "json"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	got := strings.TrimSpace(stdout.String())
	if got != "{}" {
		t.Fatalf("get json=%q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"files", folder}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("files exit %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), ".env.staging") {
		t.Fatalf("files stdout=%q", stdout.String())
	}
}

func TestSetRefusesEmptyCreateWhenManagedFileExists(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)
	t.Setenv("SOPSDECK_KEYCHAIN_DIR", state)
	mustUnsetenv(t, "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE")

	var stdout, stderr bytes.Buffer
	code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("create exit %d stderr=%q", code, stderr.String())
	}
	t.Setenv("SOPS_AGE_KEY_FILE", filepath.Join(state, "identity"))

	envFile := filepath.Join(t.TempDir(), ".env.staging")
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"set", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"set", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatal("set overwrote an existing Managed File")
	}
	if !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestSetCreatesNestedManagedFileWhenMissing(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)
	t.Setenv("SOPSDECK_KEYCHAIN_DIR", state)
	mustUnsetenv(t, "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE")

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("create exit %d stderr=%q", code, stderr.String())
	}
	t.Setenv("SOPS_AGE_KEY_FILE", filepath.Join(state, "identity"))

	envFile := filepath.Join(t.TempDir(), "apps", "web", ".env.nested")
	stdout.Reset()
	stderr.Reset()
	code := Main([]string{"set", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(envFile); err != nil {
		t.Fatal(err)
	}
}
