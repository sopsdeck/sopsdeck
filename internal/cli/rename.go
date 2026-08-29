package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/getsops/sops/v3/aes"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/decrypt"
	"github.com/getsops/sops/v3/keyservice"
)

type keyReference struct {
	Key   string   `json:"key"`
	Count int      `json:"count"`
	Files []string `json:"files,omitempty"`
}

func parseFileFlag(args []string, cmd string) (string, string, int) {
	var file string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				return "", fmt.Sprintf("%s: -f requires a file", cmd), 1
			}
			file = args[i]
		default:
			return "", fmt.Sprintf("%s: unknown argument %s", cmd, args[i]), 1
		}
	}
	if file == "" {
		return "", fmt.Sprintf("usage: sopsdeck %s -f FILE", cmd), 1
	}
	return file, "", 0
}

func managedKeys(file string) ([]string, error) {
	format := fileFormat(file)
	plain, err := decrypt.File(file, formatName(format))
	if err != nil {
		return nil, err
	}
	pairs, err := plainEnv(plain, format)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		if k == "sops" || strings.HasPrefix(k, "sops_") {
			continue
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func projectReferenceReport(file string) ([]keyReference, error) {
	keys, err := managedKeys(file)
	if err != nil {
		return nil, err
	}
	root := projectRootFor(file)
	managedRel := managedExcludeSet(file, root)
	counts, files, err := scanProjectReferences(root, keys, managedRel)
	if err != nil {
		return nil, err
	}
	out := make([]keyReference, 0, len(keys))
	for _, k := range keys {
		out = append(out, keyReference{Key: k, Count: counts[k], Files: files[k]})
	}
	return out, nil
}

func cmdReferences(args []string, stdout, stderr io.Writer) int {
	file, usage, code := parseFileFlag(args, "references")
	if usage != "" {
		fmt.Fprintln(stderr, usage)
		return code
	}
	report, err := projectReferenceReport(file)
	if err != nil {
		fmt.Fprintf(stderr, "references: %v\n", err)
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(report); err != nil {
		fmt.Fprintf(stderr, "references: %v\n", err)
		return 1
	}
	return 0
}

func cmdUnused(args []string, stdout, stderr io.Writer) int {
	file, usage, code := parseFileFlag(args, "unused")
	if usage != "" {
		fmt.Fprintln(stderr, usage)
		return code
	}
	report, err := projectReferenceReport(file)
	if err != nil {
		fmt.Fprintf(stderr, "unused: %v\n", err)
		return 1
	}
	var unused []string
	for _, ref := range report {
		if ref.Count == 0 {
			unused = append(unused, ref.Key)
		}
	}
	if unused == nil {
		unused = []string{}
	}
	if err := json.NewEncoder(stdout).Encode(unused); err != nil {
		fmt.Fprintf(stderr, "unused: %v\n", err)
		return 1
	}
	return 0
}

type renameFlags struct {
	file   string
	oldKey string
	newKey string
	yes    bool
}

func parseRenameFlags(args []string, stderr io.Writer) (renameFlags, int) {
	var f renameFlags
	var positionals []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "rename: -f requires a file")
				return f, 1
			}
			f.file = args[i]
		case "--yes":
			f.yes = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "rename: unknown flag %s\n", args[i])
				return f, 1
			}
			positionals = append(positionals, args[i])
		}
	}
	if f.file == "" || len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: sopsdeck rename OLD NEW -f FILE [--yes]")
		return f, 1
	}
	f.oldKey, f.newKey = positionals[0], positionals[1]
	if f.oldKey == f.newKey {
		fmt.Fprintln(stderr, "rename: old and new key are the same")
		return f, 1
	}
	return f, 0
}

