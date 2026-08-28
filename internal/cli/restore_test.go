package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestoreCopiesHistoricalValuesWithoutCommitting(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
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
	if code := Main([]string{"commit", "-m", "seed production", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("first commit exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"set", "HELLO", "universe", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"commit", "-m", "rotate hello", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("second commit exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"history", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("history exit %d stderr=%q", code, stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	oldest := strings.Fields(lines[len(lines)-1])
	rev := oldest[0]

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"restore", "-f", env, "--at", rev}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("restore exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get after restore exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "world" {
		t.Fatalf("restored get=%q", stdout.String())
	}

	subject := strings.TrimSpace(runGit(t, dir, "log", "-1", "--pretty=%s"))
	if subject != "rotate hello" {
		t.Fatalf("restore committed: subject=%q", subject)
	}
	status := runGit(t, dir, "status", "--porcelain", "--", "hello.env")
	if strings.TrimSpace(status) == "" {
		t.Fatal("restore left the worktree clean")
	}
}
