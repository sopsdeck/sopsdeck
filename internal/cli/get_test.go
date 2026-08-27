package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGetPrintsValueFromSOPSDotenv(t *testing.T) {
	age := testdata(t, "age.txt")
	envFile := testdata(t, "hello.env")
	t.Setenv("SOPS_AGE_KEY_FILE", age)

	var stdout, stderr bytes.Buffer
	code := Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "world\n" {
		t.Fatalf("stdout=%q want %q", got, "world\n")
	}
}

func testdata(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join("..", "..", "testdata", name)
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	return p
}
