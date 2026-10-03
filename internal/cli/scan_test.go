package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanBlocksStagedCloudKey(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@sopsdeck.example")
	runGit(t, dir, "config", "user.name", "Sopsdeck Test")
	leaked := filepath.Join(dir, "leaked.env")
	if err := os.WriteFile(leaked, []byte("AWS_KEY=AKIAIOSFODNN7EXAMPLE\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "leaked.env")
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scan"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 1 {
		t.Fatalf("exit %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got := stderr.String() + stdout.String()
	if !strings.Contains(got, "leaked.env") {
		t.Fatalf("missing path: %q", got)
	}
	if strings.Contains(got, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("secret value leaked: %q", got)
	}
}

func TestScanIgnoresSopsCiphertext(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@sopsdeck.example")
	runGit(t, dir, "config", "user.name", "Sopsdeck Test")
	env := filepath.Join(dir, "hello.env")
	body := []byte("HELLO=ENC[AES256_GCM,data:AKIAIOSFODNN7EXAMPLE,iv:x,tag:y,type:str]\nsops_version=3.11.0\n")
	if err := os.WriteFile(env, body, 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "hello.env")
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scan"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestScanBlocksStagedPrivateKeyPEM(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@sopsdeck.example")
	runGit(t, dir, "config", "user.name", "Sopsdeck Test")
	key := filepath.Join(dir, "id_rsa")
	if err := os.WriteFile(key, []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA\n-----END RSA PRIVATE KEY-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "id_rsa")
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scan"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 1 {
		t.Fatalf("exit %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got := stderr.String()
	if !strings.Contains(got, "id_rsa") {
		t.Fatalf("stderr=%q", got)
	}
	if strings.Contains(got, "MIIEowIBAAKCAQEA") {
		t.Fatalf("secret value leaked: %q", got)
	}
}

func TestScanBlocksStagedAgeIdentityWithoutPrintingIt(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	identity := filepath.Join(dir, "age-identity.txt")
	secret := "AGE-SECRET-KEY-1FAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKE"
	if err := os.WriteFile(identity, []byte(secret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "age-identity.txt")
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scan"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 1 || !strings.Contains(stderr.String(), "Age identity") || strings.Contains(stderr.String(), secret) {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestScanBlocksStagedPlaintextManagedSecretAndAllowsPublicFields(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	manifest := "[[managed_file]]\npath = \"config.json\"\nformat = \"json\"\nencrypted_keys = [\"token\"]\npublic_keys = [\"metadata\"]\n"
	if err := os.WriteFile(filepath.Join(dir, ".sopsdeck.toml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "config.json")
	secret := "fake_managed_value"
	if err := os.WriteFile(file, []byte("{\"token\":\""+secret+"\",\"metadata\":{\"version\":\"1\"}}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".sopsdeck.toml", "config.json")
	if err := os.WriteFile(file, []byte("{\"token\":\"safe_worktree_value\",\"metadata\":{\"version\":\"1\"}}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scan"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 1 || !strings.Contains(stderr.String(), "plaintext managed secret token") || strings.Contains(stderr.String(), secret) {
		t.Fatalf("staged secret was not safely reported: exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	if err := os.WriteFile(file, []byte("{\"token\":\"\",\"metadata\":{\"version\":\"1\"},\"label\":\"safe\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "config.json")
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"scan"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("empty secret and public fields should pass: exit=%d stderr=%q", code, stderr.String())
	}
}

func TestScanBlocksStagedCommonToken(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@sopsdeck.example")
	runGit(t, dir, "config", "user.name", "Sopsdeck Test")
	leaked := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(leaked, []byte("token=ghp_abcdefghijklmnopqrstuvwxyz0123456789\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "notes.md")
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scan"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 1 {
		t.Fatalf("exit %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got := stderr.String()
	if !strings.Contains(got, "notes.md") {
		t.Fatalf("stderr=%q", got)
	}
	if strings.Contains(got, "ghp_abcdefghijklmnopqrstuvwxyz0123456789") {
		t.Fatalf("secret value leaked: %q", got)
	}
}

func TestScanWarnsLowerConfidenceToken(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@sopsdeck.example")
	runGit(t, dir, "config", "user.name", "Sopsdeck Test")
	notes := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(notes, []byte("stripe=sk_test_demo_value_here\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "notes.md")
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scan"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got := stderr.String() + stdout.String()
	if !strings.Contains(got, "warn") || !strings.Contains(got, "notes.md") {
		t.Fatalf("want warn: %q", got)
	}
	if strings.Contains(got, "sk_test_demo_value_here") {
		t.Fatalf("secret value leaked: %q", got)
	}
}

func TestScanAllowlistSkipsKnownFixture(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@sopsdeck.example")
	runGit(t, dir, "config", "user.name", "Sopsdeck Test")
	if err := os.Mkdir(filepath.Join(dir, "fixtures"), 0o700); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(dir, "fixtures", "id_rsa")
	if err := os.WriteFile(key, []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA\n-----END RSA PRIVATE KEY-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("[scan]\nallowlist = [\"fixtures/id_rsa\"]\n")
	if err := os.WriteFile(filepath.Join(dir, ".sopsdeck.toml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "fixtures/id_rsa")
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scan"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestScanAllowlistResolvesNestedProjectPathsWithSpaces(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	project := filepath.Join(dir, "app with spaces")
	if err := os.MkdirAll(filepath.Join(project, "fixtures"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".sopsdeck.toml"), []byte("[scan]\nallowlist = [\"fixtures/id_rsa\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(project, "fixtures", "id_rsa")
	if err := os.WriteFile(fixture, []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA\n-----END RSA PRIVATE KEY-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "app with spaces/.sopsdeck.toml", "app with spaces/fixtures/id_rsa")
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"scan"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("nested allowlist was ignored: exit=%d stderr=%q", code, stderr.String())
	}
}

func TestScanInstallWritesHookAndManifest(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	runGit(t, dir, "config", "user.email", "test@sopsdeck.example")
	runGit(t, dir, "config", "user.name", "Sopsdeck Test")
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scan", "--install"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	hookDir, err := gitHooksPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	hook, err := os.ReadFile(filepath.Join(hookDir, "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hook), "sopsdeck scan") {
		t.Fatalf("hook=%s", hook)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".sopsdeck.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "hook") {
		t.Fatalf("manifest=%s", raw)
	}
}

func TestScanUninstallRemovesOnlyTheOwnedHook(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"scan", "--install"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("install: exit=%d stderr=%q", code, stderr.String())
	}
	hookDir, err := gitHooksPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(hookDir, "pre-commit")
	if err := os.WriteFile(hook, []byte(scanHook+"# user change\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"scan", "--uninstall"}, os.Stdin, &stdout, &stderr, os.Getenv); code == 0 {
		t.Fatalf("modified hook should be preserved: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if _, err := os.Stat(hook); err != nil {
		t.Fatalf("modified hook was removed: %v", err)
	}
	if err := os.WriteFile(hook, []byte(scanHook), 0o755); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"scan", "--uninstall"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("uninstall: exit=%d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(hook); !os.IsNotExist(err) {
		t.Fatalf("owned hook remains: %v", err)
	}
	manifest, err := loadManifest(filepath.Join(dir, ".sopsdeck.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Scan.Hook {
		t.Fatal("manifest still records an installed hook")
	}
}

func TestScanInstallPreservesExistingHook(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	existing := []byte("#!/bin/sh\necho existing hook\n")
	if err := os.WriteFile(hookPath, existing, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"scan", "--install"}, os.Stdin, &stdout, &stderr, os.Getenv); code == 0 {
		t.Fatalf("existing hook should require manual integration: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, existing) {
		t.Fatalf("existing hook changed: %q", got)
	}
}

func TestScanInstallUsesLinkedWorktreeHooksPath(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	runGit(t, repo, "config", "user.email", "test@sopsdeck.example")
	runGit(t, repo, "config", "user.name", "Sopsdeck Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGitCommit(t, repo, "seed")
	worktree := filepath.Join(t.TempDir(), "linked project with spaces")
	runGit(t, repo, "worktree", "add", "-b", "fixture", worktree, "HEAD")
	t.Chdir(worktree)
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"scan", "--install"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("install in linked worktree: exit=%d stderr=%q", code, stderr.String())
	}
	hookDir, err := gitHooksPath(worktree)
	if err != nil {
		t.Fatal(err)
	}
	hook, err := os.ReadFile(filepath.Join(hookDir, "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if string(hook) != scanHook {
		t.Fatalf("hook=%q", hook)
	}
}

func TestScanInstallHonorsLocalHooksPathWithSpaces(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	runGit(t, dir, "config", "core.hooksPath", "hooks with spaces")
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"scan", "--install"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("install with local hooks path: exit=%d stderr=%q", code, stderr.String())
	}
	hook, err := os.ReadFile(filepath.Join(dir, "hooks with spaces", "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if string(hook) != scanHook {
		t.Fatalf("hook=%q", hook)
	}
}

func TestScanInstallRefusesGlobalHooksPath(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	globalHooks := filepath.Join(t.TempDir(), "global hooks")
	runGit(t, dir, "config", "--global", "core.hooksPath", globalHooks)
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"scan", "--install"}, os.Stdin, &stdout, &stderr, os.Getenv); code == 0 {
		t.Fatalf("global hooks path should be preserved: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(globalHooks, "pre-commit")); !os.IsNotExist(err) {
		t.Fatalf("global hook path was modified: %v", err)
	}
}
