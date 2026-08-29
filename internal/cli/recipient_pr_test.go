package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
)

func TestRecipientRequestOpensMetadataOnlyPR(t *testing.T) {
	bare := t.TempDir()
	runGit(t, bare, "init", "--bare")
	runGit(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")
	work := t.TempDir()
	setupWork(t, work, bare)
	env := filepath.Join(work, "hello.env")
	writeFixture(t, env, "hello.env")
	mustCommit(t, env, "initial secrets")
	runGit(t, work, "push", "-u", "origin", "main")

	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	argsFile := fakeGH(t)
	t.Chdir(work)
	var stdout, stderr bytes.Buffer
	code := Main(
		[]string{"recipient", "request", id.Recipient().String(), "--name", "Bob Builder", "-f", env},
		os.Stdin,
		&stdout,
		&stderr,
		os.Getenv,
	)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if branch := strings.TrimSpace(runGit(t, work, "branch", "--show-current")); branch != "main" {
		t.Fatalf("left worktree on %q", branch)
	}
	runGit(t, bare, "show-ref", "--verify", "refs/heads/sopsdeck/request-bob-builder")
	if changed := strings.TrimSpace(runGit(t, bare, "diff", "main...sopsdeck/request-bob-builder", "--name-only")); changed != "" {
		t.Fatalf("request PR changed files: %q", changed)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"pr", "create", "Bob Builder", id.Recipient().String(), "hello.env"} {
		if !strings.Contains(string(args), want) {
			t.Fatalf("gh args missing %q:\n%s", want, args)
		}
	}
}

func TestRecipientGrantOpensReencryptPR(t *testing.T) {
	bare := t.TempDir()
	runGit(t, bare, "init", "--bare")
	runGit(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")
	work := t.TempDir()
	setupWork(t, work, bare)
	env := filepath.Join(work, "hello.env")
	writeFixture(t, env, "hello.env")
	mustCommit(t, env, "initial secrets")
	runGit(t, work, "push", "-u", "origin", "main")
	aliceKey, err := filepath.Abs(testdata(t, "age.txt"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOPS_AGE_KEY_FILE", aliceKey)

	bob, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	argsFile := fakeGH(t)
	t.Chdir(work)
	var stdout, stderr bytes.Buffer
	code := Main(
		[]string{"recipient", "grant", bob.Recipient().String(), "--name", "Bob", "-f", env},
		os.Stdin,
		&stdout,
		&stderr,
		os.Getenv,
	)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if branch := strings.TrimSpace(runGit(t, work, "branch", "--show-current")); branch != "main" {
		t.Fatalf("left worktree on %q", branch)
	}
	branchFile := runGit(t, bare, "show", "sopsdeck/access-bob:hello.env")
	copyPath := filepath.Join(t.TempDir(), "hello.env")
	if err := os.WriteFile(copyPath, []byte(branchFile), 0o600); err != nil {
		t.Fatal(err)
	}
	bobKey := filepath.Join(t.TempDir(), "age.txt")
	if err := os.WriteFile(bobKey, []byte(bob.String()+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOPS_AGE_KEY_FILE", bobKey)
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", copyPath}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("bob decrypt exit %d stderr=%q", code, stderr.String())
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "Grant Bob access") {
		t.Fatalf("gh args:\n%s", args)
	}
}

func fakeGH(t *testing.T) string {
	t.Helper()
	bin := t.TempDir()
	argsFile := filepath.Join(bin, "args")
	script := []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$SOPSDECK_GH_ARGS\"\n")
	if err := os.WriteFile(filepath.Join(bin, "gh"), script, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOPSDECK_GH_ARGS", argsFile)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return argsFile
}
