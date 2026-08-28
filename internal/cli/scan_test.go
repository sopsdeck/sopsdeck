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

func TestScanInstallWritesHookAndManifest(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@sopsdeck.example")
	runGit(t, dir, "config", "user.name", "Sopsdeck Test")
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scan", "--install"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	hook, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", "pre-commit"))
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
