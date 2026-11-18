package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunUnlocksStructuredFileForChildAndRelocks(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)
	mustUnsetenv(t, "HELLO")

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

	var stdout, stderr bytes.Buffer
	code := Main([]string{"run", "-f", file, "--", "cat", file}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("run exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "{\n\t\"HELLO\": \"[sopsdeck:HELLO]\"\n}\n" {
		t.Fatalf("child saw unlocked file=%q", got)
	}

	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !isEncryptedBytes(after) {
		t.Fatalf("file left unlocked after run: %.80s", after)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != initialMode {
		t.Fatalf("restored mode=%#o want %#o", got, initialMode)
	}
	if _, err := os.Stat(transientRunLockPath(file)); !os.IsNotExist(err) {
		t.Fatalf("run lock was not removed: %v", err)
	}
}

func TestRunRestoresStructuredFileWhenChildCannotStart(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	dir := t.TempDir()
	file := filepath.Join(dir, "app.config.json")
	writeFixture(t, file, "hello.json")
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Main([]string{"run", "-f", file, "--", "sopsdeck-missing-fixture-command"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatalf("expected start failure, stderr=%q", stderr.String())
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("start failure did not restore the original encrypted bytes")
	}
}

func TestTransientUnlockPreservesConcurrentFileChanges(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	file := filepath.Join(t.TempDir(), "app.config.json")
	writeFixture(t, file, "hello.json")
	var stderr bytes.Buffer
	u := &transientUnlock{file: file}
	if err := u.open(&stderr); err != nil {
		t.Fatal(err)
	}
	changed := []byte("{\"HELLO\":\"concurrent edit\"}\n")
	if err := os.WriteFile(file, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := u.close(&stderr); err == nil {
		t.Fatal("expected concurrent modification error")
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, changed) {
		t.Fatalf("concurrent change was overwritten: %q", after)
	}
}

func TestWriteAtomicRejectsFileHeldByRun(t *testing.T) {
	file := filepath.Join(t.TempDir(), "app.config.json")
	if err := os.WriteFile(transientRunLockPath(file), []byte("123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(file, []byte("new content")); err == nil {
		t.Fatal("writeAtomic ignored the active run lock")
	}
}
