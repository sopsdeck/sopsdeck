package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDelRemovesKey(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)

	dir := t.TempDir()
	envFile := filepath.Join(dir, "hello.env")
	src, err := os.ReadFile(testdata(t, "hello.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envFile, src, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Main([]string{"del", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("del exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatalf("get HELLO succeeded after del, stdout=%q", stdout.String())
	}
}

func TestDelRemovesKeyFromUnlockedFile(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)

	root := t.TempDir()
	envFile := filepath.Join(root, ".env.production")
	if err := os.WriteFile(envFile, []byte("HELLO=world\nKEEP=yes\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := cmdProject([]string{"init", root, "--file", ".env.production"}, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("init exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"unlock", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("unlock exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"del", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("del exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv); code == 0 {
		t.Fatalf("get HELLO succeeded after del, stdout=%q", stdout.String())
	}
}
