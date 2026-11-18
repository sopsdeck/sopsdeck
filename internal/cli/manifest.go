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
	LegacyOwner []manifestRecipient `toml:"owner,omitempty"`
	Scan        scanPolicy          `toml:"scan"`
}

type manifestRecipient struct {
	Key   string `toml:"key" json:"key"`
	Name  string `toml:"name" json:"name"`
	Email string `toml:"email,omitempty" json:"email,omitempty"`
	Kind  string `toml:"kind,omitempty" json:"kind,omitempty"`
}

type scanPolicy struct {
	Hook      bool     `toml:"hook,omitempty"`
	Allowlist []string `toml:"allowlist,omitempty"`
}

type manifestFile struct {
	Path          string   `toml:"path"`
	Format        string   `toml:"format,omitempty"`
	EncryptedKeys []string `toml:"encrypted_keys,omitempty"`
	PublicKeys    []string `toml:"public_keys,omitempty"`
	Recipients    []string `toml:"recipients,omitempty"`
	Repo          string   `toml:"repo,omitempty"`
	Org           string   `toml:"org,omitempty"`
	Scope         string   `toml:"scope,omitempty"`
	Environment   string   `toml:"environment,omitempty"`
	Visibility    string   `toml:"visibility,omitempty"`
	Prefix        string   `toml:"prefix,omitempty"`
	Keys          []string `toml:"keys,omitempty"`
	Synced        []string `toml:"synced,omitempty"`
}

func findManifest(start string) (root, path string) {
	start, err := filepath.Abs(start)
	if err != nil {
		return "", ""
	}
	if canonical, err := filepath.EvalSymlinks(start); err == nil {
		start = canonical
	}
	dir := start
	if info, err := os.Stat(start); err == nil && !info.IsDir() {
		dir = filepath.Dir(start)
	}
	for {
		cand := filepath.Join(dir, ".sopsdeck.toml")
		if _, err := os.Stat(cand); err == nil {
			return dir, cand
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return "", ""
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
	// Keep names from older manifests as recipient labels, without owner roles.
	labels := identityLabels(m)
	for _, owner := range m.LegacyOwner {
		found := false
		for _, recipient := range m.Recipient {
			if strings.EqualFold(owner.Key, recipient.Key) {
				found = true
				break
			}
		}
		if !found {
			m.Recipient = append(m.Recipient, owner)
		}
	}
	for i, recipient := range m.Recipient {
		m.Recipient[i] = labels[strings.ToLower(recipient.Key)]
	}
	m.LegacyOwner = nil
	return m, nil
}

func mappingFor(file string) (manifestFile, string, string) {
	file, _ = filepath.Abs(file)
	if canonical, err := filepath.EvalSymlinks(file); err == nil {
		file = canonical
	}
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
	return writeAtomic(path, raw)
}

func checkFileManifest(file string) error {
	_, path := findManifest(file)
	if path == "" {
		return nil
	}
	_, err := loadManifest(path)
	return err
}

func setRecipientLabel(file, key, name, kind, email string) error {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" && email == "" {
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
			m.Recipient[i].Email = email
			m.Recipient[i].Kind = kind
			return writeManifest(manifestPath, m)
		}
	}
	m.Recipient = append(m.Recipient, manifestRecipient{Key: key, Name: name, Email: email, Kind: kind})
	return writeManifest(manifestPath, m)
}

func configureSyncTarget(file, scope, repo, org, environment, prefix, visibility string) error {
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

func recipientsFromEnv(root string, getenv func(string) string) []manifestRecipient {
	if getenv == nil {
		getenv = os.Getenv
	}
	key, err := ageRecipientFromEnv(getenv)
	if err != nil || strings.TrimSpace(key) == "" {
		return nil
	}
	name, email := gitIdentity(root)
	return []manifestRecipient{{Key: key, Name: name, Email: email, Kind: "person"}}
}

func setSynced(path, rel string, names []string) error {
	m, err := loadManifest(path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	for i := range m.ManagedFile {
		if filepath.ToSlash(m.ManagedFile[i].Path) == rel {
			m.ManagedFile[i].Synced = names
			return writeManifest(path, m)
		}
	}
	return nil
}
