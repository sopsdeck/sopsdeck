package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// countReferences returns the number of times key appears as a whole word in
// body. Word boundaries treat $, {, and } as non-word characters, so KEY,
// $KEY, and ${KEY} all match while KEY inside KEYHOLDER or MY_KEY does not.
func countReferences(key string, body []byte) int {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(key) + `\b`)
	return len(re.FindAll(body, -1))
}

// scanProjectReferences walks root (excluding .git and binary files) and
// counts references to each key. managedRel marks project-relative paths to
// skip (the Managed Files themselves, where a key names its own definition).
func scanProjectReferences(root string, keys []string, managedRel map[string]bool) (map[string]int, map[string][]string, error) {
	counts := make(map[string]int, len(keys))
	for _, k := range keys {
		counts[k] = 0
	}
	files := make(map[string][]string, len(keys))
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if managedRel[rel] {
			return nil
		}
		if isBinary(path) {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, k := range keys {
			if c := countReferences(k, body); c > 0 {
				counts[k] += c
				files[k] = append(files[k], rel)
			}
		}
		return nil
	})
	return counts, files, err
}

// isBinary reports whether path looks like a non-text file, by extension
// or by a NUL byte in the first 512 bytes.
func isBinary(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".icns", ".pdf",
		".zip", ".gz", ".tar", ".webm", ".mp4", ".mov", ".mp3", ".wav", ".lock":
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
	}
	return false
}
