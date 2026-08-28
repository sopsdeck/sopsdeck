package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestIdentityCreateWithoutBackupConfirmDoesNotPersist(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)

	var stdout, stderr bytes.Buffer
	code := Main([]string{"identity", "create"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatal("expected non-zero exit without backup confirmation")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("backup")) {
		t.Fatalf("stderr=%q, want a backup warning", stderr.String())
	}

	if _, err := os.Stat(filepath.Join(state, "age.txt")); err == nil {
		t.Fatal("persisted age.txt without backup confirmation")
	}
	if _, err := os.Stat(filepath.Join(state, "identity")); err == nil {
		t.Fatal("persisted keychain identity without backup confirmation")
	}
}

func TestIdentityCreateStoresKeySoDecryptWorksWithoutAgeFile(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)
	t.Setenv("SOPSDECK_KEYCHAIN_DIR", state)
	mustUnsetenv(t, "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY")

	var stdout, stderr bytes.Buffer
	code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("create exit %d stderr=%q", code, stderr.String())
	}
	pub := bytes.TrimSpace(stdout.Bytes())
	if !bytes.HasPrefix(pub, []byte("age1")) {
		t.Fatalf("stdout=%q, want age1 public key", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(state, "age.txt")); err == nil {
		t.Fatal("persisted age.txt; private key must live in the keychain")
	}

	plain := filepath.Join(t.TempDir(), "hello.env")
	if err := os.WriteFile(plain, []byte("HELLO=world\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	enc := filepath.Join(t.TempDir(), "hello.env")
	sops := exec.Command("sops", "--encrypt", "--input-type", "dotenv", "--output-type", "dotenv", "--age", string(pub), plain)
	out, err := sops.Output()
	if err != nil {
		t.Fatalf("sops encrypt: %v %s", err, out)
	}
	if err := os.WriteFile(enc, out, 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SOPS_AGE_KEY_CMD", "cat "+strconv.Quote(filepath.Join(state, "identity")))
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "HELLO", "-f", enc}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "world\n" {
		t.Fatalf("get stdout=%q", got)
	}
}

func TestIdentityImportWithBackupConfirmRestoresAccess(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)
	t.Setenv("SOPSDECK_KEYCHAIN_DIR", state)
	mustUnsetenv(t, "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE")

	var stdout, stderr bytes.Buffer
	code := Main([]string{"identity", "import", "-f", testdata(t, "age.txt"), "--confirmed-backup"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("import exit %d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(state, "age.txt")); err == nil {
		t.Fatal("import wrote age.txt; private key must live in the keychain")
	}

	t.Setenv("SOPS_AGE_KEY_CMD", "cat "+strconv.Quote(filepath.Join(state, "identity")))
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "HELLO", "-f", testdata(t, "hello.env")}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "world\n" {
		t.Fatalf("get stdout=%q", got)
	}
}

func TestIdentityKeyPrintsStoredIdentity(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)
	t.Setenv("SOPSDECK_KEYCHAIN_DIR", state)

	var stdout, stderr bytes.Buffer
	code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("create exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"identity", "key"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("identity key exit %d stderr=%q", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("AGE-SECRET-KEY-")) {
		t.Fatalf("stdout=%q, want AGE-SECRET-KEY", stdout.String())
	}
	if bytes.Contains(stderr.Bytes(), []byte("AGE-SECRET-KEY-")) {
		t.Fatalf("stderr leaked private key: %q", stderr.String())
	}
}
