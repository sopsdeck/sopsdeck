package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestGetOutputJSONPrintsPairsForAnyFormat(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))

	for _, file := range []string{"hello.env", "hello.json", "hello.yaml"} {
		t.Run(file, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Main([]string{"get", "-f", testdata(t, file), "--output", "json"}, os.Stdin, &stdout, &stderr, os.Getenv)
			if code != 0 {
				t.Fatalf("exit %d stderr=%q", code, stderr.String())
			}
			var got map[string]string
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatalf("stdout=%q: %v", stdout.String(), err)
			}
			if got["HELLO"] != "world" {
				t.Fatalf("pairs=%v", got)
			}
		})
	}
}
