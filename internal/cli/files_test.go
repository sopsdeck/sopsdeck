package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesListsDotenvInFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env.production"), []byte("HELLO=world\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"files", dir}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), ".env.production") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestFilesRejectsAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(path, []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"files", path}, os.Stdin, &stdout, &stderr, os.Getenv); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "not a folder") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestFilesRejectsExtraArgs(t *testing.T) {
	var stderr bytes.Buffer
	if code := Main([]string{"files", "a", "b"}, os.Stdin, &bytes.Buffer{}, &stderr, os.Getenv); code != 1 {
		t.Fatalf("exit %d", code)
	}
}
