package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupReferenceProject(t *testing.T) string {
	t.Helper()
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	envFile := filepath.Join(root, ".env.production")
	src, err := os.ReadFile(testdata(t, "hello.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envFile, src, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.js"), []byte("const v = process.env.HELLO;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte("hello: ${HELLO}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("DATABASE_URL is unused.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestReferencesCommandReportsCountsAndFiles(t *testing.T) {
	root := setupReferenceProject(t)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"references", "-f", filepath.Join(root, ".env.production")}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	var report []keyReference
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	byKey := map[string]keyReference{}
	for _, r := range report {
		byKey[r.Key] = r
	}
	if byKey["HELLO"].Count != 2 {
		t.Fatalf("HELLO count=%d, want 2", byKey["HELLO"].Count)
	}
	if len(byKey["HELLO"].Files) != 2 {
		t.Fatalf("HELLO files=%v, want 2", byKey["HELLO"].Files)
	}
}

func TestUnusedCommandListsZeroReferenceKeys(t *testing.T) {
	root := setupReferenceProject(t)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"unused", "-f", filepath.Join(root, ".env.production")}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	var unused []string
	if err := json.Unmarshal(stdout.Bytes(), &unused); err != nil {
		t.Fatal(err)
	}
	if len(unused) != 0 {
		t.Fatalf("unused=%v, want empty (HELLO is referenced)", unused)
	}
}

func TestRenameCommandDryRunListsPlannedChanges(t *testing.T) {
	root := setupReferenceProject(t)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"rename", "HELLO", "GREETING", "-f", filepath.Join(root, ".env.production")}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if !contains(out, "GREETING") || !contains(out, "app.js") || !contains(out, "config.yaml") {
		t.Fatalf("dry-run output missing plans:\n%s", out)
	}
}

func TestRenameCommandYesRenamesKeyAndRewritesReferences(t *testing.T) {
	root := setupReferenceProject(t)
	envFile := filepath.Join(root, ".env.production")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"rename", "HELLO", "GREETING", "-f", envFile, "--yes"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"get", "GREETING", "-f", envFile}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("get GREETING exit %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "world\n" {
		t.Fatalf("get GREETING=%q, want world", got)
	}
	appBody, _ := os.ReadFile(filepath.Join(root, "app.js"))
	if !contains(string(appBody), "GREETING") || contains(string(appBody), "HELLO") {
		t.Fatalf("app.js not rewritten:\n%s", appBody)
	}
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
