package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncPushesWhenRemoteCanFastForward(t *testing.T) {
	bare := t.TempDir()
	runGit(t, bare, "init", "--bare")
	runGit(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")

	work := t.TempDir()
	runGit(t, work, "init")
	runGit(t, work, "config", "user.email", "test@sopsdeck.example")
	runGit(t, work, "config", "user.name", "Sopsdeck Test")
	runGit(t, work, "checkout", "-b", "main")
	runGit(t, work, "remote", "add", "origin", bare)

	env := filepath.Join(work, "hello.env")
	src, err := os.ReadFile(testdata(t, "hello.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env, src, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"commit", "-m", "first", "-f", env}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("first commit exit %d stderr=%q", code, stderr.String())
	}
	runGit(t, work, "push", "-u", "origin", "main")

	jsonFile := filepath.Join(work, "hello.json")
	js, err := os.ReadFile(testdata(t, "hello.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonFile, js, 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"commit", "-m", "second", "-f", jsonFile}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("second commit exit %d stderr=%q", code, stderr.String())
	}

	t.Chdir(work)
	stdout.Reset()
	stderr.Reset()
	code := Main([]string{"sync"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("sync exit %d stderr=%q", code, stderr.String())
	}

	log := runGit(t, bare, "log", "--pretty=%s")
	if !strings.Contains(log, "second") {
		t.Fatalf("bare log=%q, want second commit", log)
	}
}
