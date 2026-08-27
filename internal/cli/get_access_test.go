package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetWithoutAccessExplainsMissingAccess(t *testing.T) {
	state := t.TempDir()
	t.Setenv("SOPSDECK_STATE_DIR", state)
	mustUnsetenv(t, "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY")

	var stdout, stderr bytes.Buffer
	if code := Main([]string{"identity", "create", "--confirmed-backup"}, os.Stdin, &stdout, &stderr, os.Getenv); code != 0 {
		t.Fatalf("identity exit %d stderr=%q", code, stderr.String())
	}
	t.Setenv("SOPS_AGE_KEY_FILE", filepath.Join(state, "age.txt"))

	stdout.Reset()
	stderr.Reset()
	code := Main([]string{"get", "HELLO", "-f", testdata(t, "hello.env")}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatal("expected non-zero exit without Access")
	}
	got := stderr.String()
	if strings.Contains(got, "no identity matched") ||
		strings.Contains(got, "Failed to get the data key") ||
		strings.Contains(got, "Error getting data key") {
		t.Fatalf("raw sops leaked: %q", got)
	}
	if !strings.Contains(got, "no Access") {
		t.Fatalf("stderr=%q, want no Access", got)
	}
}
