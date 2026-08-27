package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sopsdeck/internal/studio"
)

func TestDriveInvokeListsAndReadsManagedFile(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	dir := t.TempDir()
	dst := filepath.Join(dir, ".env.production")
	src, err := os.ReadFile(testdata(t, "hello.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, src, 0o600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(&drive{getenv: os.Getenv})
	t.Cleanup(srv.Close)

	list := postInvoke(t, srv.URL, invokeReq{Cmd: "list_managed_files", Path: dir})
	if !bytes.Contains(list, []byte(".env.production")) {
		t.Fatalf("list=%s", list)
	}

	got := postInvoke(t, srv.URL, invokeReq{Cmd: "get_managed_file", Path: dst})
	if !bytes.Contains(got, []byte("HELLO")) || !bytes.Contains(got, []byte("world")) {
		t.Fatalf("get=%s", got)
	}
}

func TestDriveListenMustBeLocalhost(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := cmdDrive([]string{"--listen", "0.0.0.0:4174"}, &stdout, &stderr, os.Getenv); code != 1 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "127.0.0.1") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestDriveInvokePublishesToFakeGitHub(t *testing.T) {
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

	srv := httptest.NewServer(&drive{getenv: alice.Getenv})
	t.Cleanup(srv.Close)
	body := postInvoke(t, srv.URL, invokeReq{
		Cmd:    "publish_managed_file",
		Path:   env,
		Prefix: "SD_",
		Yes:    true,
	})
	if !bytes.Contains(body, []byte("published")) {
		t.Fatalf("publish=%s", body)
	}
	names := st.GitHub.Names()
	found := false
	for _, name := range names {
		if name == "SD_HELLO" {
			found = true
		}
	}
	if !found {
		t.Fatalf("github names=%v", names)
	}
}

func TestDriveHealthAndDemoJSON(t *testing.T) {
	srv := httptest.NewServer(&drive{
		uiRoot: t.TempDir(),
		demo: &demoInfo{
			Project:      "/tmp/alice",
			BobPublicKey: "age1test",
			GitHubAPI:    "http://127.0.0.1:1",
		},
	})
	t.Cleanup(srv.Close)

	health, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = health.Body.Close() }()
	if health.StatusCode != http.StatusOK {
		t.Fatalf("health %d", health.StatusCode)
	}

	demo, err := http.Get(srv.URL + "/demo")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = demo.Body.Close() }()
	var info demoInfo
	if err := json.NewDecoder(demo.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info.BobPublicKey != "age1test" {
		t.Fatalf("demo=%+v", info)
	}
}

func TestDriveInvokeAddsRecipient(t *testing.T) {
	st, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	alice, err := st.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := st.Identity("bob", "bob@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(alice.Home, ".env.production")
	if err := aliceCLI(alice, "set", "HELLO", "shared", "-f", env); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(&drive{getenv: alice.Getenv})
	t.Cleanup(srv.Close)
	_ = postInvoke(t, srv.URL, invokeReq{
		Cmd:       "add_recipient",
		Path:      env,
		PublicKey: bob.PublicKey,
	})

	var stdout, stderr bytes.Buffer
	var code int
	bob.WithWorld(func() {
		code = Main([]string{"get", "HELLO", "-f", env}, os.Stdin, &stdout, &stderr, bob.Getenv)
	})
	if code != 0 {
		t.Fatalf("bob get: %s", stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "shared" {
		t.Fatalf("bob get %q", stdout.String())
	}
}

func postInvoke(t *testing.T, base string, req invokeReq) []byte {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(base+"/invoke", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body=%s", resp.StatusCode, buf.String())
	}
	return buf.Bytes()
}

func TestSeedDemoCreatesSharedManagedFile(t *testing.T) {
	info, getenv, err := seedDemo()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(info.BobPublicKey, "age1") {
		t.Fatalf("bob key %q", info.BobPublicKey)
	}
	if getenv("SOPSDECK_DEV_PROJECT") != info.Project {
		t.Fatalf("project %q", getenv("SOPSDECK_DEV_PROJECT"))
	}
	if _, err := os.Stat(filepath.Join(info.Project, ".env.production")); err != nil {
		t.Fatal(err)
	}
}

func TestDriveFlagErrors(t *testing.T) {
	var stderr bytes.Buffer
	if code := cmdDrive([]string{"--listen"}, io.Discard, &stderr, os.Getenv); code != 1 {
		t.Fatalf("listen exit %d", code)
	}
	stderr.Reset()
	if code := cmdDrive([]string{"--ui"}, io.Discard, &stderr, os.Getenv); code != 1 {
		t.Fatalf("ui exit %d", code)
	}
	stderr.Reset()
	if code := cmdDrive([]string{"--nope"}, io.Discard, &stderr, os.Getenv); code != 1 {
		t.Fatalf("nope exit %d", code)
	}
}

func TestDriveInvokeSetCommitSyncAndErrors(t *testing.T) {
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
	if err := aliceCLI(alice, "commit", "-m", "seed", "-f", env); err != nil {
		t.Fatal(err)
	}
	if _, err := alice.Git("push", "-u", "origin", "main"); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(&drive{getenv: alice.Getenv})
	t.Cleanup(srv.Close)

	_ = postInvoke(t, srv.URL, invokeReq{Cmd: "set_managed_key", Path: env, Key: "HELLO", Value: "next"})
	got := postInvoke(t, srv.URL, invokeReq{Cmd: "get_managed_file", Path: env})
	if !bytes.Contains(got, []byte("next")) {
		t.Fatalf("get=%s", got)
	}
	_ = postInvoke(t, srv.URL, invokeReq{Cmd: "commit_managed_file", Path: env, Message: "rotate hello"})
	_ = postInvoke(t, srv.URL, invokeReq{Cmd: "sync_project", Path: env})
	boot := postInvoke(t, srv.URL, invokeReq{Cmd: "boot_project"})
	if bytes.Contains(boot, []byte(alice.Home)) {
		t.Fatal("boot should be empty without SOPSDECK_DEV_PROJECT")
	}

	resp, err := http.Get(srv.URL + "/invoke")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("get invoke %d", resp.StatusCode)
	}

	bad, err := http.Post(srv.URL+"/invoke", "application/json", strings.NewReader("{"))
	if err != nil {
		t.Fatal(err)
	}
	_ = bad.Body.Close()
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad json %d", bad.StatusCode)
	}

	unknown, err := http.Post(srv.URL+"/invoke", "application/json", strings.NewReader(`{"cmd":"nope"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = unknown.Body.Close()
	if unknown.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown %d", unknown.StatusCode)
	}

	demo404, err := http.Get(srv.URL + "/demo")
	if err != nil {
		t.Fatal(err)
	}
	_ = demo404.Body.Close()
	if demo404.StatusCode != http.StatusNotFound {
		t.Fatalf("demo %d", demo404.StatusCode)
	}
}
