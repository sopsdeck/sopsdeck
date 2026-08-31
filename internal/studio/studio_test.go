package studio_test

import (
	"bytes"
	"os"
	"os/exec"
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

func TestTeammateLosesAccessAfterRecipientRemoveAndSync(t *testing.T) {
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
	bobEnv := filepath.Join(bob.Home, ".env.production")
	if _, stderr, code := run(bob, "get", "HELLO", "-f", bobEnv); code != 0 {
		t.Fatalf("bob get before remove: %s", stderr)
	}

	if _, stderr, code := run(alice, "recipient", "remove", bobKeys.PublicKey, "-f", env); code != 0 {
		t.Fatalf("recipient remove: %s", stderr)
	}
	if _, stderr, code := run(alice, "commit", "-m", "drop bob", "-f", env); code != 0 {
		t.Fatalf("commit remove: %s", stderr)
	}
	if _, err := alice.Git("push", "origin", "main"); err != nil {
		t.Fatal(err)
	}

	if stdout, stderr, code := run(bob, "get", "HELLO", "-f", bobEnv); code != 0 {
		t.Fatalf("bob get of un-synced copy: %s", stderr)
	} else if strings.TrimSpace(stdout) != "from-alice" {
		t.Fatalf("bob copy %q", stdout)
	}

	if _, stderr, code := run(bob, "sync"); code != 0 {
		t.Fatalf("bob sync: %s", stderr)
	}
	if _, stderr, code := run(bob, "get", "HELLO", "-f", bobEnv); code == 0 {
		t.Fatalf("bob still has Access after sync: %s", stderr)
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

func TestPrepareWritesNavigableHomesWithoutTouchingHostGitIdentity(t *testing.T) {
	hostName := hostGit(t, "user.name")
	hostEmail := hostGit(t, "user.email")
	root := t.TempDir()
	s, alice, bob, err := studio.Prepare(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)

	if alice.Home != filepath.Join(root, "alice-home", "checkout") || bob.Home != filepath.Join(root, "bob-home", "checkout") {
		t.Fatalf("worktrees alice=%q bob=%q", alice.Home, bob.Home)
	}
	if alice.ConfigHome != filepath.Join(root, "alice-home") {
		t.Fatalf("alice home %q", alice.ConfigHome)
	}
	if _, err := os.Stat(filepath.Join(root, "alice.env")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "paths.txt")); err != nil {
		t.Fatal(err)
	}
	aliceEmail := strings.TrimSpace(runGit(t, alice.Home, "config", "--get", "user.email"))
	bobEmail := strings.TrimSpace(runGit(t, bob.Home, "config", "--get", "user.email"))
	if aliceEmail != "alice@sopsdeck.example" || bobEmail != "bob@sopsdeck.example" {
		t.Fatalf("worktree emails alice=%q bob=%q", aliceEmail, bobEmail)
	}
	if hostGit(t, "user.name") != hostName || hostGit(t, "user.email") != hostEmail {
		t.Fatal("host Git identity changed")
	}

	scratch := filepath.Join(alice.ConfigHome, "scratch")
	if err := os.MkdirAll(scratch, 0o700); err != nil {
		t.Fatal(err)
	}
	gitWithHome(t, alice.ConfigHome, scratch, "init", "-b", "main")
	got := strings.TrimSpace(gitWithHome(t, alice.ConfigHome, scratch, "config", "--get", "user.email"))
	if got != "alice@sopsdeck.example" {
		t.Fatalf("new repo in alice-home email=%q", got)
	}
}

func TestPrepareTeammateCanRunAfterGrant(t *testing.T) {
	s, alice, bob, err := studio.Prepare(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)

	env := filepath.Join(alice.Home, ".env.production")
	if _, stderr, code := run(alice, "set", "HELLO", "from-alice", "-f", env); code != 0 {
		t.Fatalf("set: %s", stderr)
	}
	if _, stderr, code := run(alice, "recipient", "add", bob.PublicKey, "--name", "Bob", "-f", env); code != 0 {
		t.Fatalf("recipient add: %s", stderr)
	}
	if _, stderr, code := run(alice, "commit", "-m", "share production", "-f", env); code != 0 {
		t.Fatalf("commit: %s", stderr)
	}
	if _, err := alice.Git("push", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := run(bob, "sync"); code != 0 {
		t.Fatalf("bob sync: %s", stderr)
	}
	mustUnsetenv(t, "HELLO")
	stdout, stderr, code := run(bob, "run", "-f", filepath.Join(bob.Home, ".env.production"), "--", "printenv", "HELLO")
	if code != 0 {
		t.Fatalf("bob run: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "from-alice" {
		t.Fatalf("bob run %q", stdout)
	}
}

func TestPrepareIsIdempotent(t *testing.T) {
	root := t.TempDir()
	s, alice, _, err := studio.Prepare(root)
	if err != nil {
		t.Fatal(err)
	}
	key := alice.PublicKey
	s.Close()
	s, alice, _, err = studio.Prepare(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	if alice.PublicKey != key {
		t.Fatalf("age identity rotated: %q vs %q", alice.PublicKey, key)
	}
}

func TestSharedProjectCreatesDistinctWorktrees(t *testing.T) {
	s, _, _, err := studio.Prepare(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	aliceDir, bobDir, err := s.SharedProject("myapp")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(aliceDir) != "myapp" || filepath.Base(bobDir) != "myapp" {
		t.Fatalf("alice=%q bob=%q", aliceDir, bobDir)
	}
	if !strings.HasSuffix(filepath.Dir(aliceDir), "alice-home") || !strings.HasSuffix(filepath.Dir(bobDir), "bob-home") {
		t.Fatalf("clones not in homes alice=%q bob=%q", aliceDir, bobDir)
	}
	if aliceDir == bobDir {
		t.Fatal("shared Project used one folder for both Users")
	}
	aliceRemote := strings.TrimSpace(runGit(t, aliceDir, "remote", "get-url", "origin"))
	bobRemote := strings.TrimSpace(runGit(t, bobDir, "remote", "get-url", "origin"))
	if aliceRemote == "" || aliceRemote != bobRemote {
		t.Fatalf("remotes alice=%q bob=%q", aliceRemote, bobRemote)
	}
	aliceEmail := strings.TrimSpace(runGit(t, aliceDir, "config", "--get", "user.email"))
	bobEmail := strings.TrimSpace(runGit(t, bobDir, "config", "--get", "user.email"))
	if aliceEmail == bobEmail {
		t.Fatalf("same Git identity on both worktrees: %q", aliceEmail)
	}
}

func hostGit(t *testing.T, key string) string {
	t.Helper()
	out, err := exec.Command("git", "config", "--global", "--get", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s", args, out)
	}
	return string(out)
}

func gitWithHome(t *testing.T, home, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HOME="+home, "GIT_CONFIG_GLOBAL="+filepath.Join(home, ".gitconfig"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s", args, out)
	}
	return string(out)
}

func mustUnsetenv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
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
