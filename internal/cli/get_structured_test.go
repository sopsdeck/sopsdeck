package cli

import (
	"bytes"
	"os"
	"strings"
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

func TestGetPrintsValueFromEASJSONAndWarnsOnStderr(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))

	var stdout, stderr bytes.Buffer
	code := Main([]string{"get", "EXPO_PUBLIC_API_URL", "-f", testdata(t, "eas.json")}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "https://api.acme.test\n" {
		t.Fatalf("stdout=%q", got)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "EAS CLI will not read SOPS ciphertext") {
		t.Fatalf("stderr=%q, want EAS CLI ciphertext warning", errOut)
	}
	if strings.Contains(errOut, "usage:") || strings.Contains(errOut, "(1)") {
		t.Fatalf("stderr dumped extra help: %q", errOut)
	}
}

func TestGetPrintsValueFromComposeYAML(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))

	var stdout, stderr bytes.Buffer
	code := Main([]string{"get", "POSTGRES_PASSWORD", "-f", testdata(t, "compose.yaml")}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "acme_pg_demo_password\n" {
		t.Fatalf("stdout=%q", got)
	}
}
