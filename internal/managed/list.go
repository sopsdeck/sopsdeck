package managed

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// File is a Managed File discovered in a Project folder.
type File struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Rel  string `json:"rel"`
}

// List returns regular files in root that are safe to inspect as Project files.
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
		if d.IsDir() {
			if SkipDirectory(path, root) {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || isProjectMetadata(path, root) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
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

// Candidates returns regular files that can be imported into a Project.
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
			if SkipDirectory(path, root) {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if isProjectMetadata(path, root) {
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

func isProjectMetadata(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel == ".sopsdeck.toml"
}

// SkipDirectory keeps discovery and reference edits inside one Project.
// Ordinary monorepo folders are included; nested repos/projects are not.
func SkipDirectory(path, root string) bool {
	if path == root {
		return false
	}
	for _, marker := range []string{".git", ".sopsdeck.toml"} {
		if _, err := os.Stat(filepath.Join(path, marker)); err == nil {
			return true
		}
	}
	return skipDir(filepath.Base(path))
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
