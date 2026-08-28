package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoryListsCommitsOnAManagedFile(t *testing.T) {
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
	got := stdout.String()
	if strings.Contains(got, "universe") || strings.Contains(got, "ENC[") {
		t.Fatalf("leaked values: %q", got)
	}
	if !strings.Contains(got, "seed production") || !strings.Contains(got, "rotate hello") {
		t.Fatalf("stdout=%q", got)
	}
}

func TestGetAtRevisionPrintsHistoricalValue(t *testing.T) {
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
	if len(lines) < 2 {
		t.Fatalf("history=%q", stdout.String())
	}
	oldest := strings.Fields(lines[len(lines)-1])
	if len(oldest) == 0 {
		t.Fatalf("history line=%q", lines[len(lines)-1])
	}
	rev := oldest[0]

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", env, "--at", rev}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get --at exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "world" {
		t.Fatalf("historical get=%q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("get HEAD exit %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "universe" {
		t.Fatalf("HEAD get=%q", stdout.String())
	}
}
