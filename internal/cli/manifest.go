package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

type projectManifest struct {
	ManagedFile []manifestFile      `toml:"managed_file"`
	Recipient   []manifestRecipient `toml:"recipient,omitempty"`
	Scan        scanPolicy          `toml:"scan"`
}

type manifestRecipient struct {
	Key  string `toml:"key"`
	Name string `toml:"name"`
	Kind string `toml:"kind,omitempty"`
}

type scanPolicy struct {
	Hook      bool     `toml:"hook,omitempty"`
	Allowlist []string `toml:"allowlist,omitempty"`
}

type manifestFile struct {
	Path          string   `toml:"path"`
	EncryptedKeys []string `toml:"encrypted_keys,omitempty"`
	Repo          string   `toml:"repo,omitempty"`
	Org           string   `toml:"org,omitempty"`
	Scope         string   `toml:"scope,omitempty"`
	Environment   string   `toml:"environment,omitempty"`
	Visibility    string   `toml:"visibility,omitempty"`
	Prefix        string   `toml:"prefix,omitempty"`
	Keys          []string `toml:"keys,omitempty"`
	Published     []string `toml:"published,omitempty"`
}

func findManifest(start string) (root, path string) {
	dir := start
	if info, err := os.Stat(start); err == nil && !info.IsDir() {
		dir = filepath.Dir(start)
	}
	for {
		cand := filepath.Join(dir, ".sopsdeck.toml")
		if _, err := os.Stat(cand); err == nil {
			return dir, cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ""
		}
		dir = parent
	}
}

func loadManifest(path string) (projectManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return projectManifest{}, err
	}
	var m projectManifest
	if err := toml.Unmarshal(raw, &m); err != nil {
		return projectManifest{}, err
	}
	return m, nil
}

func mappingFor(file string) (manifestFile, string, string) {
	root, path := findManifest(file)
	if path == "" {
		return manifestFile{}, "", ""
	}
	m, err := loadManifest(path)
	if err != nil {
		return manifestFile{}, root, path
	}
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return manifestFile{}, root, path
	}
	rel = filepath.ToSlash(rel)
	for _, entry := range m.ManagedFile {
		if filepath.ToSlash(entry.Path) == rel {
			return entry, root, path
		}
	}
	return manifestFile{}, root, path
}

func writeManifest(path string, m projectManifest) error {
	raw, err := toml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func setRecipientLabel(file, key, name, kind string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	_, manifestPath := findManifest(file)
	if manifestPath == "" {
		return nil
	}
	m, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	for i := range m.Recipient {
		if strings.EqualFold(m.Recipient[i].Key, key) {
			m.Recipient[i].Name = name
			m.Recipient[i].Kind = kind
			return writeManifest(manifestPath, m)
		}
	}
	m.Recipient = append(m.Recipient, manifestRecipient{Key: key, Name: name, Kind: kind})
	return writeManifest(manifestPath, m)
}

func configureIntegration(file, scope, repo, org, environment, prefix, visibility string) error {
	root, manifestPath := findManifest(file)
	if manifestPath == "" {
		return fmt.Errorf("project is not initialized")
	}
	m, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	for i := range m.ManagedFile {
		if filepath.ToSlash(m.ManagedFile[i].Path) != rel {
			continue
		}
		m.ManagedFile[i].Scope = scope
		m.ManagedFile[i].Repo = repo
		m.ManagedFile[i].Org = org
		m.ManagedFile[i].Environment = environment
		m.ManagedFile[i].Prefix = prefix
		m.ManagedFile[i].Visibility = visibility
		return writeManifest(manifestPath, m)
	}
	return fmt.Errorf("file is not managed")
}

func setPublished(path, rel string, names []string) error {
	m, err := loadManifest(path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	for i := range m.ManagedFile {
		if filepath.ToSlash(m.ManagedFile[i].Path) == rel {
			m.ManagedFile[i].Published = names
			return writeManifest(path, m)
		}
	}
	return nil
}
