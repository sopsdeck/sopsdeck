package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appver "sopsdeck/internal/version"
)

func TestVersionPrintsAppVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--version"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	got := strings.TrimSpace(stdout.String())
	if got != appver.Version {
		t.Fatalf("version %q want %q", got, appver.Version)
	}
}

func TestNoArgsPrintsUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main(nil, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr.String(), "usage: sopsdeck") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestHelpPrintsUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--help"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "usage: sopsdeck") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestHelpColorAlwaysRendersCommands(t *testing.T) {
	t.Setenv("SOPSDECK_COLOR", "always")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"-h"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Commands") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestUnknownCommandPrintsUsageHint(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"not-a-command"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr.String(), `unknown command "not-a-command"`) {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestConfigureIntegrationRequiresSevenArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"configure_integration", "file"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr.String(), "usage: sopsdeck configure_integration") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestVersionMatchesPackage(t *testing.T) {
	root := repoRoot(t)
	want := appver.Version
	npm := readJSONVersion(t, filepath.Join(root, "package.json"))
	if npm != want {
		t.Fatalf("version drift: go=%s npm=%s", want, npm)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("go.mod not found")
	return ""
}

func readJSONVersion(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Version
}
