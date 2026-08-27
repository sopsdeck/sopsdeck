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
	got := stdout.String()
	if !bytes.Contains([]byte(got), []byte("HELLO=world\n")) {
		t.Fatalf("stdout=%q, want HELLO=world among dumped pairs", got)
	}
}
