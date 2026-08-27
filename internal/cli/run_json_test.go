package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestRunInjectsJSONSecrets(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)
	mustUnsetenv(t, "HELLO")

	var stdout, stderr bytes.Buffer
	code := Main([]string{"run", "-f", testdata(t, "hello.json"), "--", "printenv", "HELLO"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("run exit %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if got := stdout.String(); got != "world\n" {
		t.Fatalf("stdout=%q", got)
	}
}
