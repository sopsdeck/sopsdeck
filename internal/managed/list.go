package managed

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// File is a Managed File discovered in a Project folder.
type File struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Rel  string `json:"rel"`
}

// List returns dotenv files and SOPS-looking JSON/YAML under root.
func List(root string) ([]File, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fs.ErrInvalid
	}
	var out []File
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if skipDir(name) && path != root {
				return fs.SkipDir
			}
			return nil
		}
		if !isDotenvOrSOPS(name, path) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		out = append(out, File{Name: name, Path: path, Rel: rel})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, nil
}

// Candidates returns supported files that can be imported into a Project.
func Candidates(root string) ([]File, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fs.ErrInvalid
	}
	var out []File
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir(d.Name()) && path != root {
				return fs.SkipDir
			}
			return nil
		}
		if isLockfileName(d.Name()) {
			return nil
		}
		if !isDotenvName(d.Name()) && !isStructuredName(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, File{Name: d.Name(), Path: path, Rel: rel})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, nil
}

func skipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "target", "dist", "vendor", ".scratch",
		".next", ".nuxt", ".svelte-kit", ".turbo", ".gradle",
		"build", "out", "coverage", ".cache", "__pycache__",
		".parcel-cache", ".pnpm-store", ".venv", ".tox",
		".mypy_cache", ".pytest_cache", ".dart_tool":
		return true
	default:
		return false
	}
}

func isDotenvOrSOPS(name, path string) bool {
	if isDotenvName(name) {
		// Per issue 05, a Managed File already has SOPS metadata; a plain
		// dotenv is not managed until it is encrypted.
		return looksSOPSDotenv(path)
	}
	return isStructuredName(name) && looksSOPS(path)
}

func isDotenvName(name string) bool {
	return name == ".env" || strings.HasPrefix(name, ".env.") || strings.HasSuffix(strings.ToLower(name), ".env")
}

func isStructuredName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

// isLockfileName reports whether name is a generated dependency lockfile.
// Lockfiles are machine-generated and never hold secrets worth managing as
// a Managed File, so they are excluded from candidates. Their structure
// (e.g. package-lock.json's empty-string root key under "packages") also
// breaks the dotted key-path tree.
func isLockfileName(name string) bool {
	switch name {
	case "package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml",
		"yarn.lock", "bun.lock", "bun.lockb", "composer.lock",
		"Cargo.lock", "poetry.lock", "Pipfile.lock", "mix.lock",
		"Gemfile.lock", "packages.lock.json", "nuget.lock.json":
		return true
	}
	return false
}

func looksSOPS(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if len(data) > 16_384 {
		data = data[:16_384]
	}
	sample := string(data)
	return strings.Contains(sample, "ENC[") ||
		(strings.Contains(sample, `"sops"`) && strings.Contains(sample, `"sops": {`)) ||
		strings.Contains(sample, "sops:\n")
}
