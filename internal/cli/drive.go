package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sopsdeck/internal/managed"
	"sopsdeck/internal/studio"
)

type invokeReq struct {
	Cmd         string             `json:"cmd"`
	Path        string             `json:"path"`
	Key         string             `json:"key"`
	Value       string             `json:"value"`
	Text        string             `json:"text"`
	Name        string             `json:"name"`
	Kind        string             `json:"kind"`
	Scope       string             `json:"scope"`
	Org         string             `json:"org"`
	Repo        string             `json:"repo"`
	Environment string             `json:"environment"`
	Visibility  string             `json:"visibility"`
	Message     string             `json:"message"`
	Email       string             `json:"email"`
	PublicKey   string             `json:"publicKey"`
	Prefix      string             `json:"prefix"`
	Yes         bool               `json:"yes"`
	Prune       bool               `json:"prune"`
	At          string             `json:"at"`
	Files       []projectSelection `json:"files"`
	File        string             `json:"file"`
	Keys        []string           `json:"keys"`
}

type demoInfo struct {
	Project      string   `json:"project"`
	Projects     []string `json:"projects"`
	BobPublicKey string   `json:"bobPublicKey"`
	GitHubAPI    string   `json:"githubAPI"`
}

type drive struct {
	uiRoot string
	getenv func(string) string
	demo   *demoInfo
}

func cmdDrive(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	listen := "127.0.0.1:4174"
	uiRoot := getenv("SOPSDECK_UI_ROOT")
	demo := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--listen":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "drive: --listen requires addr")
				return 1
			}
			listen = args[i]
		case "--ui":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "drive: --ui requires a folder")
				return 1
			}
			uiRoot = args[i]
		case "--demo":
			demo = true
		default:
			fmt.Fprintln(stderr, "usage: sopsdeck drive [--listen 127.0.0.1:4174] [--ui desktop/src] [--demo]")
			return 1
		}
	}
	if !strings.HasPrefix(listen, "127.0.0.1:") && !strings.HasPrefix(listen, "localhost:") {
		fmt.Fprintln(stderr, "drive: listen address must be 127.0.0.1")
		return 1
	}
	if uiRoot == "" {
		uiRoot = filepath.Join("desktop", "src")
	}
	if abs, err := filepath.Abs(uiRoot); err == nil {
		uiRoot = abs
	}
	handler := &drive{uiRoot: uiRoot, getenv: getenv}
	if demo {
		demoUser := getenv("SOPSDECK_DEMO_USER")
		if demoUser == "" {
			demoUser = "checkout"
		}
		info, getenvDemo, err := seedDemoFor(demoUser)
		if err != nil {
			fmt.Fprintf(stderr, "drive: %v\n", err)
			return 1
		}
		handler.demo = info
		handler.getenv = getenvDemo
	}
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		fmt.Fprintf(stderr, "drive: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "listening on http://%s\n", ln.Addr())
	if err := http.Serve(ln, handler); err != nil {
		fmt.Fprintf(stderr, "drive: %v\n", err)
		return 1
	}
	return 0
}

func seedDemo() (*demoInfo, func(string) string, error) {
	return seedDemoFor("checkout")
}

