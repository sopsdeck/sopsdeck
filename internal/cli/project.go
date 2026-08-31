package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
	Initialized bool               `json:"initialized"`
	Managed     []projectFile      `json:"managed"`
	Candidates  []projectCandidate `json:"candidates"`
}

type projectCandidate struct {
	managed.File
	Keys []string `json:"keys,omitempty"`
}

type projectSelection struct {
	Path string   `json:"path"`
	Keys []string `json:"keys"`
}

const projectUsage = "usage: sopsdeck project files FOLDER | init FOLDER [--file PATH]... | add FOLDER --file PATH | encrypt FILE --keys PATH,..."

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
	file, rel, err := projectPath(root, args[2])
	keys := []string(nil)
	if len(args) == 5 {
		keys = splitKeys(args[4])
	}
	if err != nil {
		if filepath.IsAbs(args[2]) || strings.Contains(filepath.Clean(args[2]), ".."+string(filepath.Separator)) {
			fmt.Fprintf(stderr, "project add: %v\n", err)
			return 1
		}
		rel = filepath.Clean(filepath.FromSlash(args[2]))
		file = filepath.Join(root, rel)
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
	manifestPath := filepath.Join(root, ".sopsdeck.toml")
	m, err := loadManifest(manifestPath)
	if os.IsNotExist(err) {
		if err := writeManifest(manifestPath, projectManifest{
			ManagedFile: []manifestFile{{
				Path:          filepath.ToSlash(rel),
				EncryptedKeys: keys,
			}},
			Owner: ownersFromEnv(root, getenv),
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
	m.ManagedFile = append(m.ManagedFile, manifestFile{Path: filepath.ToSlash(rel), EncryptedKeys: keys})
	if err := writeManifest(manifestPath, m); err != nil {
		fmt.Fprintf(stderr, "project add: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "managed file added")
	return 0
}

func inspectProject(root string) (projectState, error) {
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
		return projectState{Candidates: candidates}, nil
	}
	if err != nil {
		return projectState{}, err
	}
	managedByRel := make(map[string]bool, len(m.ManagedFile))
	for _, file := range m.ManagedFile {
		managedByRel[filepath.ToSlash(file.Path)] = true
	}
	var managedFiles []projectFile
	var available []projectCandidate
	for _, file := range candidates {
		if managedByRel[filepath.ToSlash(file.Rel)] {
			managedFiles = append(managedFiles, projectFile{File: file.File, Managed: true})
		} else {
			available = append(available, file)
		}
	}
	return projectState{Initialized: true, Managed: managedFiles, Candidates: available}, nil
}

func projectKeys(file managed.File) []string {
	data, err := os.ReadFile(file.Path)
	if err != nil || isEncryptedBytes(data) {
		return nil
	}
	branches, err := common.StoreForFormat(fileFormat(file.Path), config.NewStoresConfig()).LoadPlainFile(data)
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
	entries := make([]manifestFile, 0, len(selections))
	for _, selection := range selections {
		file, rel, err := projectPath(root, selection.Path)
		if err != nil {
			fmt.Fprintf(stderr, "project init: %v\n", err)
			return 1
		}
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
		entries = append(entries, manifestFile{Path: filepath.ToSlash(rel), EncryptedKeys: selection.Keys})
	}
	if err := writeManifest(filepath.Join(root, ".sopsdeck.toml"), projectManifest{
		ManagedFile: entries,
		Owner:       ownersFromEnv(root, getenv),
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
	rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(raw)))
	if rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("file must stay inside the Project")
	}
	file := filepath.Join(root, rel)
	info, err := os.Stat(file)
	if err != nil {
		return "", "", err
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("%s is not a regular file", rel)
	}
	return file, rel, nil
}

func isEncryptedBytes(data []byte) bool {
	sample := string(data)
	return strings.Contains(sample, "ENC[") &&
		(strings.Contains(sample, `"sops"`) || strings.Contains(sample, "sops:") || strings.Contains(sample, "sops_"))
}
