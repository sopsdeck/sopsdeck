package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	sopsage "github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/cmd/sops/formats"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/decrypt"
	"github.com/getsops/sops/v3/keyservice"
	"github.com/getsops/sops/v3/version"
	"go.yaml.in/yaml/v3"
)

var errLoneValueNeedsKey = errors.New("lone paste value requires KEY")

type pasteFlags struct {
	file string
	yes  bool
	key  string
}

func applyPaste(args []string, payload []byte, stdout, stderr io.Writer, getenv func(string) string) int {
	flags, usage := parsePasteFlags(args)
	if usage != "" {
		fmt.Fprintln(stderr, usage)
		return 1
	}
	pairs, err := parsePastePayload(payload, flags.key)
	if err != nil {
		if errors.Is(err, errLoneValueNeedsKey) {
			fmt.Fprintln(stderr, "usage: sopsdeck set KEY -f FILE [--yes]")
			return 1
		}
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	if len(pairs) == 0 {
		fmt.Fprintln(stderr, "set: empty paste")
		return 1
	}

	current, err := currentEnvPairs(flags.file)
	if err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	adds, changes := classifyPasteKeys(current, pairs)
	if !flags.yes {
		printPastePreview(stdout, adds, changes)
		return 0
	}

	if _, err := os.Stat(flags.file); os.IsNotExist(err) {
		return pasteCreate(flags.file, pairs, stderr, getenv)
	}
	return pasteApplyExisting(flags.file, pairs, stderr)
}

func parsePasteFlags(args []string) (pasteFlags, string) {
	var flags pasteFlags
	var positionals []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				return pasteFlags{}, "set: -f requires a file"
			}
			flags.file = args[i]
		case "--yes":
			flags.yes = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return pasteFlags{}, fmt.Sprintf("set: unknown flag %s", args[i])
			}
			positionals = append(positionals, args[i])
		}
	}
	if flags.file == "" {
		return pasteFlags{}, "usage: sopsdeck set [KEY] -f FILE [--yes]"
	}
	if len(positionals) > 1 {
		return pasteFlags{}, "usage: sopsdeck set [KEY] -f FILE [--yes]"
	}
	if len(positionals) == 1 {
		flags.key = positionals[0]
	}
	return flags, ""
}

func parsePastePayload(payload []byte, loneKey string) (map[string]string, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty paste")
	}
	if trimmed[0] == '{' {
		var doc map[string]any
		if err := json.Unmarshal(trimmed, &doc); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		return stringifyPasteMap(doc), nil
	}
	text := string(payload)
	if pasteLooksDotenv(text) {
		return dotenvMap([]byte(text)), nil
	}
	var doc map[string]any
	if err := yaml.Unmarshal(payload, &doc); err == nil && pasteLooksYAMLMap(doc) {
		return stringifyPasteMap(doc), nil
	}
	if loneKey == "" {
		return nil, errLoneValueNeedsKey
	}
	return map[string]string{loneKey: text}, nil
}

func pasteLooksDotenv(text string) bool {
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, _, ok := strings.Cut(line, "=")
		if ok && k != "" {
			return true
		}
	}
	return false
}

func pasteLooksYAMLMap(doc map[string]any) bool {
	if len(doc) == 0 {
		return false
	}
	for k := range doc {
		if k == "" {
			return false
		}
	}
	return true
}

func stringifyPasteMap(doc map[string]any) map[string]string {
	out := make(map[string]string, len(doc))
	for k, raw := range doc {
		if s, ok := raw.(string); ok {
			out[k] = s
			continue
		}
		out[k] = fmt.Sprint(raw)
	}
	return out
}

func currentEnvPairs(file string) (map[string]string, error) {
	format := fileFormat(file)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	plain, err := decrypt.File(file, formatName(format))
	if err != nil {
		mapping, _, _ := mappingFor(file)
		if mapping.Path == "" {
			return nil, err
		}
		plain, err = os.ReadFile(file)
		if err != nil || isEncryptedBytes(plain) {
			return nil, err
		}
	}
	return plainPairs(plain, format)
}

func classifyPasteKeys(current, incoming map[string]string) (adds, changes []string) {
	for k := range incoming {
		if _, ok := current[k]; ok {
			changes = append(changes, k)
		} else {
			adds = append(adds, k)
		}
	}
	sort.Strings(adds)
	sort.Strings(changes)
	return adds, changes
}

func printPastePreview(stdout io.Writer, adds, changes []string) {
	fmt.Fprintf(stdout, "preview %d add %d change\n", len(adds), len(changes))
	for _, k := range adds {
		fmt.Fprintf(stdout, "add %s\n", k)
	}
	for _, k := range changes {
		fmt.Fprintf(stdout, "change %s\n", k)
	}
}

