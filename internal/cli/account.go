package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type accountIdentity struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	PublicKey   string `json:"public_key"`
	HasIdentity bool   `json:"has_identity"`
}

func cmdAccount(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: sopsdeck account show -f PATH | configure NAME EMAIL [-f PATH]")
		return 1
	}
	switch args[0] {
	case "show":
		path := ""
		for i := 1; i < len(args); i++ {
			if args[i] != "-f" || i+1 >= len(args) {
				fmt.Fprintln(stderr, "usage: sopsdeck account show -f PATH")
				return 1
			}
			path = args[i+1]
			i++
		}
		if err := json.NewEncoder(stdout).Encode(accountForPath(path, getenv)); err != nil {
			fmt.Fprintf(stderr, "account show: %v\n", err)
			return 1
		}
		return 0
	case "configure":
		if len(args) < 3 || strings.TrimSpace(args[1]) == "" || strings.TrimSpace(args[2]) == "" {
			fmt.Fprintln(stderr, "usage: sopsdeck account configure NAME EMAIL [-f PATH]")
			return 1
		}
		path := ""
		for i := 3; i < len(args); i++ {
			if args[i] != "-f" || i+1 >= len(args) {
				fmt.Fprintln(stderr, "usage: sopsdeck account configure NAME EMAIL [-f PATH]")
				return 1
			}
			path = args[i+1]
			i++
		}
		if err := configureGitIdentity(path, args[1], args[2]); err != nil {
			fmt.Fprintf(stderr, "account configure: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "usage: sopsdeck account show -f PATH | configure NAME EMAIL [-f PATH]")
		return 1
	}
}

func accountForPath(path string, getenv func(string) string) accountIdentity {
	name, email := gitIdentity(path)
	publicKey, err := ageRecipientFromEnv(getenv)
	return accountIdentity{
		Name:        name,
		Email:       email,
		PublicKey:   publicKey,
		HasIdentity: err == nil && publicKey != "",
	}
}

func gitIdentity(path string) (string, string) {
	return gitConfig(path, "user.name"), gitConfig(path, "user.email")
}

func gitConfig(path, key string) string {
	if path == "" {
		return gitConfigGlobal(key)
	}
	args := []string{"config", "--get", key}
	dir := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		dir = filepath.Dir(path)
	}
	args = append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", args...).Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return gitConfigGlobal(key)
}

func gitConfigGlobal(key string) string {
	out, err := exec.Command("git", "config", "--global", "--get", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func configureGitIdentity(path, name, email string) error {
	if path == "" {
		path = "."
	}
	currentName, currentEmail := gitIdentity(path)
	fields := []struct {
		key        string
		want       string
		configured string
	}{
		{key: "user.name", want: name, configured: currentName},
		{key: "user.email", want: email, configured: currentEmail},
	}
	for _, current := range fields {
		if current.configured != "" {
			if current.configured != strings.TrimSpace(current.want) {
				return fmt.Errorf("%s is already configured as %q", current.key, current.configured)
			}
		}
	}
	for _, current := range fields {
		if current.configured != "" {
			continue
		}
		if out, err := exec.Command("git", "config", "--global", current.key, strings.TrimSpace(current.want)).CombinedOutput(); err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = err.Error()
			}
			return fmt.Errorf("git config %s: %s", current.key, msg)
		}
	}
	return nil
}

func invokeCreateUserIdentity(path string, getenv func(string) string) (any, error) {
	var stdout, stderr strings.Builder
	if code := identityCreate([]string{"--confirmed-backup"}, &stdout, &stderr, getenv); code != 0 {
		return nil, fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}
	return accountForPath(path, getenv), nil
}
