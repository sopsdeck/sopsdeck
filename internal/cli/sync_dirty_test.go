package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncRefusesDirtyManagedFile(t *testing.T) {
	bare := t.TempDir()
	runGit(t, bare, "init", "--bare")
	runGit(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")

	work := t.TempDir()
	setupWork(t, work, bare)
	env := filepath.Join(work, "hello.env")
	writeFixture(t, env, "hello.env")
	mustCommit(t, env, "first")
	runGit(t, work, "push", "-u", "origin", "main")

	if err := os.WriteFile(env, append(mustRead(t, env), []byte("\n# dirty\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Chdir(work)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"sync"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatal("expected non-zero exit when dirty")
	}
	if !strings.Contains(stderr.String(), "commit local Managed File") {
		t.Fatalf("stderr=%q", stderr.String())
	}
	log := runGit(t, bare, "log", "--pretty=%s")
	if strings.Contains(log, "dirty") {
		t.Fatalf("pushed dirty worktree; log=%q", log)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