func seedDemoFor(demoUser string) (*demoInfo, func(string) string, error) {
	dir, err := os.MkdirTemp("", "sopsdeck-demo-")
	if err != nil {
		return nil, nil, err
	}
	st, err := studio.New(dir)
	if err != nil {
		return nil, nil, err
	}
	if demoUser == "" {
		demoUser = "checkout"
	}
	alice, err := st.User(demoUser, demoUser+"@sopsdeck.example")
	if err != nil {
		return nil, nil, err
	}
	teammate := "bob"
	if demoUser == teammate {
		teammate = "alice"
	}
	bob, err := st.Identity(teammate, teammate+"@sopsdeck.example")
	if err != nil {
		return nil, nil, err
	}
	if err := seedFile(alice, filepath.Join(alice.Home, ".env.production"), "STRIPE_SECRET", "sk_test_demo", "seed production"); err != nil {
		return nil, nil, err
	}
	if err := seedFile(alice, filepath.Join(alice.Home, "eas.json"), "EXPO_PUBLIC_API_URL", "https://api.acme.test", "seed eas.json"); err != nil {
		return nil, nil, err
	}
	if err := seedFile(alice, filepath.Join(alice.Home, "compose.yaml"), "POSTGRES_PASSWORD", "acme_pg_demo_password", "seed compose.yaml"); err != nil {
		return nil, nil, err
	}
	if err := seedFile(alice, filepath.Join(alice.Home, "apps", "web", ".env"), "NEXT_PUBLIC_APP_URL", "https://app.acme.test", "seed nested env"); err != nil {
		return nil, nil, err
	}
	if _, err := alice.Git("push", "-u", "origin", "main"); err != nil {
		return nil, nil, err
	}
	manifest := []byte("[[managed_file]]\npath = \".env.production\"\nrepo = \"studio/demo\"\nprefix = \"SD_\"\n")
	if err := os.WriteFile(filepath.Join(alice.Home, ".sopsdeck.toml"), manifest, 0o600); err != nil {
		return nil, nil, err
	}
	atlas, err := seedSiblingProject(alice, "atlas-web", "eas.json", "EXPO_PUBLIC_API_URL", "https://api.atlas.test", "seed atlas eas.json")
	if err != nil {
		return nil, nil, err
	}
	docs, err := seedSiblingProject(alice, "docs-site", ".env", "DOCS_SEARCH_KEY", "search_demo_key", "seed docs env")
	if err != nil {
		return nil, nil, err
	}
	info := &demoInfo{
		Project:      alice.Home,
		Projects:     []string{alice.Home, atlas, docs},
		BobPublicKey: bob.PublicKey,
		GitHubAPI:    alice.Getenv("SOPSDECK_GITHUB_API"),
	}
	getenv := func(key string) string {
		if key == "SOPSDECK_DEV_PROJECT" {
			return alice.Home
		}
		return alice.Getenv(key)
	}
	return info, getenv, nil
}

func seedFile(alice *studio.User, path, key, value, msg string) error {
	if err := aliceCLI(alice, "set", key, value, "-f", path); err != nil {
		return err
	}
	return aliceCLI(alice, "commit", "-m", msg, "-f", path)
}

func seedSiblingProject(alice *studio.User, name, rel, key, value, msg string) (string, error) {
	dir := filepath.Join(filepath.Dir(alice.Home), name)
	if err := initGitDir(dir, name, "alice@sopsdeck.example"); err != nil {
		return "", err
	}
	if err := seedFile(alice, filepath.Join(dir, rel), key, value, msg); err != nil {
		return "", err
	}
	return dir, nil
}

func initGitDir(dir, name, email string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	steps := [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", email},
		{"config", "user.name", name},
	}
	for _, args := range steps {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func aliceCLI(alice *studio.User, args ...string) error {
	var stderr strings.Builder
	var code int
	alice.WithWorld(func() {
		code = Main(args, os.Stdin, io.Discard, &stderr, alice.Getenv)
	})
	if code != 0 {
		return fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (d *drive) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/invoke":
		d.handleInvoke(w, r)
		return
	case "/health":
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
		return
	case "/demo":
		if d.demo == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(d.demo)
		return
	}
	http.FileServer(http.Dir(d.uiRoot)).ServeHTTP(w, r)
}

func (d *drive) handleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req invokeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInvokeErr(w, err.Error())
		return
	}
	result, err := d.invoke(req)
	if err != nil {
		writeInvokeErr(w, err.Error())
		return
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"result": result})
}

