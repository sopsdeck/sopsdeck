package cli

import (
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

type projectManifest struct {
	ManagedFile []manifestFile `toml:"managed_file"`
}

type manifestFile struct {
	Path        string   `toml:"path"`
	Repo        string   `toml:"repo,omitempty"`
	Environment string   `toml:"environment,omitempty"`
	Prefix      string   `toml:"prefix,omitempty"`
	Keys        []string `toml:"keys,omitempty"`
	Published   []string `toml:"published,omitempty"`
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
