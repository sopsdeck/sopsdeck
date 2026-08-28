package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevScriptBuildOnlyWritesCLI(t *testing.T) {
	root := filepath.Join("..", "..")
	cmd := exec.Command("./scripts/dev", "--build-only")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dev --build-only: %v %s", err, out)
	}
	bin := strings.TrimSpace(string(out))
	if filepath.Base(bin) != "sopsdeck" {
		t.Fatalf("stdout=%q, want path to sopsdeck", bin)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatal(err)
	}
	ver, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatalf("sopsdeck --version: %v", err)
	}
	if strings.TrimSpace(string(ver)) == "" {
		t.Fatal("empty --version")
	}
}