func writeInvokeErr(w http.ResponseWriter, msg string) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (d *drive) invoke(req invokeReq) (any, error) {
	getenv := d.getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	restore := applyProcessEnv(getenv)
	defer restore()
	switch req.Cmd {
	case "list_managed_files":
		return managed.List(req.Path)
	case "inspect_project":
		return invokeProject([]string{"files", req.Path}, getenv)
	case "initialize_project":
		args := []string{"init", req.Path}
		for _, file := range req.Files {
			args = append(args, "--file", file.Path)
			if len(file.Keys) > 0 {
				args = append(args, "--keys", strings.Join(file.Keys, ","))
			}
		}
		return invokeProject(args, getenv)
	case "add_project_file":
		args := []string{"add", req.Path, "--file", req.File}
		if len(req.Keys) > 0 {
			args = append(args, "--keys", strings.Join(req.Keys, ","))
		}
		return invokeProject(args, getenv)
	case "get_managed_file":
		return invokeGet(req.Path, req.At)
	case "get_managed_file_contents":
		return invokeContents(req.Path)
	case "copy_text":
		return nil, copyToClipboard(req.Text)
	case "get_managed_file_status":
		return invokeFileStatus(req.Path)
	case "get_account":
		return accountForPath(req.Path, getenv), nil
	case "configure_account":
		if err := configureGitIdentity(req.Path, req.Name, req.Email); err != nil {
			return nil, err
		}
		return accountForPath(req.Path, getenv), nil
	case "create_user_identity":
		return invokeCreateUserIdentity(req.Path, getenv)
	case "list_file_access":
		return invokeRecipientList(req.Path, getenv)
	case "create_robot_identity":
		return invokeRobot(req.Name)
	case "configure_integration":
		return nil, configureIntegration(req.Path, req.Scope, req.Repo, req.Org, req.Environment, req.Prefix, req.Visibility)
	case "unlock_managed_file":
		return nil, invokeFileLock("unlock", req.Path, getenv)
	case "lock_managed_file":
		return nil, invokeFileLock("lock", req.Path, getenv)
	case "set_managed_key":
		return nil, invokeSet(req, getenv)
	case "del_managed_key":
		return nil, invokeDel(req)
	case "create_managed_file":
		return nil, invokeCreate(req, getenv)
	case "commit_managed_file":
		return nil, invokeCommit(req)
	case "sync_project":
		return nil, d.invokeSync(req.Path)
	case "review_managed_file":
		return invokeReview(req)
	case "history_managed_file":
		return invokeHistory(req)
	case "restore_managed_file":
		return nil, invokeRestore(req)
	case "add_recipient":
		return nil, invokeRecipientAdd(req, getenv)
	case "remove_recipient":
		return invokeRecipientRemove(req, getenv)
	case "publish_managed_file":
		return invokePublish(req, getenv)
	case "get_publish_mapping":
		return invokePublishMapping(req, getenv)
	case "references", "unused", "rename_key":
		return invokeReferenceCommands(req)
	case "pick_project_folder", "boot_project":
		project := getenv("SOPSDECK_DEV_PROJECT")
		if project == "" {
			return nil, nil
		}
		return project, nil
	default:
		return nil, fmt.Errorf("unknown command %q", req.Cmd)
	}
}

func invokeProject(args []string, getenv func(string) string) (any, error) {
	var stdout, stderr strings.Builder
	if err := cliErr(cmdProject(args, &stdout, &stderr, getenv), &stderr); err != nil {
		return nil, err
	}
	if args[0] == "init" || args[0] == "add" {
		return strings.TrimSpace(stdout.String()), nil
	}
	var state projectState
	if err := json.Unmarshal([]byte(stdout.String()), &state); err != nil {
		return nil, err
	}
	return state, nil
}