func pasteCreate(file string, pairs map[string]string, stderr io.Writer, getenv func(string) string) int {
	pub, err := ageRecipientFromEnv(getenv)
	if err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	mk, err := sopsage.MasterKeyFromRecipient(pub)
	if err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	branch := sops.TreeBranch{}
	for _, k := range sortedKeys(pairs) {
		branch = append(branch, sops.TreeItem{Key: k, Value: pairs[k]})
	}
	tree := sops.Tree{
		FilePath: file,
		Metadata: sops.Metadata{
			Version:           version.Version,
			UnencryptedSuffix: sops.DefaultUnencryptedSuffix,
			KeyGroups:         []sops.KeyGroup{{mk}},
		},
		Branches: sops.TreeBranches{branch},
	}
	svcs := []keyservice.KeyServiceClient{keyservice.NewLocalClient()}
	dataKey, errs := tree.GenerateDataKeyWithKeyServices(svcs)
	if len(errs) > 0 {
		fmt.Fprintf(stderr, "set: %v\n", errs)
		return 1
	}
	if err := common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    &tree,
		Cipher:  aes.NewCipher(),
	}); err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	format := fileFormat(file)
	store := common.StoreForFormat(format, config.NewStoresConfig())
	out, err := store.EmitEncryptedFile(tree)
	if err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	if err := writeAtomic(file, out); err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	return 0
}

func encryptPlainFile(file string, plain []byte, getenv func(string) string, keys []string) error {
	store := common.StoreForFormat(fileFormat(file), config.NewStoresConfig())
	branches, err := store.LoadPlainFile(plain)
	if err != nil {
		return err
	}
	pub, err := ageRecipientFromEnv(getenv)
	if err != nil {
		return err
	}
	mk, err := sopsage.MasterKeyFromRecipient(pub)
	if err != nil {
		return err
	}
	tree := sops.Tree{
		FilePath: file,
		Metadata: sops.Metadata{
			Version:           version.Version,
			UnencryptedSuffix: sops.DefaultUnencryptedSuffix,
			KeyGroups:         []sops.KeyGroup{{mk}},
		},
		Branches: branches,
	}
	tree.Metadata.EncryptedRegex = encryptedKeyRegex(keys)
	if tree.Metadata.EncryptedRegex != "" {
		tree.Metadata.UnencryptedSuffix = ""
	}
	dataKey, errs := tree.GenerateDataKeyWithKeyServices([]keyservice.KeyServiceClient{keyservice.NewLocalClient()})
	if len(errs) > 0 {
		return fmt.Errorf("%v", errs)
	}
	if err := common.EncryptTree(common.EncryptTreeOpts{DataKey: dataKey, Tree: &tree, Cipher: aes.NewCipher()}); err != nil {
		return err
	}
	out, err := store.EmitEncryptedFile(tree)
	if err != nil {
		return err
	}
	return writeAtomic(file, out)
}

func pasteApplyExisting(file string, pairs map[string]string, stderr io.Writer) int {
	format := fileFormat(file)
	store := common.StoreForFormat(format, config.NewStoresConfig())
	if raw, readErr := os.ReadFile(file); readErr == nil && !isEncryptedBytes(raw) {
		mapping, _, _ := mappingFor(file)
		if mapping.Path == "" {
			fmt.Fprintln(stderr, "set: not a SOPS-encrypted file")
			return 1
		}
		branches, err := store.LoadPlainFile(raw)
		if err != nil {
			fmt.Fprintf(stderr, "set: %v\n", err)
			return 1
		}
		for k := range pairs {
			path, err := treePath(k, format != formats.Dotenv)
			if err != nil {
				fmt.Fprintf(stderr, "set: %v\n", err)
				return 1
			}
			branches[0], _ = branches[0].Set(path, pairs[k])
		}
		out, err := store.EmitPlainFile(branches)
		if err != nil {
			fmt.Fprintf(stderr, "set: %v\n", err)
			return 1
		}
		if err := writeAtomic(file, out); err != nil {
			fmt.Fprintf(stderr, "set: %v\n", err)
			return 1
		}
		return 0
	}
	tree, err := common.LoadEncryptedFile(store, file)
	if err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	cipher := aes.NewCipher()
	dataKey, err := common.DecryptTree(common.DecryptTreeOpts{
		Tree:        tree,
		Cipher:      cipher,
		KeyServices: []keyservice.KeyServiceClient{keyservice.NewLocalClient()},
	})
	if err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	for _, k := range sortedKeys(pairs) {
		path, err := treePath(k, format != formats.Dotenv)
		if err != nil {
			fmt.Fprintf(stderr, "set: %v\n", err)
			return 1
		}
		tree.Branches[0], _ = tree.Branches[0].Set(path, pairs[k])
	}
	if err := common.EncryptTree(common.EncryptTreeOpts{DataKey: dataKey, Tree: tree, Cipher: cipher}); err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	out, err := store.EmitEncryptedFile(*tree)
	if err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	if err := writeAtomic(file, out); err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	return 0
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func readPaste(stdin io.Reader) []byte {
	if stdin == nil {
		return nil
	}
	if f, ok := stdin.(*os.File); ok {
		info, err := f.Stat()
		if err != nil {
			return nil
		}
		if info.Mode()&os.ModeCharDevice != 0 {
			return nil
		}
	}
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return nil
	}
	return bytes.TrimSpace(raw)
}
