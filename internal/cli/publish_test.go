package cli

import (
	"bytes"
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

func TestPublishPrunesPrefixedSecretsNotInFile(t *testing.T) {
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
	for _, name := range st.GitHub.Names() {
		if name == "SD_OLD" {
			t.Fatalf("pruned name still present: %v", st.GitHub.Names())
		}
	}
}
