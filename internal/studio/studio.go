package studio

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"filippo.io/age"
	"sopsdeck/internal/githubfake"
)

// Studio is a local collaboration world: one bare origin, throwaway Age
// identities, and a fake GitHub Actions secrets API. No extra machines or
// GitHub accounts.
type Studio struct {
	Root   string
	Origin string
	GitHub *githubfake.Server

	cwdMu sync.Mutex
}

type User struct {
	Name       string
	Email      string
	Home       string
	ConfigHome string
	StateDir   string
	AgeFile    string
	PublicKey  string
	studio     *Studio
}

func New(root string) (*Studio, error) {
	origin := filepath.Join(root, "origin.git")
	if err := os.MkdirAll(origin, 0o700); err != nil {
		return nil, err
	}
	if err := git(origin, "init", "--bare"); err != nil {
		return nil, err
	}
	if err := git(origin, "symbolic-ref", "HEAD", "refs/heads/main"); err != nil {
		return nil, err
	}
	return &Studio{
		Root:   root,
		Origin: origin,
		GitHub: githubfake.New(),
	}, nil
}

// Open reuses an existing studio root, or creates one.
func Open(root string) (*Studio, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	origin := filepath.Join(root, "origin.git")
	if _, err := os.Stat(filepath.Join(origin, "HEAD")); err == nil {
		return &Studio{
			Root:   root,
			Origin: origin,
			GitHub: githubfake.New(),
		}, nil
	}
	return New(root)
}

const DefaultProject = "checkout"

// Prepare creates Alice and Bob with isolated Git identities and a shared
// checkout clone in each home. It does not write the host Git identity.
func Prepare(root string) (*Studio, *User, *User, error) {
	s, err := Open(root)
	if err != nil {
		return nil, nil, nil, err
	}
	alice, err := s.Identity("alice", "alice@sopsdeck.example")
	if err != nil {
		s.Close()
		return nil, nil, nil, err
	}
	bob, err := s.Identity("bob", "bob@sopsdeck.example")
	if err != nil {
		s.Close()
		return nil, nil, nil, err
	}
	aliceDir, bobDir, err := s.SharedProject(DefaultProject)
	if err != nil {
		s.Close()
		return nil, nil, nil, err
	}
	alice.Home = aliceDir
	bob.Home = bobDir
	if err := writeTeamFiles(s, alice, bob); err != nil {
		s.Close()
		return nil, nil, nil, err
	}
	return s, alice, bob, nil
}

func (s *Studio) Close() {
	if s.GitHub != nil {
		s.GitHub.Close()
	}
}

// User creates a worktree with its own Age identity and a remote named origin.
func (s *Studio) User(name, email string) (*User, error) {
	u, err := s.Identity(name, email)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(u.Home, ".git")); err == nil {
		if err := git(u.Home, "config", "user.email", email); err != nil {
			return nil, err
		}
		if err := git(u.Home, "config", "user.name", name); err != nil {
			return nil, err
		}
		if err := ensureRemote(u.Home, s.Origin); err != nil {
			return nil, err
		}
		return u, nil
	}
	if err := os.MkdirAll(u.Home, 0o700); err != nil {
		return nil, err
	}
	if err := git(u.Home, "init", "-b", "main"); err != nil {
		return nil, err
	}
	if err := git(u.Home, "config", "user.email", email); err != nil {
		return nil, err
	}
	if err := git(u.Home, "config", "user.name", name); err != nil {
		return nil, err
	}
	if err := git(u.Home, "remote", "add", "origin", s.Origin); err != nil {
		return nil, err
	}
	return u, nil
}

// Clone clones origin into a new User worktree with its own Age identity.
func (s *Studio) Clone(name, email string) (*User, error) {
	u, err := s.Identity(name, email)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(u.Home, ".git")); err == nil {
		if err := git(u.Home, "config", "user.email", email); err != nil {
			return nil, err
		}
		if err := git(u.Home, "config", "user.name", name); err != nil {
			return nil, err
		}
		return u, nil
	}
	parent := filepath.Dir(u.Home)
	base := filepath.Base(u.Home)
	if err := os.RemoveAll(u.Home); err != nil {
		return nil, err
	}
	cmd := exec.Command("git", "clone", s.Origin, base)
	cmd.Dir = parent
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone: %s", out)
	}
	if err := git(u.Home, "config", "user.email", email); err != nil {
		return nil, err
	}
	if err := git(u.Home, "config", "user.name", name); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Studio) Identity(name, email string) (*User, error) {
	home := filepath.Join(s.Root, name)
	state := filepath.Join(s.Root, name+"-state")
	if err := os.MkdirAll(state, 0o700); err != nil {
		return nil, err
	}
	u := &User{
		Name:       name,
		Email:      email,
		Home:       home,
		ConfigHome: filepath.Join(s.Root, name+"-home"),
		StateDir:   state,
		AgeFile:    filepath.Join(state, "age.txt"),
		studio:     s,
	}
	if err := u.writeGitConfig(); err != nil {
		return nil, err
	}
	if _, err := os.Stat(u.AgeFile); err == nil {
		pub, err := publicKeyFromFile(u.AgeFile)
		if err != nil {
			return nil, err
		}
		u.PublicKey = pub
		return u, nil
	}
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, err
	}
	body := "# public key: " + id.Recipient().String() + "\n" + id.String() + "\n"
	if err := os.WriteFile(u.AgeFile, []byte(body), 0o600); err != nil {
		return nil, err
	}
	u.PublicKey = id.Recipient().String()
	return u, nil
}

