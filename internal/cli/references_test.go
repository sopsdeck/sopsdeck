package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCountReferencesMatchesWordDollarAndBraces(t *testing.T) {
	body := []byte("STRIPE_SECRET=$STRIPE_SECRET\nAPI=${STRIPE_SECRET}\n# STRIPE_SECRET here\nSTRIPE_SECRET_KEY=no\nMYSTRIPE_SECRET=no\n")
	got := countReferences("STRIPE_SECRET", body)
	if got != 4 {
		t.Fatalf("count=%d, want 4", got)
	}
}

func TestCountReferencesIgnoresSubstrings(t *testing.T) {
	if c := countReferences("KEY", []byte("MYKEY KEY2 KEY ENKEY")); c != 1 {
		t.Fatalf("count=%d, want 1", c)
	}
}

func TestScanProjectReferencesCountsAndExcludesManagedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env.production"), []byte("STRIPE_SECRET=ENC[...]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.js"), []byte("const key = process.env.STRIPE_SECRET;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "defaults.yaml"), []byte("stripe: ${STRIPE_SECRET}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("Use STRIPE_SECRET and DATABASE_URL.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	managed := map[string]bool{".env.production": true}
	counts, files, err := scanProjectReferences(root, []string{"STRIPE_SECRET", "DATABASE_URL"}, managed)
	if err != nil {
		t.Fatal(err)
	}
	if counts["STRIPE_SECRET"] != 3 {
		t.Fatalf("STRIPE_SECRET=%d, want 3", counts["STRIPE_SECRET"])
	}
	if counts["DATABASE_URL"] != 1 {
		t.Fatalf("DATABASE_URL=%d, want 1", counts["DATABASE_URL"])
	}
	if len(files["STRIPE_SECRET"]) != 3 {
		t.Fatalf("files=%v, want 3", files["STRIPE_SECRET"])
	}
}

func TestScanProjectReferencesSkipsGitAndBinary(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "refs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "refs", "STRIPE_SECRET"), []byte("STRIPE_SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "logo.png"), []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 'S', 'T', 'R', 'I', 'P', 'E', '_'}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.js"), []byte("STRIPE_SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	counts, _, err := scanProjectReferences(root, []string{"STRIPE_SECRET"}, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if counts["STRIPE_SECRET"] != 1 {
		t.Fatalf("count=%d, want 1 (git and binary excluded)", counts["STRIPE_SECRET"])
	}
}
