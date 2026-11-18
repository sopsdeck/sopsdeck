package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/getsops/sops/v3/cmd/sops/formats"
)

var (
	cloudKey    = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	ageSecretRE = regexp.MustCompile(`AGE-SECRET-KEY-[A-Z0-9-]+`)
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
	if len(args) == 1 && args[0] == "--uninstall" {
		return uninstallScanHook(stdout, stderr)
	}
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: sopsdeck scan [--install|--uninstall]")
		return 1
	}
	top, err := gitTopLevel(".")
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	names, err := stagedNames()
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	blocked := 0
	for _, name := range names {
		if scanAllowed(top, name) {
			continue
		}
		raw, err := stagedBlob(name)
		if err != nil {
			fmt.Fprintf(stderr, "scan: %v\n", err)
			return 1
		}
		if key, err := stagedPlaintextManagedKey(top, name, raw); err != nil {
			fmt.Fprintf(stderr, "scan: %v\n", err)
			return 1
		} else if key != "" {
			fmt.Fprintf(stderr, "scan: block plaintext managed secret %s in %s\n", key, name)
			blocked++
			continue
		}
		body := scanBody(raw)
		switch {
		case cloudKey.Match(body):
			fmt.Fprintf(stderr, "scan: block cloud key in %s\n", name)
			blocked++
		case privatePEM.Match(body):
			fmt.Fprintf(stderr, "scan: block private key in %s\n", name)
			blocked++
		case ageSecretRE.Match(body):
			fmt.Fprintf(stderr, "scan: block Age identity in %s\n", name)
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
	out, err := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=ACMRT", "-z").Output()
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

func stagedPlaintextManagedKey(top, name string, raw []byte) (string, error) {
	file := filepath.Join(top, filepath.FromSlash(name))
	mapping, _, _ := mappingFor(file)
	if mapping.Path == "" || isEncryptedBytes(raw) {
		return "", nil
	}
	format := formats.FormatFromString(mapping.Format)
	if mapping.Format == "" {
		format = fileFormat(file)
	}
	var pairs map[string]string
	var err error
	if format == formats.Dotenv {
		pairs, err = parseDotenvMap(raw)
	} else {
		pairs, err = plainPairs(raw, format, nil)
	}
	if err != nil {
		return "", fmt.Errorf("could not inspect managed file %s", name)
	}
	for key, value := range pairs {
		if value == "" || publicRedactionPath(key, mapping.PublicKeys, format) {
			continue
		}
		if len(mapping.EncryptedKeys) > 0 && !pairEncrypted(key, mapping.EncryptedKeys, false, "") {
			continue
		}
		return key, nil
	}
	return "", nil
}

func stagedBlob(name string) ([]byte, error) {
	return exec.Command("git", "show", ":"+name).Output()
}

func scanAllowed(top, name string) bool {
	file := filepath.Join(top, filepath.FromSlash(name))
	root, path := findManifest(file)
	if path == "" {
		return false
	}
	m, err := loadManifest(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return false
	}
	for _, allowed := range m.Scan.Allowlist {
		if filepath.ToSlash(allowed) == filepath.ToSlash(rel) {
			return true
		}
	}
	return false
}

func installScanHook(stdout, stderr io.Writer) int {
	top, err := gitTopLevel(".")
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	hookDir, err := gitHooksPath(top)
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	hook := filepath.Join(hookDir, "pre-commit")
	created := false
	if info, err := os.Lstat(hook); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			fmt.Fprintf(stderr, "scan: existing pre-commit hook preserved; add `sopsdeck scan` to %s manually\n", hook)
			return 1
		}
		existing, readErr := os.ReadFile(hook)
		if readErr != nil {
			fmt.Fprintf(stderr, "scan: %v\n", readErr)
			return 1
		}
		if string(existing) != scanHook {
			fmt.Fprintf(stderr, "scan: existing pre-commit hook preserved; add `sopsdeck scan` to %s manually\n", hook)
			return 1
		}
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	} else if err := createScanHook(hook); err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	} else {
		created = true
	}
	projectRoot, manifestPath := findManifest(".")
	if manifestPath == "" {
		projectRoot = top
	}
	if err := recordScanHook(projectRoot, true); err != nil {
		if created {
			_ = os.Remove(hook)
		}
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "scan hook installed")
	return 0
}

func uninstallScanHook(stdout, stderr io.Writer) int {
	top, err := gitTopLevel(".")
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	hookDir, err := gitHooksPath(top)
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	hook := filepath.Join(hookDir, "pre-commit")
	if info, err := os.Lstat(hook); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			fmt.Fprintln(stderr, "scan: existing pre-commit hook preserved; remove it manually if appropriate")
			return 1
		}
		body, readErr := os.ReadFile(hook)
		if readErr != nil {
			fmt.Fprintf(stderr, "scan: %v\n", readErr)
			return 1
		}
		if string(body) != scanHook {
			fmt.Fprintln(stderr, "scan: modified pre-commit hook preserved; remove the Sopsdeck command manually")
			return 1
		}
		if err := os.Remove(hook); err != nil {
			fmt.Fprintf(stderr, "scan: %v\n", err)
			return 1
		}
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 1
	}
	projectRoot, manifestPath := findManifest(".")
	if manifestPath != "" {
		if err := recordScanHook(projectRoot, false); err != nil {
			fmt.Fprintf(stderr, "scan: %v\n", err)
			return 1
		}
	}
	fmt.Fprintln(stdout, "scan hook uninstalled")
	return 0
}

func gitHooksPath(top string) (string, error) {
	config := exec.Command("git", "config", "--local", "--path", "--get", "core.hooksPath")
	config.Dir = top
	if out, err := config.Output(); err == nil {
		path := strings.TrimSpace(string(out))
		if !filepath.IsAbs(path) {
			path = filepath.Join(top, path)
		}
		return filepath.Clean(path), nil
	}
	global := exec.Command("git", "config", "--path", "--get", "core.hooksPath")
	global.Dir = top
	if _, err := global.Output(); err == nil {
		return "", fmt.Errorf("core.hooksPath is configured outside this Project; add `sopsdeck scan` to that pre-commit hook manually")
	}
	cmd := exec.Command("git", "rev-parse", "--git-path", "hooks")
	cmd.Dir = top
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not resolve Git hooks path")
	}
	path := strings.TrimSpace(string(out))
	if !filepath.IsAbs(path) {
		path = filepath.Join(top, path)
	}
	return filepath.Clean(path), nil
}

func createScanHook(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("pre-commit hook appeared during install; it was preserved")
		}
		return err
	}
	if _, err = file.WriteString(scanHook); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
	}
	return err
}

func recordScanHook(root string, installed bool) error {
	path := filepath.Join(root, ".sopsdeck.toml")
	m, err := loadManifest(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	m.Scan.Hook = installed
	return writeManifest(path, m)
}
