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

func TestDriveInvokeReferencesAndUnused(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	dir := t.TempDir()
	env := filepath.Join(dir, ".env.production")
	src, err := os.ReadFile(testdata(t, "hello.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env, src, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("process.env.HELLO\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(&drive{getenv: os.Getenv})
	t.Cleanup(srv.Close)
	refs := postInvoke(t, srv.URL, invokeReq{Cmd: "references", Path: env})
	if !bytes.Contains(refs, []byte("HELLO")) || !bytes.Contains(refs, []byte("app.js")) {
		t.Fatalf("references=%s", refs)
	}
	unused := postInvoke(t, srv.URL, invokeReq{Cmd: "unused", Path: env})
	if bytes.Contains(unused, []byte("HELLO")) {
		t.Fatalf("unused should not list HELLO: %s", unused)
	}
}

func TestDriveInvokeRenamesKeyAndRewritesReferences(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	dir := t.TempDir()
	env := filepath.Join(dir, ".env.production")
	src, err := os.ReadFile(testdata(t, "hello.env"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env, src, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("process.env.HELLO\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(&drive{getenv: os.Getenv})
	t.Cleanup(srv.Close)
	postInvoke(t, srv.URL, invokeReq{Cmd: "rename_key", Path: env, Key: "HELLO", Value: "GREETING", Yes: true})
	got := postInvoke(t, srv.URL, invokeReq{Cmd: "get_managed_file", Path: env})
	if !bytes.Contains(got, []byte("GREETING")) || bytes.Contains(got, []byte("HELLO")) {
		t.Fatalf("get after rename=%s", got)
	}
	appBody, _ := os.ReadFile(filepath.Join(dir, "app.js"))
	if !bytes.Contains(appBody, []byte("GREETING")) {
		t.Fatalf("app.js not rewritten: %s", appBody)
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

func TestDriveInvokeReturnsPublishMapping(t *testing.T) {
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
	srv := httptest.NewServer(&drive{getenv: alice.Getenv})
	t.Cleanup(srv.Close)
	body := postInvoke(t, srv.URL, invokeReq{Cmd: "get_publish_mapping", Path: env})
	if !bytes.Contains(body, []byte("acme/app")) || !bytes.Contains(body, []byte("production")) || !bytes.Contains(body, []byte("SD_")) {
		t.Fatalf("mapping=%s", body)
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

	removed := postInvoke(t, srv.URL, invokeReq{
		Cmd:       "remove_recipient",
		Path:      env,
		PublicKey: bob.PublicKey,
	})
	if !bytes.Contains(removed, []byte("Git history")) {
		t.Fatalf("remove=%s", removed)
	}

	stdout.Reset()
	stderr.Reset()
	bob.WithWorld(func() {
		code = Main([]string{"get", "HELLO", "-f", env}, os.Stdin, &stdout, &stderr, bob.Getenv)
	})
	if code == 0 {
		t.Fatal("bob still has Access after remove")
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
	t.Setenv("SOPSDECK_TEAM_ROOT", "")
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
	if _, err := os.Stat(filepath.Join(info.Project, ".sopsdeck.toml")); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"files", info.Project}, os.Stdin, &stdout, &stderr, getenv); code != 0 {
		t.Fatalf("files exit %d stderr=%q", code, stderr.String())
	}
	var files []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &files); err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 || files[0].Name != ".env.production" {
		t.Fatalf("files=%v, want .env.production first", files)
	}
	if _, err := os.Stat(filepath.Join(info.Project, "apps", "web", ".env")); err != nil {
		t.Fatal(err)
	}
	if len(info.Projects) != 3 {
		t.Fatalf("projects=%v", info.Projects)
	}
	if info.Projects[0] != info.Project {
		t.Fatalf("first project %q", info.Projects[0])
	}
	if filepath.Base(info.Projects[1]) != "atlas-web" || filepath.Base(info.Projects[2]) != "docs-site" {
		t.Fatalf("projects=%v", info.Projects)
	}
	if _, err := os.Stat(filepath.Join(info.Projects[1], "eas.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(info.Projects[1], ".sopsdeck.toml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(info.Projects[2], ".env")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(info.Projects[2], ".sopsdeck.toml")); err != nil {
		t.Fatal(err)
	}
}

func TestDriveTeamRootUsesSharedStudio(t *testing.T) {
	root := t.TempDir()
	getenv := func(key string) string {
		switch key {
		case "SOPSDECK_TEAM_ROOT":
			return root
		default:
			return os.Getenv(key)
		}
	}
	aliceInfo, aliceEnv, err := seedDemoForEnv("alice", getenv)
	if err != nil {
		t.Fatal(err)
	}
	bobInfo, bobEnv, err := seedDemoForEnv("bob", getenv)
	if err != nil {
		t.Fatal(err)
	}
	if aliceInfo.Project != filepath.Join(root, "alice-home", "checkout") {
		t.Fatalf("alice project %q", aliceInfo.Project)
	}
	if bobInfo.Project != filepath.Join(root, "bob-home", "checkout") {
		t.Fatalf("bob project %q", bobInfo.Project)
	}
	if aliceInfo.BobPublicKey == bobInfo.BobPublicKey {
		t.Fatal("both windows received the same teammate key")
	}
	if !strings.HasPrefix(aliceInfo.BobPublicKey, "age1") || !strings.HasPrefix(bobInfo.BobPublicKey, "age1") {
		t.Fatalf("keys alice-sees=%q bob-sees=%q", aliceInfo.BobPublicKey, bobInfo.BobPublicKey)
	}
	if aliceEnv("HOME") != filepath.Join(root, "alice-home") {
		t.Fatalf("alice HOME %q", aliceEnv("HOME"))
	}
	if bobEnv("HOME") != filepath.Join(root, "bob-home") {
		t.Fatalf("bob HOME %q", bobEnv("HOME"))
	}
	if aliceEnv("SOPSDECK_DEV_PROJECT") != aliceInfo.Project {
		t.Fatalf("alice boot %q", aliceEnv("SOPSDECK_DEV_PROJECT"))
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

func TestDriveInvokeReviewHistoryAndRestore(t *testing.T) {
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
	if err := aliceCLI(alice, "commit", "-m", "seed production", "-f", env); err != nil {
		t.Fatal(err)
	}
	if err := aliceCLI(alice, "set", "HELLO", "next", "-f", env); err != nil {
		t.Fatal(err)
	}
	if err := aliceCLI(alice, "commit", "-m", "rotate hello", "-f", env); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(&drive{getenv: alice.Getenv})
	t.Cleanup(srv.Close)

	hist := postInvoke(t, srv.URL, invokeReq{Cmd: "history_managed_file", Path: env})
	if !bytes.Contains(hist, []byte("seed production")) || !bytes.Contains(hist, []byte("rotate hello")) {
		t.Fatalf("history=%s", hist)
	}
	var histBody struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(hist, &histBody); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(histBody.Result), "\n")
	oldest := strings.Fields(lines[len(lines)-1])
	rev := oldest[0]
	got := postInvoke(t, srv.URL, invokeReq{Cmd: "get_managed_file", Path: env, At: rev})
	if !bytes.Contains(got, []byte("world")) {
		t.Fatalf("get at=%s", got)
	}

	_ = postInvoke(t, srv.URL, invokeReq{Cmd: "set_managed_key", Path: env, Key: "HELLO", Value: "later"})
	review := postInvoke(t, srv.URL, invokeReq{Cmd: "review_managed_file", Path: env})
	if !bytes.Contains(review, []byte("HELLO")) || !bytes.Contains(review, []byte("later")) {
		t.Fatalf("review=%s", review)
	}

	_ = postInvoke(t, srv.URL, invokeReq{Cmd: "restore_managed_file", Path: env, At: rev})
	after := postInvoke(t, srv.URL, invokeReq{Cmd: "get_managed_file", Path: env})
	if !bytes.Contains(after, []byte("world")) {
		t.Fatalf("restored=%s", after)
	}
}

func TestDriveInvokeCreatesEmptyManagedFile(t *testing.T) {
	st, err := studio.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	alice, err := st.User("alice", "alice@sopsdeck.example")
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(&drive{getenv: alice.Getenv})
	t.Cleanup(srv.Close)

	staging := filepath.Join(alice.Home, ".env.staging")
	_ = postInvoke(t, srv.URL, invokeReq{Cmd: "create_managed_file", Path: staging})

	list := postInvoke(t, srv.URL, invokeReq{Cmd: "list_managed_files", Path: alice.Home})
	if !bytes.Contains(list, []byte(".env.staging")) {
		t.Fatalf("list=%s", list)
	}

	got := postInvoke(t, srv.URL, invokeReq{Cmd: "get_managed_file", Path: staging})
	var body struct {
		Result []map[string]string `json:"result"`
	}
	if err := json.Unmarshal(got, &body); err != nil {
		t.Fatalf("get=%s: %v", got, err)
	}
	if len(body.Result) != 0 {
		t.Fatalf("get=%s", got)
	}
}

func TestDriveInvokeDeletesManagedKey(t *testing.T) {
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
	if err := aliceCLI(alice, "set", "KEEP", "yes", "-f", env); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(&drive{getenv: alice.Getenv})
	t.Cleanup(srv.Close)

	_ = postInvoke(t, srv.URL, invokeReq{Cmd: "del_managed_key", Path: env, Key: "HELLO"})
	got := postInvoke(t, srv.URL, invokeReq{Cmd: "get_managed_file", Path: env})
	if bytes.Contains(got, []byte("HELLO")) {
		t.Fatalf("get still has deleted key: %s", got)
	}
	if !bytes.Contains(got, []byte("KEEP")) {
		t.Fatalf("get=%s", got)
	}
}

func TestDriveInvokeMarksAndUpdatesEncryptedJSONLeaves(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	root := t.TempDir()
	file := filepath.Join(root, "eas.json")
	mustWriteFile(t, file, `{"build":{"env":{"SECRET":"value","PUBLIC":"safe"}}}`)
	var stdout, stderr bytes.Buffer
	mustCLI(t, cmdProject([]string{"init", root, "--file", "eas.json", "--keys", "build.env.SECRET"}, &stdout, &stderr, os.Getenv), &stderr, "init")

	srv := httptest.NewServer(&drive{getenv: os.Getenv})
	t.Cleanup(srv.Close)
	got := postInvoke(t, srv.URL, invokeReq{Cmd: "get_managed_file", Path: file})
	if !bytes.Contains(got, []byte(`"encrypted":true`)) || !bytes.Contains(got, []byte(`"encrypted":false`)) {
		t.Fatalf("get=%s", got)
	}

	_ = postInvoke(t, srv.URL, invokeReq{Cmd: "set_encrypted_keys", Path: file, Keys: []string{"build.env.PUBLIC"}})
	text := mustReadFile(t, file)
	mustContain(t, text, `"PUBLIC": "ENC[`, "updated path was not encrypted")
	mustContain(t, text, `"SECRET": "value"`, "removed path stayed encrypted")
}
