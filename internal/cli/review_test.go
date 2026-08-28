package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewShowsPlaintextSemanticDiffOfUncommittedManagedFile(t *testing.T) {
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
	if code := Main([]string{"commit", "-m", "seed", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("commit exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"set", "HELLO", "universe", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"review", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("review exit %d stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	if strings.Contains(got, "ENC[") || strings.Contains(got, "sops_") {
		t.Fatalf("leaked ciphertext: %q", got)
	}
	if !strings.Contains(got, "HELLO") || !strings.Contains(got, "world") || !strings.Contains(got, "universe") {
		t.Fatalf("stdout=%q", got)
	}
}

func TestReviewShowsThreeWayWhenManagedFileConflicts(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
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
	if code := Main([]string{"commit", "-m", "seed", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("seed commit exit %d stderr=%q", code, stderr.String())
	}

	runGit(t, dir, "checkout", "-b", "theirs")
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"set", "HELLO", "from-theirs", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("theirs set exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"commit", "-m", "theirs", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("theirs commit exit %d stderr=%q", code, stderr.String())
	}

	runGit(t, dir, "checkout", "main")
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"set", "HELLO", "from-ours", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("ours set exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"commit", "-m", "ours", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("ours commit exit %d stderr=%q", code, stderr.String())
	}

	merge := exec.Command("git", "merge", "--no-edit", "theirs")
	merge.Dir = dir
	if err := merge.Run(); err == nil {
		t.Fatal("expected merge conflict")
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"review", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("review exit %d stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	if strings.Contains(got, "ENC[") || strings.Contains(got, "<<<<<<<") {
		t.Fatalf("leaked ciphertext or conflict markers: %q", got)
	}
	if !strings.Contains(got, "HELLO") || !strings.Contains(got, "world") || !strings.Contains(got, "from-ours") || !strings.Contains(got, "from-theirs") {
		t.Fatalf("stdout=%q", got)
	}
}