func cmdRename(args []string, stdout, stderr io.Writer) int {
	f, code := parseRenameFlags(args, stderr)
	if code != 0 {
		return code
	}
	root := projectRootFor(f.file)
	managedRel := managedExcludeSet(f.file, root)
	changes, err := plannedReferenceRenames(root, f.oldKey, f.newKey, managedRel)
	if err != nil {
		fmt.Fprintf(stderr, "rename: %v\n", err)
		return 1
	}
	if !f.yes {
		printRenamePlan(stdout, f, changes)
		return 0
	}
	if err := applyRename(f, root, changes, stderr); err != nil {
		fmt.Fprintf(stderr, "rename: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "renamed %s -> %s\n", f.oldKey, f.newKey)
	return 0
}

func printRenamePlan(stdout io.Writer, f renameFlags, changes []plannedRename) {
	fmt.Fprintf(stdout, "rename %s -> %s in %s\n", f.oldKey, f.newKey, f.file)
	if len(changes) > 0 {
		fmt.Fprintf(stdout, "rewrite references in %d file(s):\n", len(changes))
		for _, c := range changes {
			fmt.Fprintf(stdout, "  %s (%d)\n", c.file, c.count)
		}
	} else {
		fmt.Fprintln(stdout, "no references to rewrite")
	}
	fmt.Fprintln(stdout, "rerun with --yes to apply")
}

func applyRename(f renameFlags, root string, changes []plannedRename, stderr io.Writer) error {
	if err := renameManagedKey(f.file, f.oldKey, f.newKey, stderr); err != nil {
		return err
	}
	for _, c := range changes {
		if err := rewriteReferencesInFile(root, c.file, f.oldKey, f.newKey); err != nil {
			return err
		}
	}
	return nil
}

func projectRootFor(file string) string {
	_, root, _ := mappingFor(file)
	if root == "" {
		return filepath.Dir(file)
	}
	return root
}

func managedExcludeSet(file, root string) map[string]bool {
	managedRel := map[string]bool{}
	if rel, err := filepath.Rel(root, file); err == nil {
		managedRel[filepath.ToSlash(rel)] = true
	}
	if _, manifestPath := findManifest(file); manifestPath != "" {
		if manifest, err := loadManifest(manifestPath); err == nil {
			for _, entry := range manifest.ManagedFile {
				managedRel[filepath.ToSlash(entry.Path)] = true
			}
		}
	}
	return managedRel
}

type plannedRename struct {
	file  string
	count int
}

func plannedReferenceRenames(root, oldKey, newKey string, managedRel map[string]bool) ([]plannedRename, error) {
	re := referenceRenameRegexp(oldKey)
	var changes []plannedRename
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
		if managedRel[rel] || isBinary(path) {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if c := len(re.FindAll(body, -1)); c > 0 {
			changes = append(changes, plannedRename{file: rel, count: c})
		}
		return nil
	})
	return changes, err
}

func referenceRenameRegexp(key string) *regexp.Regexp {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(key) + `\b`)
}

func rewriteReferencesInFile(root, rel, oldKey, newKey string) error {
	path := filepath.Join(root, rel)
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	re := referenceRenameRegexp(oldKey)
	out := re.ReplaceAll(body, []byte(newKey))
	if string(out) == string(body) {
		return nil
	}
	return writeAtomicText(path, out)
}

func writeAtomicText(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".sopsdeck-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func renameManagedKey(file, oldKey, newKey string, stderr io.Writer) error {
	format := fileFormat(file)
	store := common.StoreForFormat(format, config.NewStoresConfig())
	tree, err := common.LoadEncryptedFile(store, file)
	if err != nil {
		return err
	}
	cipher := aes.NewCipher()
	dataKey, err := common.DecryptTree(common.DecryptTreeOpts{
		Tree:        tree,
		Cipher:      cipher,
		KeyServices: []keyservice.KeyServiceClient{keyservice.NewLocalClient()},
	})
	if err != nil {
		return err
	}
	plain, err := decrypt.File(file, formatName(format))
	if err != nil {
		return err
	}
	pairs, err := plainEnv(plain, format)
	if err != nil {
		return err
	}
	value, ok := pairs[oldKey]
	if !ok {
		return fmt.Errorf("missing key %s", oldKey)
	}
	branch, err := tree.Branches[0].Unset([]interface{}{oldKey})
	if err != nil {
		return err
	}
	tree.Branches[0] = branch
	tree.Branches[0], _ = tree.Branches[0].Set([]interface{}{newKey}, value)
	if err := common.EncryptTree(common.EncryptTreeOpts{DataKey: dataKey, Tree: tree, Cipher: cipher}); err != nil {
		return err
	}
	out, err := store.EmitEncryptedFile(*tree)
	if err != nil {
		return err
	}
	return writeAtomic(file, out)
}
