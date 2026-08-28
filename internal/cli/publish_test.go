package cli

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sopsdeck/internal/studio"
)

func TestPublishDryRunDoesNotWrite(t *testing.T) {
	st, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	alice, err := st.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(alice.Home, ".env.production")
	if err := aliceCLI(alice, "set", "HELLO", "world", "-f", env); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	var code int
	alice.WithWorld(func() {
		code = Main([]string{"publish", "-f", env, "--prefix", "SD_"}, os.Stdin, &stdout, &stderr, alice.Getenv)
	})
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "dry-run") || !strings.Contains(stdout.String(), "SD_HELLO") {
		t.Fatalf("stdout=%q", stdout.String())
	}
	if len(st.GitHub.Names()) != 0 {
		t.Fatalf("names=%v", st.GitHub.Names())
	}
}

func TestPublishRequiresAPI(t *testing.T) {
	var stderr bytes.Buffer
	getenv := func(string) string { return "" }
	if code := Main([]string{"publish", "-f", "missing.env"}, os.Stdin, &bytes.Buffer{}, &stderr, getenv); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "SOPSDECK_GITHUB_API") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestPublishUsesManifestPrefixAndRepo(t *testing.T) {
	st, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	alice, err := st.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(alice.Home, ".env.production")
	if err := aliceCLI(alice, "set", "HELLO", "world", "-f", env); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("[[managed_file]]\npath = \".env.production\"\nrepo = \"acme/app\"\nprefix = \"SD_\"\n")
	if err := os.WriteFile(filepath.Join(alice.Home, ".sopsdeck.toml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	var code int
	alice.WithWorld(func() {
		code = Main([]string{"publish", "-f", env}, os.Stdin, &stdout, &stderr, alice.Getenv)
	})
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "dry-run") || !strings.Contains(out, "SD_HELLO") || !strings.Contains(out, "acme/app") {
		t.Fatalf("stdout=%q", out)
	}
}

func TestPublishManifestKeysSelectsSubset(t *testing.T) {
	st, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	alice, err := st.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(alice.Home, ".env.production")
	if err := aliceCLI(alice, "set", "HELLO", "world", "-f", env); err != nil {
		t.Fatal(err)
	}
	if err := aliceCLI(alice, "set", "OTHER", "skip", "-f", env); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("[[managed_file]]\npath = \".env.production\"\nprefix = \"SD_\"\nkeys = [\"HELLO\"]\n")
	if err := os.WriteFile(filepath.Join(alice.Home, ".sopsdeck.toml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var code int
	alice.WithWorld(func() {
		code = Main([]string{"publish", "-f", env}, os.Stdin, &stdout, io.Discard, alice.Getenv)
	})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "SD_HELLO") {
		t.Fatalf("stdout=%q", out)
	}
	if strings.Contains(out, "OTHER") {
		t.Fatalf("published unselected key: %q", out)
	}
}

func TestPublishRecordsLastPublishedNames(t *testing.T) {
	st, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	alice, err := st.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(alice.Home, ".env.production")
	if err := aliceCLI(alice, "set", "HELLO", "world", "-f", env); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(alice.Home, ".sopsdeck.toml")
	manifest := []byte("[[managed_file]]\npath = \".env.production\"\nprefix = \"SD_\"\n")
	if err := os.WriteFile(manifestPath, manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	var code int
	alice.WithWorld(func() {
		code = Main([]string{"publish", "-f", env, "--yes"}, os.Stdin, io.Discard, io.Discard, alice.Getenv)
	})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "SD_HELLO") {
		t.Fatalf("manifest missing published names:\n%s", raw)
	}
}

func TestPublishPruneDeletesOnlyPreviouslyPublishedNames(t *testing.T) {
	st, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	alice, err := st.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(alice.Home, ".env.production")
	if err := aliceCLI(alice, "set", "HELLO", "world", "-f", env); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("[[managed_file]]\npath = \".env.production\"\nprefix = \"SD_\"\npublished = [\"SD_OLD\"]\n")
	if err := os.WriteFile(filepath.Join(alice.Home, ".sopsdeck.toml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"SD_OLD", "SD_EXTRA"} {
		putURL := st.GitHub.URL() + "/repos/studio/demo/actions/secrets/" + name
		req, err := http.NewRequest(http.MethodPut, putURL, strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}
	if err := aliceCLI(alice, "publish", "-f", env, "--yes", "--prune"); err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, name := range st.GitHub.Names() {
		have[name] = true
	}
	if have["SD_OLD"] {
		t.Fatalf("previously published name still present: %v", st.GitHub.Names())
	}
	if !have["SD_EXTRA"] {
		t.Fatalf("unrecorded prefixed name was deleted: %v", st.GitHub.Names())
	}
	if !have["SD_HELLO"] {
		t.Fatalf("current name missing: %v", st.GitHub.Names())
	}
}

func TestPublishPruneLeavesUnrecordedPrefixedNames(t *testing.T) {
	st, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	alice, err := st.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(alice.Home, ".env.production")
	if err := aliceCLI(alice, "set", "HELLO", "world", "-f", env); err != nil {
		t.Fatal(err)
	}
	putURL := st.GitHub.URL() + "/repos/studio/demo/actions/secrets/SD_OLD"
	req, err := http.NewRequest(http.MethodPut, putURL, strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if err := aliceCLI(alice, "publish", "-f", env, "--prefix", "SD_", "--yes", "--prune"); err != nil {
		t.Fatal(err)
	}
	have := false
	for _, name := range st.GitHub.Names() {
		if name == "SD_OLD" {
			have = true
		}
	}
	if !have {
		t.Fatalf("unrecorded prefixed name was deleted: %v", st.GitHub.Names())
	}
}

func TestPublishManifestEnvironmentPutsEnvironmentSecrets(t *testing.T) {
	st, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	alice, err := st.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(alice.Home, ".env.production")
	if err := aliceCLI(alice, "set", "HELLO", "world", "-f", env); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("[[managed_file]]\npath = \".env.production\"\nrepo = \"acme/app\"\nenvironment = \"production\"\nprefix = \"SD_\"\n")
	if err := os.WriteFile(filepath.Join(alice.Home, ".sopsdeck.toml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := aliceCLI(alice, "publish", "-f", env, "--yes"); err != nil {
		t.Fatal(err)
	}
	if len(st.GitHub.Names()) != 0 {
		t.Fatalf("wrote repository secrets: %v", st.GitHub.Names())
	}
	listURL := st.GitHub.URL() + "/repos/acme/app/environments/production/secrets"
	resp, err := http.Get(listURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "SD_HELLO") {
		t.Fatalf("environment secrets status=%d body=%s", resp.StatusCode, body)
	}
}
