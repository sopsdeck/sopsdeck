package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishFailedPutExplainsRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	t.Setenv("SOPSDECK_GITHUB_API", srv.URL)
	t.Setenv("SOPSDECK_GITHUB_REPO", "studio/demo")

	env := filepath.Join(t.TempDir(), "hello.env")
	src, err := os.ReadFile(testdata(t, "hello.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env, src, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Main([]string{"publish", "-f", env, "--yes"}, os.Stdin, &stdout, &stderr, os.Getenv)
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	got := stderr.String()
	if strings.Contains(got, "502") && strings.Contains(got, "PUT") {
		t.Fatalf("raw HTTP leaked: %q", got)
	}
	if !strings.Contains(got, "Publish did not finish") {
		t.Fatalf("stderr=%q", got)
	}
}
