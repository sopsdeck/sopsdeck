package studio_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sopsdeck/internal/cli"
	"sopsdeck/internal/studio"
)

func run(u *studio.User, args ...string) (string, string, int) {
	var stdout, stderr bytes.Buffer
	var code int
	u.WithWorld(func() {
		code = cli.Main(args, os.Stdin, &stdout, &stderr, u.Getenv)
	})
	return stdout.String(), stderr.String(), code
}

func TestTeammateDecryptsAfterRecipientAddAndSync(t *testing.T) {
	s, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)

	alice, err := s.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	bobKeys, err := s.Identity("bob", "bob@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}

	env := filepath.Join(alice.Home, ".env.production")
	if _, stderr, code := run(alice, "set", "HELLO", "from-alice", "-f", env); code != 0 {
		t.Fatalf("set: %s", stderr)
	}
	if _, stderr, code := run(alice, "recipient", "add", bobKeys.PublicKey, "-f", env); code != 0 {
		t.Fatalf("recipient add: %s", stderr)
	}
	if _, stderr, code := run(alice, "commit", "-m", "share production", "-f", env); code != 0 {
		t.Fatalf("commit: %s", stderr)
	}
	if _, err := alice.Git("push", "-u", "origin", "main"); err != nil {
		t.Fatal(err)
	}

	bob, err := s.Clone("bob", "bob@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	if bob.PublicKey == "" {
		t.Fatal("bob identity missing")
	}

	stdout, stderr, code := run(bob, "get", "HELLO", "-f", filepath.Join(bob.Home, ".env.production"))
	if code != 0 {
		t.Fatalf("bob get: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "from-alice" {
		t.Fatalf("bob get %q", stdout)
	}
}

func TestPublishPutsPrefixedNamesOnFakeGitHub(t *testing.T) {
	s, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)

	alice, err := s.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(alice.Home, ".env.production")
	if _, stderr, code := run(alice, "set", "HELLO", "world", "-f", env); code != 0 {
		t.Fatalf("set: %s", stderr)
	}
	if _, stderr, code := run(alice, "publish", "-f", env, "--prefix", "SD_", "--yes"); code != 0 {
		t.Fatalf("publish: %s", stderr)
	}
	names := s.GitHub.Names()
	found := false
	for _, n := range names {
		if n == "SD_HELLO" {
			found = true
		}
	}
	if !found {
		t.Fatalf("github names=%v, want SD_HELLO", names)
	}
}

func TestFilesCommandListsStudioManagedFile(t *testing.T) {
	s, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	alice, err := s.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(alice.Home, ".env.production")
	if _, stderr, code := run(alice, "set", "HELLO", "world", "-f", env); code != 0 {
		t.Fatalf("set: %s", stderr)
	}
	stdout, stderr, code := run(alice, "files", alice.Home)
	if code != 0 {
		t.Fatalf("files: %s", stderr)
	}
	if !strings.Contains(stdout, ".env.production") {
		t.Fatalf("files stdout=%q", stdout)
	}
	if _, err := os.Stat(env); err != nil {
		t.Fatal(err)
	}
}
