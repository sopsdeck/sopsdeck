package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
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

func TestVersionMatchesDesktopManifests(t *testing.T) {
	root := repoRoot(t)
	want := appver.Version
	tauri := readJSONVersion(t, filepath.Join(root, "desktop/src-tauri/tauri.conf.json"))
	pkg := readJSONVersion(t, filepath.Join(root, "desktop/package.json"))
	cargo := readCargoVersion(t, filepath.Join(root, "desktop/src-tauri/Cargo.toml"))
	if tauri != want || pkg != want || cargo != want {
		t.Fatalf("version drift: go=%s tauri=%s package=%s cargo=%s", want, tauri, pkg, cargo)
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

func readCargoVersion(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`(?m)^version = "([^"]+)"`).FindSubmatch(raw)
	if match == nil {
		t.Fatalf("no version in %s", path)
	}
	return string(match[1])
}
