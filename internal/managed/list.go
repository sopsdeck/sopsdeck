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
		if !isDotenvName(name) && (!isStructuredName(name) || !looksSOPS(path)) {
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

func skipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "target", "dist", "vendor", ".scratch":
		return true
	default:
		return false
	}
}

func isDotenvName(name string) bool {
	return name == ".env" || strings.HasPrefix(name, ".env.") || strings.HasSuffix(strings.ToLower(name), ".env")
}

func isStructuredName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
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
	return strings.Contains(sample, `"sops"`) || strings.Contains(sample, "sops:")
}
