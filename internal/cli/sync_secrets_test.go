package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/nacl/box"

	"sopsdeck/internal/studio"
)

func TestGitHubSecretURLsForScopes(t *testing.T) {
	base := "https://api.github.test"
	tests := []struct {
		name   string
		target syncTarget
		secret string
		want   string
	}{
		{name: "repository", target: syncTarget{Scope: "repo", Repo: "acme/app"}, secret: "SD_TOKEN", want: base + "/repos/acme/app/actions/secrets/SD_TOKEN"},
		{name: "organization", target: syncTarget{Scope: "org", Org: "acme"}, secret: "SD_TOKEN", want: base + "/orgs/acme/actions/secrets/SD_TOKEN"},
		{name: "environment", target: syncTarget{Scope: "environment", Repo: "acme/app", Environment: "production"}, secret: "SD_TOKEN", want: base + "/repos/acme/app/environments/production/secrets/SD_TOKEN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := secretURL(base, tt.target, tt.secret)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("url=%q want %q", got, tt.want)
			}
		})
	}
}

func TestSyncSealsValuesWithGitHubPublicKey(t *testing.T) {
	publicKey, privateKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var decrypted, keyID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"key_id": "github-key",
				"key":    base64.StdEncoding.EncodeToString(publicKey[:]),
			})
			return
		}
		var body struct {
			EncryptedValue string `json:"encrypted_value"`
			KeyID          string `json:"key_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		sealed, err := base64.StdEncoding.DecodeString(body.EncryptedValue)
		if err != nil {
			t.Error(err)
			return
		}
		plaintext, ok := box.OpenAnonymous(nil, sealed, publicKey, privateKey)
		if !ok {
			t.Error("encrypted_value is not a valid sealed box")
			return
		}
		decrypted, keyID = string(plaintext), body.KeyID
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

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
	getenv := func(key string) string {
		if key == "SOPSDECK_GITHUB_API" {
			return server.URL
		}
		return alice.Getenv(key)
	}
	alice.WithWorld(func() {
		if code := Main([]string{"sync", "-f", env, "--prefix", "SD_"}, os.Stdin, io.Discard, io.Discard, getenv); code != 0 {
			t.Fatalf("sync exit %d", code)
		}
	})
	if decrypted != "world" || keyID != "github-key" {
		t.Fatalf("decrypted=%q key_id=%q", decrypted, keyID)
	}
}

func TestSyncWritesWithoutDryRunCeremony(t *testing.T) {
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
		code = Main([]string{"sync", "-f", env, "--prefix", "SD_"}, os.Stdin, &stdout, &stderr, alice.Getenv)
	})
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "dry-run") {
		t.Fatalf("sync must not dry-run by default: %q", stdout.String())
	}
	names := st.GitHub.Names()
	if len(names) != 1 || names[0] != "SD_HELLO" {
		t.Fatalf("names=%v", names)
	}
}

func TestSyncRequiresAPI(t *testing.T) {
	var stderr bytes.Buffer
	getenv := func(string) string { return "" }
	if code := Main([]string{"sync", "-f", "missing.env"}, os.Stdin, &bytes.Buffer{}, &stderr, getenv); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "SOPSDECK_GITHUB_API") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestSyncUsesManifestPrefixAndRepo(t *testing.T) {
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
		code = Main([]string{"sync", "-f", env}, os.Stdin, &stdout, &stderr, alice.Getenv)
	})
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	names := st.GitHub.Names()
	if len(names) != 1 || names[0] != "SD_HELLO" {
		t.Fatalf("names=%v", names)
	}
}

func TestSyncMappingPrintsResolvedTarget(t *testing.T) {
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
	getenv := func(key string) string {
		if key == "SOPSDECK_GITHUB_API" {
			return ""
		}
		return alice.Getenv(key)
	}
	var stdout, stderr bytes.Buffer
	var code int
	alice.WithWorld(func() {
		code = Main([]string{"sync", "-f", env, "--mapping"}, os.Stdin, &stdout, &stderr, getenv)
	})
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if strings.Contains(stderr.String(), "SOPSDECK_GITHUB_API") {
		t.Fatalf("mapping required API: %q", stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "acme/app") || !strings.Contains(out, "production") || !strings.Contains(out, "SD_") {
		t.Fatalf("stdout=%q", out)
	}
}

func TestSyncManifestKeysSelectsSubset(t *testing.T) {
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
		code = Main([]string{"sync", "-f", env}, os.Stdin, &stdout, io.Discard, alice.Getenv)
	})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	have := map[string]bool{}
	for _, name := range st.GitHub.Names() {
		have[name] = true
	}
	if !have["SD_HELLO"] || have["SD_OTHER"] {
		t.Fatalf("names=%v", st.GitHub.Names())
	}
}

func TestSyncRecordsLastSyncedNames(t *testing.T) {
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
		code = Main([]string{"sync", "-f", env}, os.Stdin, io.Discard, io.Discard, alice.Getenv)
	})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "SD_HELLO") {
		t.Fatalf("manifest missing synced names:\n%s", raw)
	}
}

func TestSyncPruneDeletesOnlyPreviouslySyncedNames(t *testing.T) {
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
	manifest := []byte("[[managed_file]]\npath = \".env.production\"\nprefix = \"SD_\"\nsynced = [\"SD_OLD\"]\n")
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
	if err := aliceCLI(alice, "sync", "-f", env, "--prune"); err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, name := range st.GitHub.Names() {
		have[name] = true
	}
	if have["SD_OLD"] {
		t.Fatalf("previously synced name still present: %v", st.GitHub.Names())
	}
	if !have["SD_EXTRA"] {
		t.Fatalf("unrecorded prefixed name was deleted: %v", st.GitHub.Names())
	}
	if !have["SD_HELLO"] {
		t.Fatalf("current name missing: %v", st.GitHub.Names())
	}
}

func TestSyncPruneLeavesUnrecordedPrefixedNames(t *testing.T) {
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
	if err := aliceCLI(alice, "sync", "-f", env, "--prefix", "SD_", "--prune"); err != nil {
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

func TestSyncSendsGitHubToken(t *testing.T) {
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
	getenv := func(key string) string {
		if key == "GH_TOKEN" {
			return "gho_test"
		}
		return alice.Getenv(key)
	}
	var code int
	alice.WithWorld(func() {
		code = Main([]string{"sync", "-f", env, "--prefix", "SD_"}, os.Stdin, io.Discard, io.Discard, getenv)
	})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got := st.GitHub.LastAuthorization(); got != "Bearer gho_test" {
		t.Fatalf("Authorization=%q", got)
	}
}

func TestSyncUsesGhAuthToken(t *testing.T) {
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
	bin := t.TempDir()
	script := []byte("#!/bin/sh\n[ \"$1\" = auth ] && [ \"$2\" = token ] && echo gho_from_gh && exit 0\nexit 1\n")
	if err := os.WriteFile(filepath.Join(bin, "gh"), script, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	getenv := func(key string) string {
		if key == "GH_TOKEN" || key == "GITHUB_TOKEN" {
			return ""
		}
		return alice.Getenv(key)
	}
	var code int
	alice.WithWorld(func() {
		code = Main([]string{"sync", "-f", env, "--prefix", "SD_"}, os.Stdin, io.Discard, io.Discard, getenv)
	})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got := st.GitHub.LastAuthorization(); got != "Bearer gho_from_gh" {
		t.Fatalf("Authorization=%q", got)
	}
}

func TestSyncManifestEnvironmentPutsEnvironmentSecrets(t *testing.T) {
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
	if err := aliceCLI(alice, "sync", "-f", env); err != nil {
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
