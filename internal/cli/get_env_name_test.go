package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGetReadsDotenvNamedEnvProduction(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	dir := t.TempDir()
	dst := filepath.Join(dir, ".env.production")
	src, err := os.ReadFile(testdata(t, "hello.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, src, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Main([]string{"get", "HELLO", "-f", dst}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "world\n" {
		t.Fatalf("stdout=%q", got)
	}
}
