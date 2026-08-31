package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitStagesManagedFileAndRecordsMessage(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@sopsdeck.example")
	runGit(t, dir, "config", "user.name", "Sopsdeck Test")

	env := filepath.Join(dir, "hello.env")
	src, err := os.ReadFile(testdata(t, "hello.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env, src, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Main([]string{"commit", "-m", "add production secrets", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}

	subject := strings.TrimSpace(runGit(t, dir, "log", "-1", "--pretty=%s"))
	if subject != "add production secrets" {
		t.Fatalf("subject=%q", subject)
	}
	tracked := strings.TrimSpace(runGit(t, dir, "ls-files", "hello.env"))
	if tracked != "hello.env" {
		t.Fatalf("ls-files=%q", tracked)
	}
}

func TestCommitSkipsGpgWhenRepoAsksToSign(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@sopsdeck.example")
	runGit(t, dir, "config", "user.name", "Sopsdeck Test")
	runGit(t, dir, "config", "commit.gpgsign", "true")
	runGit(t, dir, "config", "gpg.program", filepath.Join(dir, "would-hang"))

	env := filepath.Join(dir, "hello.env")
	src, err := os.ReadFile(testdata(t, "hello.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env, src, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Main([]string{"commit", "-m", "unsigned", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
}

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
