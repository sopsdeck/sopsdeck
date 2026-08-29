package managed

import (
	"os"
	"path/filepath"
	"testing"
)

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
	want := []string{".env.production", filepath.Join("nested", "app.yaml"), "secrets.json"}
	if len(got) != len(want) {
		t.Fatalf("files=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("files=%v want %v", got, want)
		}
	}
}

func TestListExcludesPlainDotenv(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("HELLO=world\nSTRIPE_KEY=sk_live_xyz\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("plain .env should not be a Managed File: %v", files)
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
	for _, name := range []string{"compose.yaml", "hello.multiline.env", "eas.json"} {
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
