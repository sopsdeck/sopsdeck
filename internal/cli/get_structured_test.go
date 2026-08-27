package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestGetPrintsValueFromSOPSJSONAndYAML(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)

	for _, file := range []string{"hello.json", "hello.yaml"} {
		t.Run(file, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Main([]string{"get", "HELLO", "-f", testdata(t, file)}, os.Stdin, &stdout, &stderr, os.Getenv)
			if code != 0 {
				t.Fatalf("exit %d stderr=%q", code, stderr.String())
			}
			if got := stdout.String(); got != "world\n" {
				t.Fatalf("stdout=%q", got)
			}
		})
	}
}
