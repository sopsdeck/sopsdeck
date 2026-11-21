package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnignoreManagedFile(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q", root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %s %v", out, err)
	}
	mustWriteFile(t, filepath.Join(root, ".gitignore"), ".env*\n")
	changed, err := unignoreManagedFile(root, ".env")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if !strings.Contains(mustReadFile(t, filepath.Join(root, ".gitignore")), "!/.env\n") {
		t.Fatal("missing exception")
	}
	changed, err = unignoreManagedFile(root, ".env")
	if err != nil || changed {
		t.Fatalf("second run: changed=%v err=%v", changed, err)
	}
	mustWriteFile(t, filepath.Join(root, ".gitignore"), "private/\n")
	if err := os.Mkdir(filepath.Join(root, "private"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := unignoreManagedFile(root, "private/.env"); err == nil {
		t.Fatal("expected ignored parent warning")
	}
}
