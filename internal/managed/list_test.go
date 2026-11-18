package managed

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverySeparatesNestedProjectsAndRepos(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"apps/web", "other-project", "other-repo", "worktree"} {
		folder := filepath.Join(root, dir)
		if err := os.MkdirAll(folder, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, ".env"), []byte("HELLO=world\nsops_mac=ENC[test]\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for dir, marker := range map[string]string{"other-project": ".sopsdeck.toml", "other-repo": ".git", "worktree": ".git"} {
		if err := os.WriteFile(filepath.Join(root, dir, marker), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(root, "apps/web/.env"), filepath.Join(root, "alias.env")); err != nil {
		t.Fatal(err)
	}
	for _, scan := range []func(string) ([]File, error){List, Candidates} {
		files, err := scan(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 1 || filepath.ToSlash(files[0].Rel) != "apps/web/.env" {
			t.Fatalf("nested boundaries: %+v", files)
		}
	}
}

func TestListFindsDotenvAndSOPSStructuredFiles(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(".env.production", "HELLO=world\nsops_mac=ENC[AES256_GCM,data:.,tag:.=,type:str]\n")
	write("plain.env", "HELLO=world\n")
	write("plain.json", `{"HELLO":"world"}`+"\n")
	write("secrets.json", "{\n  \"HELLO\": \"world\",\n  \"sops\": {}\n}\n")
	write("nested/app.yaml", "sops:\n  kms: []\n")
	write("node_modules/skip.env", "NO=pe\n")

	files, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(files))
	for i, f := range files {
		got[i] = f.Rel
	}
	want := []string{".env.production", filepath.Join("nested", "app.yaml"), "plain.env", "plain.json", "secrets.json"}
	if len(got) != len(want) {
		t.Fatalf("files=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("files=%v want %v", got, want)
		}
	}
}

func TestListIncludesPlainDotenv(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("HELLO=world\nSTRIPE_KEY=sk_live_xyz\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Rel != ".env" {
		t.Fatalf("plain .env should be a Managed File: %v", files)
	}
}

func TestListIncludesPlainStructuredFileContainingSOPSWord(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.json")
	if err := os.WriteFile(path, []byte(`{"sops":"disabled","value":"plain"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Rel != "config.json" {
		t.Fatalf("plain structured files should be listed: %v", files)
	}
}

func TestCandidatesIncludeLockfiles(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("package-lock.json", `{"packages":{"":{"name":"app"}}}`+"\n")
	write("pnpm-lock.yaml", "packages: {}\n")
	write("yarn.lock", "# yarn lockfile\n")
	write("secrets.json", `{"HELLO":"world"}`+"\n")
	files, err := Candidates(root)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(files))
	for i, f := range files {
		got[i] = f.Name
	}
	want := []string{"package-lock.json", "pnpm-lock.yaml", "secrets.json", "yarn.lock"}
	if len(got) != len(want) {
		t.Fatalf("candidates=%v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidates=%v, want %v", got, want)
		}
	}
}

func TestListSkipsGeneratedBuildDirs(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{".next", "build", "coverage", "__pycache__", ".turbo"} {
		path := filepath.Join(root, dir, ".env")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		body := "HELLO=world\nsops_mac=ENC[AES256_GCM,data:.,tag:.=,type:str]\n"
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	files, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("generated dirs should be skipped: %v", files)
	}
}

func TestListFindsCommittedComposeYAMLAndMultilineDotenv(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	files, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range files {
		got[f.Name] = true
	}
	for _, name := range []string{"compose.yaml", "hello.multiline.env", "config.json"} {
		if !got[name] {
			t.Fatalf("names=%v, want %s", keys(got), name)
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestCandidatesIncludeEASJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "eas.json"), []byte(`{"build":{"env":{"SECRET":"value"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := Candidates(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Rel != "eas.json" {
		t.Fatalf("eas.json should be a Managed File candidate: %v", files)
	}
}
