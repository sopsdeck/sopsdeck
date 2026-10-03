package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/cmd/sops/formats"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/decrypt"
	"sopsdeck/internal/managed"
)

type projectFile struct {
	managed.File
	Managed bool `json:"managed"`
}

type projectState struct {
	Path        string             `json:"path"`
	GitRoot     string             `json:"git_root"`
	Initialized bool               `json:"initialized"`
	Managed     []projectFile      `json:"managed"`
	Candidates  []projectCandidate `json:"candidates"`
	Warnings    []projectWarning   `json:"warnings,omitempty"`
}

type projectWarning struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type projectCandidate struct {
	managed.File
	Keys []string `json:"keys,omitempty"`
}

type projectSelection struct {
	Path string   `json:"path"`
	Keys []string `json:"keys"`
}

const projectUsage = "usage: sopsdeck project files FOLDER | init FOLDER [--file PATH]... | add FOLDER --file PATH | remove FOLDER --file PATH | encrypt FILE --keys PATH,..."

const neverMatchRegex = `[^\s\S]`

func cmdProject(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, projectUsage)
		return 1
	}
	switch args[0] {
	case "files":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "usage: sopsdeck project files FOLDER")
			return 1
		}
		state, err := inspectProject(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "project files: %v\n", err)
			return 1
		}
		if err := json.NewEncoder(stdout).Encode(state); err != nil {
			fmt.Fprintf(stderr, "project files: %v\n", err)
			return 1
		}
		return 0
	case "init":
		return initProject(args[1:], stdout, stderr, getenv)
	case "add":
		return addProjectFile(args[1:], stdout, stderr, getenv)
	case "remove":
		return removeProjectFile(args[1:], stdout, stderr)
	case "encrypt":
		return encryptProjectKeys(args[1:], stdout, stderr, getenv)
	default:
		fmt.Fprintln(stderr, projectUsage)
		return 1
	}
}

func addProjectFile(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) != 3 && len(args) != 5 || args[1] != "--file" || (len(args) == 5 && args[3] != "--keys") {
		fmt.Fprintln(stderr, "usage: sopsdeck project add FOLDER --file PATH [--keys PATH,...]")
		return 1
	}
	root := args[0]
	root, err := canonicalProjectFolder(root)
	if err != nil {
		fmt.Fprintf(stderr, "project add: %v\n", err)
		return 1
	}
	file, rel, err := projectPath(root, args[2])
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "project add: %v\n", err)
		return 1
	}
	detectedFormat := fileFormat(file)
	m, manifestErr := loadManifest(filepath.Join(root, ".sopsdeck.toml"))
	if manifestErr != nil && !os.IsNotExist(manifestErr) {
		fmt.Fprintf(stderr, "project add: read .sopsdeck.toml: %v\n", manifestErr)
		return 1
	}
	for _, entry := range m.ManagedFile {
		if filepath.ToSlash(entry.Path) == filepath.ToSlash(rel) {
			fmt.Fprintf(stderr, "project add: %s is already managed\n", rel)
			return 1
		}
	}
	keys := []string(nil)
	if len(args) == 5 {
		keys = splitKeys(args[4])
	}
	if err != nil {
		if code := setCreate(file, "", "", stderr, getenv); code != 0 {
			return code
		}
	} else {
		data, readErr := os.ReadFile(file)
		if readErr != nil {
			fmt.Fprintf(stderr, "project add: %v\n", readErr)
			return 1
		}
		if !isEncryptedBytes(data) {
			if err := encryptPlainFile(file, data, getenv, keys); err != nil {
				fmt.Fprintf(stderr, "project add: %s: %v\n", rel, err)
				return 1
			}
		}
	}
	return appendManagedEntry(root, rel, keys, formatName(detectedFormat), stdout, stderr, getenv)
}

