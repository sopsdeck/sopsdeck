package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRedactsSecretValuesOnChildStdout(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)
	mustUnsetenv(t, "HELLO")

	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	secret := "sk_live_abcdef123456"
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"set", "HELLO", secret, "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code := Main([]string{"run", "-f", file, "--", "sh", "-c", "echo $HELLO"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("run exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "[sopsdeck:HELLO]") {
		t.Fatalf("stdout=%q want redacted value", got)
	}
	if strings.Contains(stdout.String(), secret) {
		t.Fatalf("secret value leaked to stdout: %q", stdout.String())
	}
}

func TestRunRedactsShortSecretOnStdoutAndStderr(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)
	mustUnsetenv(t, "PUBLIC_TOKEN")

	file := filepath.Join(t.TempDir(), ".env")
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"set", "PUBLIC_TOKEN", "abc", "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code := Main([]string{"run", "-f", file, "--", "sh", "-c", `printf '%s' "$PUBLIC_TOKEN"; printf '%s' "$PUBLIC_TOKEN" >&2`}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("run exit %d stderr=%q", code, stderr.String())
	}
	for name, output := range map[string]string{"stdout": stdout.String(), "stderr": stderr.String()} {
		if output != "[sopsdeck:PUBLIC_TOKEN]" {
			t.Errorf("%s=%q want short value redacted", name, output)
		}
	}
}

func TestRunNoRedactPassesSecretThrough(t *testing.T) {
	age := testdata(t, "age.txt")
	t.Setenv("SOPS_AGE_KEY_FILE", age)
	mustUnsetenv(t, "HELLO")

	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	secret := "sk_live_abcdef123456"
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"set", "HELLO", secret, "-f", file}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code := Main([]string{"run", "--no-redact", "-f", file, "--", "sh", "-c", "echo $HELLO"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("run exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != secret+"\n" {
		t.Fatalf("--no-redact must pass the value through, stdout=%q", got)
	}
}
