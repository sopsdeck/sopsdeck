package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pasteTestEnv(t *testing.T) string {
	t.Helper()
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	dir := t.TempDir()
	envFile := filepath.Join(dir, "hello.env")
	src, err := os.ReadFile(testdata(t, "hello.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envFile, src, 0o600); err != nil {
		t.Fatal(err)
	}
	return envFile
}

func TestSetStdinDotenvPreviewsWithoutWriting(t *testing.T) {
	envFile := pasteTestEnv(t)

	var stdout, stderr bytes.Buffer
	code := Main(
		[]string{"set", "-f", envFile},
		bytes.NewBufferString("NEW=pasted\n"),
		&stdout, &stderr, os.Getenv,
	)
	if code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "preview") {
		t.Fatalf("stdout=%q, want preview", out)
	}
	if !strings.Contains(out, "NEW") {
		t.Fatalf("stdout=%q, want key NEW", out)
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "NEW", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatalf("get NEW should fail before apply, stdout=%q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get HELLO exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "world\n" {
		t.Fatalf("get HELLO stdout=%q", got)
	}
}

func TestSetStdinDotenvAppliesWithYes(t *testing.T) {
	envFile := pasteTestEnv(t)

	var stdout, stderr bytes.Buffer
	code := Main(
		[]string{"set", "-f", envFile, "--yes"},
		bytes.NewBufferString("NEW=pasted\n"),
		&stdout, &stderr, os.Getenv,
	)
	if code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "NEW", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get NEW exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "pasted\n" {
		t.Fatalf("get NEW stdout=%q", got)
	}
}

func TestSetStdinJSONBulkAppliesWithYes(t *testing.T) {
	envFile := pasteTestEnv(t)

	var stdout, stderr bytes.Buffer
	code := Main(
		[]string{"set", "-f", envFile, "--yes"},
		bytes.NewBufferString(`{"NEW":"fromjson"}`),
		&stdout, &stderr, os.Getenv,
	)
	if code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "NEW", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get NEW exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "fromjson\n" {
		t.Fatalf("get NEW stdout=%q", got)
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "HELLO", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get HELLO exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "world\n" {
		t.Fatalf("get HELLO stdout=%q", got)
	}
}

func TestSetStdinYAMLBulkAppliesWithYes(t *testing.T) {
	envFile := pasteTestEnv(t)

	var stdout, stderr bytes.Buffer
	code := Main(
		[]string{"set", "-f", envFile, "--yes"},
		bytes.NewBufferString("NEW: fromyaml\n"),
		&stdout, &stderr, os.Getenv,
	)
	if code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "NEW", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get NEW exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "fromyaml\n" {
		t.Fatalf("get NEW stdout=%q", got)
	}
}

func TestSetStdinLoneValueUsesKey(t *testing.T) {
	envFile := pasteTestEnv(t)

	var stdout, stderr bytes.Buffer
	code := Main(
		[]string{"set", "NEW", "-f", envFile, "--yes"},
		bytes.NewBufferString("lonely-secret"),
		&stdout, &stderr, os.Getenv,
	)
	if code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "NEW", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get NEW exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "lonely-secret\n" {
		t.Fatalf("get NEW stdout=%q", got)
	}
}

func TestSetStdinDoesNotLeakValuesInPreview(t *testing.T) {
	envFile := pasteTestEnv(t)

	var stdout, stderr bytes.Buffer
	code := Main(
		[]string{"set", "-f", envFile},
		bytes.NewBufferString("NEW=supersecretvalue\n"),
		&stdout, &stderr, os.Getenv,
	)
	if code != 0 {
		t.Fatalf("set exit %d stderr=%q", code, stderr.String())
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "preview") {
		t.Fatalf("output=%q, want preview", combined)
	}
	if !strings.Contains(combined, "NEW") {
		t.Fatalf("output=%q, want key NEW", combined)
	}
	if strings.Contains(combined, "supersecretvalue") {
		t.Fatalf("output leaked secret value: %q", combined)
	}
}
