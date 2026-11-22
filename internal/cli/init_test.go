package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitExistingDotenv(t *testing.T) {
	identity, err := filepath.Abs(testdata(t, "age.txt"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("SOPS_AGE_KEY_FILE", identity)
	mustWriteFile(t, ".env", "TOKEN=hello\n")
	mustWriteFile(t, ".env.example", "TOKEN=example\n")
	var out, errOut bytes.Buffer
	if code := Main([]string{"init", "--yes"}, strings.NewReader(""), &out, &errOut, os.Getenv); code != 0 {
		t.Fatalf("init: %s", errOut.String())
	}
	if !isEncryptedBytes([]byte(mustReadFile(t, ".env"))) {
		t.Fatal("dotenv not encrypted")
	}
	if got := mustReadFile(t, ".env.example"); got != "TOKEN=example\n" {
		t.Fatalf("template changed: %q", got)
	}
	if _, err := os.Stat(".sopsdeck.toml"); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := Main([]string{"set", "TOKEN", "updated"}, strings.NewReader(""), &out, &errOut, os.Getenv); code != 0 {
		t.Fatalf("set: %s", errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := Main([]string{"get", "TOKEN"}, strings.NewReader(""), &out, &errOut, os.Getenv); code != 0 || strings.TrimSpace(out.String()) != "updated" {
		t.Fatalf("get: %q %s", out.String(), errOut.String())
	}
}

func TestInitRequiresBackupAndConfirmation(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("SOPSDECK_KEYCHAIN_DIR", filepath.Join(root, "keychain"))
	mustUnsetenv(t, "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD")
	mustWriteFile(t, ".env", "TOKEN=hello\n")
	var out, errOut bytes.Buffer
	if code := Main([]string{"init"}, strings.NewReader("y\nn\n"), &out, &errOut, os.Getenv); code == 0 {
		t.Fatal("expected backup refusal")
	}
	if _, err := os.Stat(filepath.Join(root, "keychain", "identity")); !os.IsNotExist(err) {
		t.Fatal("identity saved without backup confirmation")
	}
	if got := mustReadFile(t, ".env"); got != "TOKEN=hello\n" {
		t.Fatal("file changed before confirmation")
	}
	out.Reset()
	errOut.Reset()
	if code := Main([]string{"init", "--yes"}, strings.NewReader(""), &out, &errOut, os.Getenv); code == 0 {
		t.Fatal("--yes bypassed backup confirmation")
	}
}

func TestInitCreatesBackedUpIdentity(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("SOPSDECK_KEYCHAIN_DIR", filepath.Join(root, "keychain"))
	mustUnsetenv(t, "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD")
	mustWriteFile(t, ".env", "TOKEN=hello\n")
	var out, errOut bytes.Buffer
	if code := Main([]string{"init"}, strings.NewReader("y\ny\n"), &out, &errOut, os.Getenv); code != 0 {
		t.Fatalf("init: %s", errOut.String())
	}
	if !isEncryptedBytes([]byte(mustReadFile(t, ".env"))) {
		t.Fatal("dotenv not encrypted")
	}
	if _, err := os.Stat(filepath.Join(root, "keychain", "identity")); err != nil {
		t.Fatal(err)
	}
}
