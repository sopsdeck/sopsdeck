package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	cloudKey    = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	privatePEM  = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)
	commonToken = regexp.MustCompile(`(?:ghp|gho|github_pat|sk_live)_[A-Za-z0-9_]{8,}`)
	testToken   = regexp.MustCompile(`sk_test_[A-Za-z0-9_]+`)
	encBlob     = regexp.MustCompile(`ENC\[[^\]]*\]`)
)

const scanHook = "#!/bin/sh\nset -eu\nsopsdeck scan\n"

func cmdScan(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "--install" {
		return installScanHook(stdout, stderr)
	}
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: sopsdeck scan [--install]")
		return 1
	}
	names, err := stagedNames()
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	blocked := 0
	allow := scanAllowlist()
	for _, name := range names {
		if _, ok := allow[filepath.ToSlash(name)]; ok {
			continue
		}
		raw, err := stagedBlob(name)
		if err != nil {
			fmt.Fprintf(stderr, "scan: %v\n", err)
			return 1
		}
		body := scanBody(raw)
		switch {
		case cloudKey.Match(body):
			fmt.Fprintf(stderr, "scan: block cloud key in %s\n", name)
			blocked++
		case privatePEM.Match(body):
			fmt.Fprintf(stderr, "scan: block private key in %s\n", name)
			blocked++
		case commonToken.Match(body):
			fmt.Fprintf(stderr, "scan: block token in %s\n", name)
			blocked++
		case testToken.Match(body):
			fmt.Fprintf(stderr, "scan: warn token in %s\n", name)
		}
	}
	_ = stdout
	if blocked > 0 {
		return 1
	}
	return 0
}

func scanBody(raw []byte) []byte {
	s := encBlob.ReplaceAllString(string(raw), "")
	var keep []string
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "sops_") || trim == "sops:" || strings.HasPrefix(trim, `"sops"`) {
			continue
		}
		keep = append(keep, line)
	}
	return []byte(strings.Join(keep, "\n"))
}

func stagedNames() ([]string, error) {
	out, err := exec.Command("git", "diff", "--cached", "--name-only", "-z").Output()
	if err != nil {
		return nil, fmt.Errorf("not a git project")
	}
	if len(out) == 0 {
		return nil, nil
	}
	parts := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	var names []string
	for _, p := range parts {
		if p != "" {
			names = append(names, p)
		}
	}
	return names, nil
}

func stagedBlob(name string) ([]byte, error) {
	return exec.Command("git", "show", ":"+name).Output()
}

func scanAllowlist() map[string]struct{} {
	_, path := findManifest(".")
	if path == "" {
		return nil
	}
	m, err := loadManifest(path)
	if err != nil {
		return nil
	}
	out := map[string]struct{}{}
	for _, rel := range m.Scan.Allowlist {
		out[filepath.ToSlash(rel)] = struct{}{}
	}
	return out
}

func installScanHook(stdout, stderr io.Writer) int {
	top, err := gitTopLevel(".")
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	hookDir := filepath.Join(top, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(hookDir, "pre-commit"), []byte(scanHook), 0o755); err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	if err := recordScanHook(top); err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "scan hook installed")
	return 0
}

func recordScanHook(root string) error {
	path := filepath.Join(root, ".sopsdeck.toml")
	m, err := loadManifest(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	m.Scan.Hook = true
	return writeManifest(path, m)
}
