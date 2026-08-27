package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncWithoutUpstreamExplainsMissingTracking(t *testing.T) {
	bare := t.TempDir()
	runGit(t, bare, "init", "--bare")
	runGit(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")

	work := t.TempDir()
	setupWork(t, work, bare)
	env := filepath.Join(work, "hello.env")
	writeFixture(t, env, "hello.env")
	mustCommit(t, env, "local")

	t.Chdir(work)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"sync"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatal("expected non-zero exit without upstream")
	}
	got := stderr.String()
	if strings.Contains(got, "git-pull(1)") || strings.Contains(got, "--set-upstream-to") {
		t.Fatalf("raw git help leaked: %q", got)
	}
	if !strings.Contains(got, "no upstream") {
		t.Fatalf("stderr=%q, want no upstream", got)
	}
}