func appendManagedEntry(root, rel string, keys []string, format string, stdout, stderr io.Writer, getenv func(string) string) int {
	manifestPath := filepath.Join(root, ".sopsdeck.toml")
	m, err := loadManifest(manifestPath)
	if os.IsNotExist(err) {
		if err := writeManifest(manifestPath, projectManifest{
			ManagedFile: []manifestFile{{
				Path:          filepath.ToSlash(rel),
				EncryptedKeys: keys,
				Format:        format,
			}},
			Recipient: recipientsFromEnv(root, getenv),
		}); err != nil {
			fmt.Fprintf(stderr, "project add: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "project initialized")
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "project add: %v\n", err)
		return 1
	}
	for _, entry := range m.ManagedFile {
		if filepath.ToSlash(entry.Path) == filepath.ToSlash(rel) {
			fmt.Fprintf(stderr, "project add: %s is already managed\n", rel)
			return 1
		}
	}
	m.ManagedFile = append(m.ManagedFile, manifestFile{Path: filepath.ToSlash(rel), EncryptedKeys: keys, Format: format})
	if err := writeManifest(manifestPath, m); err != nil {
		fmt.Fprintf(stderr, "project add: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "managed file added")
	return 0
}

func removeProjectFile(args []string, stdout, stderr io.Writer) int {
	if len(args) != 3 || args[1] != "--file" {
		fmt.Fprintln(stderr, "usage: sopsdeck project remove FOLDER --file PATH")
		return 1
	}
	root, err := canonicalProjectFolder(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "project remove: %v\n", err)
		return 1
	}
	// Removing metadata must also work for missing or unsafe file paths. No file is touched.
	rel := filepath.ToSlash(strings.TrimSpace(args[2]))
	manifestPath := filepath.Join(root, ".sopsdeck.toml")
	m, err := loadManifest(manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "project remove: %v\n", err)
		return 1
	}
	managed := m.ManagedFile[:0]
	found := false
	for _, entry := range m.ManagedFile {
		if filepath.ToSlash(entry.Path) == filepath.ToSlash(rel) {
			found = true
			continue
		}
		managed = append(managed, entry)
	}
	if !found {
		fmt.Fprintf(stderr, "project remove: %s is not managed\n", rel)
		return 1
	}
	m.ManagedFile = managed
	if err := writeManifest(manifestPath, m); err != nil {
		fmt.Fprintf(stderr, "project remove: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "managed file removed")
	return 0
}

func inspectProject(root string) (projectState, error) {
	root, err := canonicalProjectFolder(root)
	if err != nil {
		return projectState{}, err
	}
	if owner, _ := findManifest(root); owner != "" {
		root = owner
	}
	gitRoot, _, _ := gitTrackedRel(filepath.Join(root, ".sopsdeck.toml"))
	state := projectState{Path: root, GitRoot: gitRoot}
	rawFiles, err := managed.Candidates(root)
	if err != nil {
		return projectState{}, err
	}
	candidates := make([]projectCandidate, 0, len(rawFiles))
	for _, file := range rawFiles {
		candidates = append(candidates, projectCandidate{
			File: file,
			Keys: projectKeys(file),
		})
	}
	manifestPath := filepath.Join(root, ".sopsdeck.toml")
	m, err := loadManifest(manifestPath)
	if os.IsNotExist(err) {
		state.Candidates = candidates
		return state, nil
	}
	if err != nil {
		return projectState{}, fmt.Errorf("read .sopsdeck.toml: %w", err)
	}
	managedByRel := make(map[string]bool, len(m.ManagedFile))
	var managedFiles []projectFile
	for _, file := range m.ManagedFile {
		path, rel, err := projectPath(root, file.Path)
		if err != nil {
			message := err.Error()
			if os.IsNotExist(err) {
				message = "File is missing. Restore it from Git or remove this entry from managed files."
			}
			state.Warnings = append(state.Warnings, projectWarning{Path: file.Path, Message: message})
			continue
		}
		managedByRel[filepath.ToSlash(rel)] = true
		managedFiles = append(managedFiles, projectFile{
			File:    managed.File{Name: filepath.Base(path), Path: path, Rel: rel},
			Managed: true,
		})
	}
	sort.Slice(managedFiles, func(i, j int) bool { return managedFiles[i].Rel < managedFiles[j].Rel })
	var available []projectCandidate
	for _, file := range candidates {
		if !managedByRel[filepath.ToSlash(file.Rel)] {
			available = append(available, file)
		}
	}
	state.Initialized, state.Managed, state.Candidates = true, managedFiles, available
	return state, nil
}

func projectKeys(file managed.File) []string {
	data, err := os.ReadFile(file.Path)
	if err != nil || isEncryptedBytes(data) {
		return nil
	}
	branches, err := loadPlainBranches(fileFormat(file.Path), data)
	if err != nil {
		return nil
	}
	var keys []string
	for _, branch := range branches {
		keys = append(keys, leafPaths(branch, "")...)
	}
	return keys
}

func leafPaths(branch sops.TreeBranch, prefix string) []string {
	var paths []string
	for _, item := range branch {
		key, ok := item.Key.(string)
		if !ok {
			continue
		}
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		paths = append(paths, leafValuePaths(item.Value, path)...)
	}
	return paths
}

func leafValuePaths(value interface{}, prefix string) []string {
	switch value := value.(type) {
	case sops.TreeBranch:
		return leafPaths(value, prefix)
	case []interface{}:
		var paths []string
		for index, child := range value {
			paths = append(paths, leafValuePaths(child, fmt.Sprintf("%s[%d]", prefix, index))...)
		}
		return paths
	case sops.Comment, nil:
		return nil
	default:
		return []string{prefix}
	}
}

func initProject(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	root := args[0]
	root, err := canonicalProjectFolder(root)
	if err != nil {
		fmt.Fprintf(stderr, "project init: %v\n", err)
		return 1
	}
	if owner, _ := findManifest(root); owner != "" && owner != root {
		fmt.Fprintf(stderr, "project init: folder already belongs to Project %s\n", owner)
		return 1
	}
	var selections []projectSelection
	for i := 1; i < len(args); i++ {
		if args[i] != "--file" || i+1 >= len(args) {
			fmt.Fprintln(stderr, "usage: sopsdeck project init FOLDER [--file PATH]...")
			return 1
		}
		selection := projectSelection{Path: args[i+1]}
		i++
		if i+2 < len(args) && args[i+1] == "--keys" {
			selection.Keys = splitKeys(args[i+2])
			i += 2
		}
		selections = append(selections, selection)
	}
	if _, err := os.Stat(filepath.Join(root, ".sopsdeck.toml")); err == nil {
		fmt.Fprintln(stderr, "project init: already initialized")
		return 1
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "project init: %v\n", err)
		return 1
	}
	for _, selection := range selections {
		if _, _, err := projectPath(root, selection.Path); err != nil {
			fmt.Fprintf(stderr, "project init: %v\n", err)
			return 1
		}
	}
	entries := make([]manifestFile, 0, len(selections))
	for _, selection := range selections {
		file, rel, err := projectPath(root, selection.Path)
		if err != nil {
			fmt.Fprintf(stderr, "project init: %v\n", err)
			return 1
		}
		detectedFormat := fileFormat(file)
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(stderr, "project init: %v\n", err)
			return 1
		}
		if !isEncryptedBytes(data) {
			if err := encryptPlainFile(file, data, getenv, selection.Keys); err != nil {
				fmt.Fprintf(stderr, "project init: %s: %v\n", rel, err)
				return 1
			}
		}
		entries = append(entries, manifestFile{
			Path:          filepath.ToSlash(rel),
			EncryptedKeys: selection.Keys,
			Format:        formatName(detectedFormat),
		})
	}
	if err := writeManifest(filepath.Join(root, ".sopsdeck.toml"), projectManifest{
		ManagedFile: entries,
		Recipient:   recipientsFromEnv(root, getenv),
	}); err != nil {
		fmt.Fprintf(stderr, "project init: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "project initialized")
	return 0
}

func splitKeys(raw string) []string {
	var out []string
	for _, key := range strings.Split(raw, ",") {
		if key = strings.TrimSpace(key); key != "" {
			out = append(out, key)
		}
	}
	return out
}

func encryptProjectKeys(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	file, keys, errMsg := parseEncryptArgs(args)
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
		return 1
	}
	if err := setFileEncryptedKeys(file, keys, getenv); err != nil {
		fmt.Fprintf(stderr, "project encrypt: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "encrypted keys updated")
	return 0
}

func parseEncryptArgs(args []string) (file string, keys []string, errMsg string) {
	keys = []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--keys":
			i++
			if i >= len(args) {
				return "", nil, "usage: sopsdeck project encrypt FILE --keys PATH,..."
			}
			keys = splitKeys(args[i])
		case "-f", "--file":
			i++
			if i >= len(args) {
				return "", nil, "usage: sopsdeck project encrypt FILE --keys PATH,..."
			}
			file = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", nil, "usage: sopsdeck project encrypt FILE --keys PATH,..."
			}
			if file != "" {
				return "", nil, "usage: sopsdeck project encrypt FILE --keys PATH,..."
			}
			file = args[i]
		}
	}
	if file == "" {
		return "", nil, "usage: sopsdeck project encrypt FILE --keys PATH,..."
	}
	return file, keys, ""
}

func setFileEncryptedKeys(file string, keys []string, getenv func(string) string) error {
	mapping, _, manifestPath := mappingFor(file)
	if manifestPath == "" {
		return fmt.Errorf("file is not managed")
	}
	m, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	rel := filepath.ToSlash(mapping.Path)
	found := false
	for i := range m.ManagedFile {
		if filepath.ToSlash(m.ManagedFile[i].Path) != rel {
			continue
		}
		m.ManagedFile[i].EncryptedKeys = keys
		found = true
		break
	}
	if !found {
		return fmt.Errorf("file is not managed")
	}
	if err := writeManifest(manifestPath, m); err != nil {
		return err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	if !isEncryptedBytes(data) {
		return nil
	}
	plain, err := decryptManaged(file)
	if err != nil {
		return err
	}
	return encryptPlainFile(file, plain, getenv, keys)
}

func decryptManaged(file string) ([]byte, error) {
	return decrypt.File(file, formatName(fileFormat(file)))
}

func encryptedKeyRegex(keys []string) string {
	seen := make(map[string]bool, len(keys))
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		key = encryptedLeafName(key)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		parts = append(parts, regexp.QuoteMeta(key))
	}
	if len(parts) == 0 {
		return ""
	}
	return "^(?:" + strings.Join(parts, "|") + ")$"
}

func encryptedLeafName(key string) string {
	key = strings.TrimSpace(key)
	if i := strings.LastIndex(key, "."); i >= 0 {
		key = key[i+1:]
	}
	return strings.Trim(key, "[]")
}

func encryptedLeaf(key string, keys []string) bool {
	if len(keys) == 0 {
		return false
	}
	leaf := encryptedLeafName(key)
	for _, item := range keys {
		if item == key || encryptedLeafName(item) == leaf {
			return true
		}
	}
	return false
}

func pairEncrypted(key string, keys []string, encryptsAll bool, regex string) bool {
	if encryptsAll {
		return true
	}
	if len(keys) > 0 {
		return encryptedLeaf(key, keys)
	}
	if regex == "" || regex == neverMatchRegex {
		return false
	}
	re, err := regexp.Compile(regex)
	if err != nil {
		return false
	}
	return re.MatchString(encryptedLeafName(key))
}

func fileEncryptionPolicy(path string) (keys []string, encryptsAll bool, regex string) {
	mapping, _, _ := mappingFor(path)
	keys = mapping.EncryptedKeys
	if fileFormat(path) == formats.Dotenv {
		return keys, true, ""
	}
	if len(keys) > 0 {
		return keys, false, encryptedKeyRegex(keys)
	}
	data, err := os.ReadFile(path)
	if err != nil || !isEncryptedBytes(data) {
		return keys, false, ""
	}
	tree, err := common.LoadEncryptedFile(common.StoreForFormat(fileFormat(path), config.NewStoresConfig()), path)
	if err != nil {
		return keys, false, ""
	}
	regex = tree.Metadata.EncryptedRegex
	return keys, regex == "", regex
}

type managedPair struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Encrypted bool   `json:"encrypted"`
}

func projectPath(root, raw string) (string, string, error) {
	root, err := canonicalProjectFolder(root)
	if err != nil {
		return "", "", err
	}
	if owner, _ := findManifest(root); owner != "" && owner != root {
		return "", "", fmt.Errorf("folder belongs to Project %s", owner)
	}
	rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(raw)))
	if rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("file must stay inside the Project")
	}
	file := filepath.Join(root, rel)
	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return "", "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", "", fmt.Errorf("file must stay inside the Project without symlinks: %s", rel)
		}
		if info.IsDir() && managed.SkipDirectory(current, root) {
			return "", "", fmt.Errorf("%s is outside this Project's scope", rel)
		}
	}
	info, err := os.Stat(file)
	if err != nil {
		return file, rel, err
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("%s is not a regular file", rel)
	}
	return file, rel, nil
}

func canonicalProjectFolder(root string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project path must be a folder")
	}
	return root, nil
}

func isEncryptedBytes(data []byte) bool {
	sample := string(data)
	return strings.Contains(sample, "ENC[") &&
		(strings.Contains(sample, `"sops"`) || strings.Contains(sample, "sops:") || strings.Contains(sample, "sops_"))
}
