package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestGetWithoutKeyPrintsAllDotenvPairs(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))

	var stdout, stderr bytes.Buffer
	code := Main([]string{"get", "-f", testdata(t, "hello.env")}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "HELLO=world\n" {
		t.Fatalf("stdout=%q want %q", got, "HELLO=world\n")
	}
}
