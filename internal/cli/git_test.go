package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func runGitCommit(t *testing.T, dir, subject string) {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "--allow-empty", "-m", subject)
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
