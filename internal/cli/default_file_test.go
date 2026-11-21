package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDotenvFile(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(root, ".sopsdeck.toml"), "[[managed_file]]\npath = '.env.production'\nformat = 'dotenv'\n")
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	if got := defaultDotenvFile(); got != filepath.Join(root, ".env.production") {
		t.Fatalf("default = %q", got)
	}
	mustWriteFile(t, filepath.Join(root, ".sopsdeck.toml"), "[[managed_file]]\npath = '.env.production'\nformat = 'dotenv'\n[[managed_file]]\npath = '.env.local'\nformat = 'dotenv'\n")
	if got := defaultDotenvFile(); got != "" {
		t.Fatalf("ambiguous default = %q", got)
	}
	mustWriteFile(t, filepath.Join(root, ".sopsdeck.toml"), "[[managed_file]]\npath = '.env.production'\nformat = 'dotenv'\n[[managed_file]]\npath = '.env'\nformat = 'dotenv'\n")
	if got := defaultDotenvFile(); got != filepath.Join(root, ".env") {
		t.Fatalf("default = %q", got)
	}
}
