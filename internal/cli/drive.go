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
	Cmd       string `json:"cmd"`
	Path      string `json:"path"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Message   string `json:"message"`
	PublicKey string `json:"publicKey"`
	Prefix    string `json:"prefix"`
	Yes       bool   `json:"yes"`
	Prune     bool   `json:"prune"`
	At        string `json:"at"`
}

type demoInfo struct {
	Project      string `json:"project"`
	BobPublicKey string `json:"bobPublicKey"`
	GitHubAPI    string `json:"githubAPI"`
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
		info, getenvDemo, err := seedDemo()
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
	dir, err := os.MkdirTemp("", "sopsdeck-demo-")
	if err != nil {
		return nil, nil, err
	}
	st, err := studio.New(dir)
	if err != nil {
		return nil, nil, err
	}
	alice, err := st.User("checkout", "alice@sopsdeck.example")
	if err != nil {
		return nil, nil, err
	}
	bob, err := st.Identity("bob", "bob@sopsdeck.example")
	if err != nil {
		return nil, nil, err
	}
	env := filepath.Join(alice.Home, ".env.production")
	if err := aliceCLI(alice, "set", "STRIPE_SECRET", "sk_test_demo", "-f", env); err != nil {
		return nil, nil, err
	}
	if err := aliceCLI(alice, "commit", "-m", "seed production", "-f", env); err != nil {
		return nil, nil, err
	}
	eas := filepath.Join(alice.Home, "eas.json")
	if err := aliceCLI(alice, "set", "EXPO_PUBLIC_API_URL", "https://api.acme.test", "-f", eas); err != nil {
		return nil, nil, err
	}
	if err := aliceCLI(alice, "commit", "-m", "seed eas.json", "-f", eas); err != nil {
		return nil, nil, err
	}
	compose := filepath.Join(alice.Home, "compose.yaml")
	if err := aliceCLI(alice, "set", "POSTGRES_PASSWORD", "acme_pg_demo_password", "-f", compose); err != nil {
		return nil, nil, err
	}
	if err := aliceCLI(alice, "commit", "-m", "seed compose.yaml", "-f", compose); err != nil {
		return nil, nil, err
	}
	if _, err := alice.Git("push", "-u", "origin", "main"); err != nil {
		return nil, nil, err
	}
	manifest := []byte("[[managed_file]]\npath = \".env.production\"\nrepo = \"studio/demo\"\nprefix = \"SD_\"\n")
	if err := os.WriteFile(filepath.Join(alice.Home, ".sopsdeck.toml"), manifest, 0o600); err != nil {
		return nil, nil, err
	}
	info := &demoInfo{
		Project:      alice.Home,
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
	case "get_managed_file":
		return invokeGet(req.Path, req.At)
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
	return cliErr(cmdRecipient([]string{"add", req.PublicKey, "-f", req.Path}, io.Discard, &stderr, getenv), &stderr)
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