// SharedProject clones NAME into each User's home so both windows open the
// same Project under different Git identities.
func (s *Studio) SharedProject(name string) (string, string, error) {
	if err := validateProjectName(name); err != nil {
		return "", "", err
	}
	alice, err := s.Identity("alice", "alice@sopsdeck.example")
	if err != nil {
		return "", "", err
	}
	bob, err := s.Identity("bob", "bob@sopsdeck.example")
	if err != nil {
		return "", "", err
	}
	origin := filepath.Join(s.Root, "origins", name+".git")
	if _, err := os.Stat(filepath.Join(origin, "HEAD")); err != nil {
		if err := os.MkdirAll(origin, 0o700); err != nil {
			return "", "", err
		}
		if err := git(origin, "init", "--bare"); err != nil {
			return "", "", err
		}
		if err := git(origin, "symbolic-ref", "HEAD", "refs/heads/main"); err != nil {
			return "", "", err
		}
	}
	aliceDir := filepath.Join(alice.ConfigHome, name)
	bobDir := filepath.Join(bob.ConfigHome, name)
	if err := initSharedWorktree(aliceDir, origin, "alice", "alice@sopsdeck.example"); err != nil {
		return "", "", err
	}
	if err := cloneSharedWorktree(bobDir, origin, "bob", "bob@sopsdeck.example"); err != nil {
		return "", "", err
	}
	return aliceDir, bobDir, nil
}

func publicKeyFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(line, "# public key: "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("no public key comment in %s", path)
}

func (u *User) Getenv(key string) string {
	switch key {
	case "HOME":
		if u.ConfigHome != "" {
			return u.ConfigHome
		}
	case "GIT_CONFIG_GLOBAL":
		if u.ConfigHome != "" {
			return filepath.Join(u.ConfigHome, ".gitconfig")
		}
	case "SOPSDECK_STATE_DIR":
		return u.StateDir
	case "SOPS_AGE_KEY_FILE":
		return u.AgeFile
	case "SOPSDECK_TEAM_ROOT":
		if u.studio != nil {
			return u.studio.Root
		}
	case "SOPSDECK_GITHUB_API":
		if u.studio != nil && u.studio.GitHub != nil {
			return u.studio.GitHub.URL()
		}
	case "SOPSDECK_GITHUB_REPO":
		return "studio/demo"
	}
	return os.Getenv(key)
}

// WithWorld overlays this User's env and home directory while fn runs.
// SOPS decrypt reads process env, so callers that invoke the CLI must use this.
func (u *User) WithWorld(fn func()) {
	u.studio.cwdMu.Lock()
	defer u.studio.cwdMu.Unlock()
	wd, err := os.Getwd()
	if err != nil {
		fn()
		return
	}
	vars := map[string]string{
		"SOPSDECK_STATE_DIR":   u.StateDir,
		"SOPS_AGE_KEY_FILE":    u.AgeFile,
		"SOPSDECK_GITHUB_API":  u.Getenv("SOPSDECK_GITHUB_API"),
		"SOPSDECK_GITHUB_REPO": u.Getenv("SOPSDECK_GITHUB_REPO"),
	}
	if u.ConfigHome != "" {
		vars["HOME"] = u.ConfigHome
		vars["GIT_CONFIG_GLOBAL"] = filepath.Join(u.ConfigHome, ".gitconfig")
	}
	restore := overlayEnv(vars)
	defer restore()
	if err := os.Chdir(u.Home); err != nil {
		fn()
		return
	}
	defer func() { _ = os.Chdir(wd) }()
	fn()
}

