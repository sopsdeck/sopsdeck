package cli

import (
	"bytes"
	"os"
	"testing"
)

// TestRunDoesNotInjectEnvFromStructuredFile pins the format dispatch: JSON
// Managed Files are transiently unlocked on disk for the child, not injected
// as environment variables.
func TestRunDoesNotInjectEnvFromStructuredFile(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)
	mustUnsetenv(t, "HELLO")

	var stdout, stderr bytes.Buffer
	code := Main([]string{"run", "-f", testdata(t, "hello.json"), "--", "printenv", "HELLO"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatalf("HELLO must not leak into the environment from a structured file, stdout=%q", stdout.String())
	}
}
