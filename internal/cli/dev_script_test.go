package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDevScriptBuildOnlyWritesCLI(t *testing.T) {
	root := filepath.Join("..", "..")
	cmd := exec.Command("bash", "./scripts/dev", "--build-only")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dev --build-only: %v %s", err, out)
	}
	bin := strings.TrimSpace(string(out))
	wantName := "sopsdeck"
	if runtime.GOOS == "windows" {
		wantName += ".exe"
	}
	if filepath.Base(bin) != wantName {
		t.Fatalf("stdout=%q, want path to %s", bin, wantName)
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
