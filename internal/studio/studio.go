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
	Name      string
	Email     string
	Home      string
	StateDir  string
	AgeFile   string
	PublicKey string
	studio    *Studio
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
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(state, 0o700); err != nil {
		return nil, err
	}
	u := &User{
		Name:     name,
		Email:    email,
		Home:     home,
		StateDir: state,
		AgeFile:  filepath.Join(state, "age.txt"),
		studio:   s,
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
	case "SOPSDECK_STATE_DIR":
		return u.StateDir
	case "SOPS_AGE_KEY_FILE":
		return u.AgeFile
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
	restore := overlayEnv(map[string]string{
		"SOPSDECK_STATE_DIR":   u.StateDir,
		"SOPS_AGE_KEY_FILE":    u.AgeFile,
		"SOPSDECK_GITHUB_API":  u.Getenv("SOPSDECK_GITHUB_API"),
		"SOPSDECK_GITHUB_REPO": u.Getenv("SOPSDECK_GITHUB_REPO"),
	})
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

func git(dir string, args ...string) error {
	_, err := gitOut(dir, args...)
	return err
}

func gitOut(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), bytes.TrimSpace(out))
	}
	return string(out), nil
}
