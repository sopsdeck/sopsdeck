package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTeamInitWritesNavigableWorktrees(t *testing.T) {
	hostEmail := hostGitEmail(t)
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Main([]string{"team", "init", dir}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	alice := filepath.Join(dir, "alice-home", "checkout")
	bob := filepath.Join(dir, "bob-home", "checkout")
	if !strings.Contains(stdout.String(), alice) || !strings.Contains(stdout.String(), bob) {
		t.Fatalf("stdout=%q, want worktree paths", stdout.String())
	}
	if !strings.Contains(stdout.String(), "alice-home") {
		t.Fatalf("stdout=%q, want isolated home", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "alice.env")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "paths.txt")); err != nil {
		t.Fatal(err)
	}
	if hostGitEmail(t) != hostEmail {
		t.Fatal("host Git identity changed")
	}
}

func TestTeamProjectPrintsTwoWorktrees(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"team", "init", dir}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("init exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code := Main([]string{"team", "project", dir, "myapp"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("project exit %d stderr=%q", code, stderr.String())
	}
	alice := filepath.Join(dir, "alice-home", "myapp")
	bob := filepath.Join(dir, "bob-home", "myapp")
	if !strings.Contains(stdout.String(), alice) || !strings.Contains(stdout.String(), bob) {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestTeamUsageOnBadArgs(t *testing.T) {
	var stderr bytes.Buffer
	if code := Main([]string{"team"}, os.Stdin, bytes.NewBuffer(nil), &stderr, os.Getenv); code == 0 {
		t.Fatal("expected non-zero")
	}
	if !strings.Contains(stderr.String(), "usage: sopsdeck team") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func hostGitEmail(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "config", "--global", "--get", "user.email").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