func cliErr(code int, stderr *strings.Builder) error {
	if code != 0 {
		msg := strings.TrimSpace(stderr.String())
		recordError(os.Getenv, msg)
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func invokeGet(path, at string) (any, error) {
	args := []string{"-f", path, "--output", "json"}
	if at != "" {
		args = append(args, "--at", at)
	}
	var stdout, stderr strings.Builder
	if err := cliErr(cmdGet(args, &stdout, &stderr), &stderr); err != nil {
		return nil, err
	}
	var pairs map[string]string
	if err := json.Unmarshal([]byte(stdout.String()), &pairs); err != nil {
		return nil, err
	}
	out := make([]map[string]string, 0, len(pairs))
	for key, value := range pairs {
		out = append(out, map[string]string{"key": key, "value": value})
	}
	return out, nil
}

func invokeContents(path string) (any, error) {
	var stdout, stderr strings.Builder
	if err := cliErr(cmdGet([]string{"-f", path}, &stdout, &stderr), &stderr); err != nil {
		return nil, err
	}
	return stdout.String(), nil
}

func invokeFileStatus(path string) (any, error) {
	var stdout, stderr strings.Builder
	if err := cliErr(cmdFileStatus([]string{"-f", path}, &stdout, &stderr), &stderr); err != nil {
		return nil, err
	}
	var status map[string]bool
	if err := json.Unmarshal([]byte(stdout.String()), &status); err != nil {
		return nil, err
	}
	return status, nil
}

func invokeFileLock(action, path string, getenv func(string) string) error {
	var stderr strings.Builder
	if action == "unlock" {
		return cliErr(cmdUnlock([]string{"-f", path}, io.Discard, &stderr), &stderr)
	}
	return cliErr(cmdLock([]string{"-f", path}, io.Discard, &stderr, getenv), &stderr)
}

func invokeRecipientList(path string, getenv func(string) string) (any, error) {
	var stdout, stderr strings.Builder
	if err := cliErr(cmdRecipient([]string{"list", "-f", path}, &stdout, &stderr, getenv), &stderr); err != nil {
		return nil, err
	}
	var list []accessRecipient
	if err := json.Unmarshal([]byte(stdout.String()), &list); err != nil {
		return nil, err
	}
	return list, nil
}

func invokeRobot(name string) (any, error) {
	var stdout, stderr strings.Builder
	if err := cliErr(cmdRobot([]string{"create", name}, &stdout, &stderr), &stderr); err != nil {
		return nil, err
	}
	var robot robotIdentity
	if err := json.Unmarshal([]byte(stdout.String()), &robot); err != nil {
		return nil, err
	}
	return robot, nil
}

func invokeSet(req invokeReq, getenv func(string) string) error {
	var stderr strings.Builder
	return cliErr(cmdSet([]string{req.Key, req.Value, "-f", req.Path}, strings.NewReader(""), io.Discard, &stderr, getenv), &stderr)
}

func invokeDel(req invokeReq) error {
	var stderr strings.Builder
	return cliErr(cmdDel([]string{req.Key, "-f", req.Path}, io.Discard, &stderr), &stderr)
}

func invokeCreate(req invokeReq, getenv func(string) string) error {
	var stderr strings.Builder
	return cliErr(cmdSet([]string{"-f", req.Path}, strings.NewReader(""), io.Discard, &stderr, getenv), &stderr)
}

func invokeCommit(req invokeReq) error {
	var stderr strings.Builder
	return cliErr(cmdCommit([]string{"-m", req.Message, "-f", req.Path}, io.Discard, &stderr), &stderr)
}

func invokeReview(req invokeReq) (any, error) {
	var stdout, stderr strings.Builder
	if err := cliErr(cmdReview([]string{"-f", req.Path}, &stdout, &stderr), &stderr); err != nil {
		return nil, err
	}
	return stdout.String(), nil
}

func invokeHistory(req invokeReq) (any, error) {
	var stdout, stderr strings.Builder
	if err := cliErr(cmdHistory([]string{"-f", req.Path}, &stdout, &stderr), &stderr); err != nil {
		return nil, err
	}
	return stdout.String(), nil
}

func invokeRestore(req invokeReq) error {
	var stderr strings.Builder
	return cliErr(cmdRestore([]string{"-f", req.Path, "--at", req.At}, io.Discard, &stderr), &stderr)
}

func (d *drive) invokeSync(path string) error {
	top, err := gitTopLevel(path)
	if err != nil {
		return err
	}
	return syncAt(top)
}

func invokeRecipientAdd(req invokeReq, getenv func(string) string) error {
	var stderr strings.Builder
	args := []string{"add", req.PublicKey, "-f", req.Path}
	if req.Name != "" {
		args = append(args, "--name", req.Name)
	}
	if req.Kind != "" {
		args = append(args, "--kind", req.Kind)
	}
	return cliErr(cmdRecipient(args, io.Discard, &stderr, getenv), &stderr)
}

func invokeRecipientRemove(req invokeReq, getenv func(string) string) (any, error) {
	var stderr strings.Builder
	if err := cliErr(cmdRecipient([]string{"remove", req.PublicKey, "-f", req.Path}, io.Discard, &stderr, getenv), &stderr); err != nil {
		return nil, err
	}
	return strings.TrimSpace(stderr.String()), nil
}

func invokePublish(req invokeReq, getenv func(string) string) (any, error) {
	args := []string{"-f", req.Path}
	for _, option := range [][2]string{{"--scope", req.Scope}, {"--repo", req.Repo}, {"--org", req.Org}, {"--environment", req.Environment}, {"--visibility", req.Visibility}} {
		if option[1] != "" {
			args = append(args, option[0], option[1])
		}
	}
	if req.Prefix != "" {
		args = append(args, "--prefix", req.Prefix)
	}
	if req.Yes {
		args = append(args, "--yes")
	}
	if req.Prune {
		args = append(args, "--prune")
	}
	var stdout, stderr strings.Builder
	if err := cliErr(cmdPublish(args, &stdout, &stderr, getenv), &stderr); err != nil {
		return nil, err
	}
	return strings.TrimSpace(stdout.String()), nil
}

func invokePublishMapping(req invokeReq, getenv func(string) string) (any, error) {
	var stdout, stderr strings.Builder
	if err := cliErr(cmdPublish([]string{"-f", req.Path, "--mapping"}, &stdout, &stderr, getenv), &stderr); err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout.String()), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func invokeReferenceCommands(req invokeReq) (any, error) {
	switch req.Cmd {
	case "references":
		return invokeReferences(req)
	case "unused":
		return invokeUnused(req)
	case "rename_key":
		return nil, invokeRenameKey(req)
	default:
		return nil, fmt.Errorf("unknown command %q", req.Cmd)
	}
}

func invokeReferences(req invokeReq) (any, error) {
	var stdout, stderr strings.Builder
	if err := cliErr(cmdReferences([]string{"-f", req.Path}, &stdout, &stderr), &stderr); err != nil {
		return nil, err
	}
	var out []keyReference
	if err := json.Unmarshal([]byte(stdout.String()), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func invokeUnused(req invokeReq) (any, error) {
	var stdout, stderr strings.Builder
	if err := cliErr(cmdUnused([]string{"-f", req.Path}, &stdout, &stderr), &stderr); err != nil {
		return nil, err
	}
	var out []string
	if err := json.Unmarshal([]byte(stdout.String()), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func invokeRenameKey(req invokeReq) error {
	args := []string{req.Key, req.Value, "-f", req.Path}
	if req.Yes {
		args = append(args, "--yes")
	}
	var stdout, stderr strings.Builder
	if err := cliErr(cmdRename(args, &stdout, &stderr), &stderr); err != nil {
		return err
	}
	return nil
}

func gitTopLevel(path string) (string, error) {
	dir := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		dir = filepath.Dir(path)
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git project")
	}
	return strings.TrimSpace(string(out)), nil
}

func applyProcessEnv(getenv func(string) string) func() {
	keys := []string{"SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD", "SOPSDECK_STATE_DIR", "SOPSDECK_KEYCHAIN_DIR", "SOPSDECK_GITHUB_API", "SOPSDECK_GITHUB_REPO"}
	prev := map[string]*string{}
	for _, key := range keys {
		if cur, ok := os.LookupEnv(key); ok {
			c := cur
			prev[key] = &c
		} else {
			prev[key] = nil
		}
		if value := getenv(key); value == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, value)
		}
	}
	return func() {
		for key, value := range prev {
			if value == nil {
				_ = os.Unsetenv(key)
				continue
			}
			_ = os.Setenv(key, *value)
		}
	}
}