func overlayEnv(vars map[string]string) func() {
	prev := map[string]*string{}
	for key, value := range vars {
		if cur, ok := os.LookupEnv(key); ok {
			c := cur
			prev[key] = &c
		} else {
			prev[key] = nil
		}
		_ = os.Setenv(key, value)
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

func (u *User) Git(args ...string) (string, error) {
	return gitOut(u.Home, args...)
}

func (u *User) writeGitConfig() error {
	if err := os.MkdirAll(u.ConfigHome, 0o700); err != nil {
		return err
	}
	file := filepath.Join(u.ConfigHome, ".gitconfig")
	if err := gitConfigFile(file, "user.name", u.Name); err != nil {
		return err
	}
	if err := gitConfigFile(file, "user.email", u.Email); err != nil {
		return err
	}
	readme := filepath.Join(u.ConfigHome, "README")
	body := u.Name + `'s isolated HOME. Shared Projects are Git clones in this folder.
Source ` + filepath.Join(u.studio.Root, u.Name+".env") + ` before running sopsdeck here.
Do not open the other person's clones in this window.
`
	return os.WriteFile(readme, []byte(body), 0o600)
}

func writeTeamFiles(s *Studio, alice, bob *User) error {
	if err := writeEnvFile(s.Root, alice); err != nil {
		return err
	}
	if err := writeEnvFile(s.Root, bob); err != nil {
		return err
	}
	body := "studio=" + s.Root + "\n" +
		"alice_project=" + alice.Home + "\n" +
		"alice_home=" + alice.ConfigHome + "\n" +
		"alice_env=" + filepath.Join(s.Root, "alice.env") + "\n" +
		"bob_project=" + bob.Home + "\n" +
		"bob_home=" + bob.ConfigHome + "\n" +
		"bob_env=" + filepath.Join(s.Root, "bob.env") + "\n" +
		"alice_public_key=" + alice.PublicKey + "\n" +
		"bob_public_key=" + bob.PublicKey + "\n"
	return os.WriteFile(filepath.Join(s.Root, "paths.txt"), []byte(body), 0o600)
}

func writeEnvFile(root string, u *User) error {
	gitconfig := filepath.Join(u.ConfigHome, ".gitconfig")
	body := "# " + u.Name + " — isolated Sopsdeck identity. Do not open this Project in the other window.\n" +
		"export HOME=" + shellQuote(u.ConfigHome) + "\n" +
		"export GIT_CONFIG_GLOBAL=" + shellQuote(gitconfig) + "\n" +
		"export SOPSDECK_STATE_DIR=" + shellQuote(u.StateDir) + "\n" +
		"export SOPS_AGE_KEY_FILE=" + shellQuote(u.AgeFile) + "\n" +
		"export SOPSDECK_GITHUB_REPO=studio/demo\n" +
		"export SOPSDECK_TEAM_ROOT=" + shellQuote(root) + "\n"
	return os.WriteFile(filepath.Join(root, u.Name+".env"), []byte(body), 0o600)
}

func initSharedWorktree(dir, origin, name, email string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		if err := git(dir, "config", "user.name", name); err != nil {
			return err
		}
		if err := git(dir, "config", "user.email", email); err != nil {
			return err
		}
		return ensureRemote(dir, origin)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := git(dir, "init", "-b", "main"); err != nil {
		return err
	}
	if err := git(dir, "config", "user.name", name); err != nil {
		return err
	}
	if err := git(dir, "config", "user.email", email); err != nil {
		return err
	}
	if err := ensureRemote(dir, origin); err != nil {
		return err
	}
	if _, err := gitOut(dir, "rev-parse", "HEAD"); err == nil {
		return nil
	}
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# "+name+"\n"), 0o600); err != nil {
		return err
	}
	if err := git(dir, "add", "README.md"); err != nil {
		return err
	}
	if err := git(dir, "commit", "-m", "start"); err != nil {
		return err
	}
	return git(dir, "push", "-u", "origin", "main")
}

func cloneSharedWorktree(dir, origin, name, email string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		if err := git(dir, "config", "user.name", name); err != nil {
			return err
		}
		return git(dir, "config", "user.email", email)
	}
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", origin, filepath.Base(dir))
	cmd.Dir = parent
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone: %s", out)
	}
	if err := git(dir, "config", "user.name", name); err != nil {
		return err
	}
	return git(dir, "config", "user.email", email)
}

func ensureRemote(dir, origin string) error {
	out, err := gitOut(dir, "remote", "get-url", "origin")
	if err != nil {
		return git(dir, "remote", "add", "origin", origin)
	}
	if strings.TrimSpace(out) == origin {
		return nil
	}
	return git(dir, "remote", "set-url", "origin", origin)
}

func validateProjectName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name {
		return fmt.Errorf("project name %q is invalid", name)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("project name %q is invalid", name)
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func gitConfigFile(file, key, value string) error {
	return git("", "config", "--file", file, key, value)
}

func git(dir string, args ...string) error {
	_, err := gitOut(dir, args...)
	return err
}

func gitOut(dir string, args ...string) (string, error) {
	gitArgs := append([]string{"-c", "commit.gpgsign=false"}, args...)
	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), bytes.TrimSpace(out))
	}
	return string(out), nil
}
