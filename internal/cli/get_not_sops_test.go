package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetOnPlainDotenvExplainsNotSOPS(t *testing.T) {
	root := t.TempDir()
	plain := filepath.Join(root, ".env")
	if err := os.WriteFile(plain, []byte("STRIPE_KEY=sk_live_xyz\nMIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQgWhtiNB6\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"get", "-f", plain, "--output", "json"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatal("expected non-zero exit for a plain dotenv")
	}
	got := stderr.String()
	if !strings.Contains(got, "not a SOPS-encrypted file") {
		t.Fatalf("stderr=%q, want a not-SOPS message", got)
	}
	if strings.Contains(got, "MIGTAgEAMBMGByqGSM49") {
		t.Fatalf("raw dotenv line leaked into error: %q", got)
	}
}
