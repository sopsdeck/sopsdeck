package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncRefusesWhenBranchHasDiverged(t *testing.T) {
	bare := t.TempDir()
	runGit(t, bare, "init", "--bare")
	runGit(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")

	a := t.TempDir()
	setupWork(t, a, bare)
	envA := filepath.Join(a, "hello.env")
	writeFixture(t, envA, "hello.env")
	mustCommit(t, envA, "shared")
	runGit(t, a, "push", "-u", "origin", "main")

	b := t.TempDir()
	runGit(t, b, "clone", bare, ".")
	runGit(t, b, "config", "user.email", "test@sopsdeck.example")
	runGit(t, b, "config", "user.name", "Sopsdeck Test")

	writeFixture(t, filepath.Join(a, "hello.json"), "hello.json")
	mustCommit(t, filepath.Join(a, "hello.json"), "on-a")
	runGit(t, a, "push")

	writeFixture(t, filepath.Join(b, "hello.yaml"), "hello.yaml")
	mustCommit(t, filepath.Join(b, "hello.yaml"), "on-b")

	t.Chdir(b)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"sync"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatalf("expected non-zero exit on diverged branch, stderr=%q", stderr.String())
	}
	remoteLog := runGit(t, bare, "log", "--pretty=%s")
	if strings.Contains(remoteLog, "on-b") {
		t.Fatalf("pushed diverged commit; remote log=%q", remoteLog)
	}
	if !strings.Contains(remoteLog, "on-a") {
		t.Fatalf("remote log=%q, want on-a", remoteLog)
	}
}

func setupWork(t *testing.T, dir, bare string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@sopsdeck.example")
	runGit(t, dir, "config", "user.name", "Sopsdeck Test")
	runGit(t, dir, "checkout", "-b", "main")
	runGit(t, dir, "remote", "add", "origin", bare)
}

func writeFixture(t *testing.T, dest, name string) {
	t.Helper()
	src, err := os.ReadFile(testdata(t, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, src, 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustCommit(t *testing.T, file, message string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"commit", "-m", message, "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("commit %s exit %d stderr=%q", message, code, stderr.String())
	}
}
