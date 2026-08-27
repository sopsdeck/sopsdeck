package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSetWritesValueRetrievableByGet(t *testing.T) {
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
	code := Main([]string{"set", "NEW", "secret", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "NEW", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get NEW exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "secret\n" {
		t.Fatalf("get NEW stdout=%q", got)
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get HELLO exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "world\n" {
		t.Fatalf("get HELLO stdout=%q", got)
	}
}
