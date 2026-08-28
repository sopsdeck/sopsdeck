package cli

import (
	"bytes"
	"os"
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
