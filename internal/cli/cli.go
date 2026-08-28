package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"filippo.io/age"
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

	appver "sopsdeck/internal/version"
)

func Main(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: sopsdeck <get|set|del|run|identity|commit|sync|review|history|restore|recipient|publish|files|drive|scan|mcp> ...")
		return 1
	}
	switch args[0] {
	case "--version", "-V", "version":
		fmt.Fprintln(stdout, appver.Version)
		return 0
	case "get":
		return cmdGet(args[1:], stdout, stderr)
	case "set":
		return cmdSet(args[1:], stdin, stdout, stderr, getenv)
	case "del":
		return cmdDel(args[1:], stdout, stderr)
	case "run":
		return cmdRun(args[1:], stdin, stdout, stderr)
	case "identity":
		return cmdIdentity(args[1:], stdout, stderr, getenv)
	case "commit":
		return cmdCommit(args[1:], stdout, stderr)
	case "sync":
		return cmdSync(args[1:], stdout, stderr)
	case "review":
		return cmdReview(args[1:], stdout, stderr)
	case "history":
		return cmdHistory(args[1:], stdout, stderr)
	case "restore":
		return cmdRestore(args[1:], stdout, stderr)
	case "recipient":
		return cmdRecipient(args[1:], stdout, stderr, getenv)
	case "publish":
		return cmdPublish(args[1:], stdout, stderr, getenv)
	case "files":
		return cmdFiles(args[1:], stdout, stderr)
	case "drive":
		return cmdDrive(args[1:], stdout, stderr, getenv)
	case "scan":
		return cmdScan(args[1:], stdout, stderr)
	case "mcp":
		return cmdMCP(args[1:], stdin, stdout, stderr, getenv)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 1
	}
}

type getFlags struct {
	key    string
	file   string
	output string
	at     string
}

func parseGetFlags(args []string) (getFlags, string) {
	var flags getFlags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				return getFlags{}, "get: -f requires a file"
			}
			flags.file = args[i]
		case "--output":
			i++
			if i >= len(args) {
				return getFlags{}, "get: --output requires a format"
			}
			flags.output = args[i]
		case "--at":
			i++
			if i >= len(args) {
				return getFlags{}, "get: --at requires a revision"
			}
			flags.at = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return getFlags{}, fmt.Sprintf("get: unknown flag %s", args[i])
			}
			if flags.key != "" {
				return getFlags{}, "get: extra argument"
			}
			flags.key = args[i]
		}
	}
	if flags.file == "" {
		return getFlags{}, "usage: sopsdeck get [KEY] -f FILE"
	}
	if flags.output != "" && flags.output != "json" {
		return getFlags{}, fmt.Sprintf("get: unknown --output %s", flags.output)
	}
	return flags, ""
}

func cmdGet(args []string, stdout, stderr io.Writer) int {
	flags, usage := parseGetFlags(args)
	if usage != "" {
		fmt.Fprintln(stderr, usage)
		return 1
	}
	key, file, output := flags.key, flags.file, flags.output
	format := fileFormat(file)
	var plain []byte
	var err error
	if flags.at != "" {
		raw, showErr := gitShowAt(file, flags.at)
		if showErr != nil {
			fmt.Fprintf(stderr, "get: %v\n", showErr)
			return 1
		}
		plain, err = decrypt.Data(raw, formatName(format))
	} else {
		plain, err = decrypt.File(file, formatName(format))
	}
	if err != nil {
		fmt.Fprintln(stderr, explainGet(err))
		return 1
	}
	warnEASCLI(file, stderr)
	if key == "" {
		if output == "json" {
			pairs, err := plainEnv(plain, format)
			if err != nil {
				fmt.Fprintf(stderr, "get: %v\n", err)
				return 1
			}
			enc, err := json.Marshal(pairs)
			if err != nil {
				fmt.Fprintf(stderr, "get: %v\n", err)
				return 1
			}
			fmt.Fprintln(stdout, string(enc))
			return 0
		}
		if _, err := stdout.Write(plain); err != nil {
			fmt.Fprintf(stderr, "get: %v\n", err)
			return 1
		}
		if len(plain) > 0 && plain[len(plain)-1] != '\n' {
			fmt.Fprintln(stdout)
		}
		return 0
	}
	value, ok, err := lookupValue(plain, format, key)
	if err != nil {
		fmt.Fprintf(stderr, "get: %v\n", err)
		return 1
	}
	if !ok {
		fmt.Fprintf(stderr, "get: missing key %s\n", key)
		return 1
	}
	fmt.Fprintln(stdout, value)
	return 0
}

func cmdSet(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	if payload := readPaste(stdin); len(payload) > 0 {
		return applyPaste(args, payload, stdout, stderr, getenv)
	}
	_ = stdout
	var key, value, file string
	var positionals []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "set: -f requires a file")
				return 1
			}
			file = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "set: unknown flag %s\n", args[i])
				return 1
			}
			positionals = append(positionals, args[i])
		}
	}
	if file == "" {
		fmt.Fprintln(stderr, "usage: sopsdeck set [KEY VALUE] -f FILE")
		return 1
	}
	if len(positionals) == 0 {
		if _, err := os.Stat(file); err == nil {
			fmt.Fprintf(stderr, "set: %s already exists\n", file)
			return 1
		} else if !os.IsNotExist(err) {
			fmt.Fprintf(stderr, "set: %v\n", err)
			return 1
		}
		return setCreate(file, "", "", stderr, getenv)
	}
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: sopsdeck set [KEY VALUE] -f FILE")
		return 1
	}
	key, value = positionals[0], positionals[1]
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return setCreate(file, key, value, stderr, getenv)
	}

	format := fileFormat(file)
	store := common.StoreForFormat(format, config.NewStoresConfig())
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
	tree.Branches[0], _ = tree.Branches[0].Set([]interface{}{key}, value)
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

func writeAtomic(path string, data []byte) error {
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
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func cmdDel(args []string, stdout, stderr io.Writer) int {
	_ = stdout
	var key, file string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "del: -f requires a file")
				return 1
			}
			file = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "del: unknown flag %s\n", args[i])
				return 1
			}
			if key != "" {
				fmt.Fprintln(stderr, "del: extra argument")
				return 1
			}
			key = args[i]
		}
	}
	if key == "" || file == "" {
		fmt.Fprintln(stderr, "usage: sopsdeck del KEY -f FILE")
		return 1
	}

	format := fileFormat(file)
	store := common.StoreForFormat(format, config.NewStoresConfig())
	tree, err := common.LoadEncryptedFile(store, file)
	if err != nil {
		fmt.Fprintf(stderr, "del: %v\n", err)
		return 1
	}
	cipher := aes.NewCipher()
	dataKey, err := common.DecryptTree(common.DecryptTreeOpts{
		Tree:        tree,
		Cipher:      cipher,
		KeyServices: []keyservice.KeyServiceClient{keyservice.NewLocalClient()},
	})
	if err != nil {
		fmt.Fprintf(stderr, "del: %v\n", err)
		return 1
	}
	branch, err := tree.Branches[0].Unset([]interface{}{key})
	if err != nil {
		fmt.Fprintf(stderr, "del: %v\n", err)
		return 1
	}
	tree.Branches[0] = branch
	if err := common.EncryptTree(common.EncryptTreeOpts{DataKey: dataKey, Tree: tree, Cipher: cipher}); err != nil {
		fmt.Fprintf(stderr, "del: %v\n", err)
		return 1
	}
	out, err := store.EmitEncryptedFile(*tree)
	if err != nil {
		fmt.Fprintf(stderr, "del: %v\n", err)
		return 1
	}
	if err := writeAtomic(file, out); err != nil {
		fmt.Fprintf(stderr, "del: %v\n", err)
		return 1
	}
	return 0
}

func cmdRun(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var file string
	dash := -1
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			dash = i
			break
		}
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "run: -f requires a file")
				return 1
			}
			file = args[i]
		default:
			fmt.Fprintln(stderr, "usage: sopsdeck run -f FILE -- CMD [ARG...]")
			return 1
		}
	}
	if dash < 0 || file == "" || dash+1 >= len(args) {
		fmt.Fprintln(stderr, "usage: sopsdeck run -f FILE -- CMD [ARG...]")
		return 1
	}
	argv := args[dash+1:]
	format := fileFormat(file)
	plain, err := decrypt.File(file, formatName(format))
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 1
	}
	fileEnv, err := plainEnv(plain, format)
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 1
	}
	childEnv := os.Environ()
	have := map[string]bool{}
	for _, kv := range childEnv {
		k, _, _ := strings.Cut(kv, "=")
		have[k] = true
	}
	for k, v := range fileEnv {
		if have[k] {
			continue
		}
		childEnv = append(childEnv, k+"="+v)
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = childEnv
	err = cmd.Run()
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	fmt.Fprintf(stderr, "run: %v\n", err)
	return 1
}

func plainEnv(plain []byte, format formats.Format) (map[string]string, error) {
	if format == formats.Dotenv {
		return dotenvMap(plain), nil
	}
	var doc map[string]any
	var err error
	switch format {
	case formats.Json:
		err = json.Unmarshal(plain, &doc)
	case formats.Yaml:
		err = yaml.Unmarshal(plain, &doc)
	default:
		return nil, fmt.Errorf("unsupported format")
	}
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for k, raw := range doc {
		if s, ok := raw.(string); ok {
			out[k] = s
			continue
		}
		out[k] = fmt.Sprint(raw)
	}
	return out, nil
}

func dotenvMap(plain []byte) map[string]string {
	out := map[string]string{}
	for line := range strings.SplitSeq(string(plain), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if ok && k != "" {
			out[k] = strings.ReplaceAll(v, `\n`, "\n")
		}
	}
	return out
}

func warnEASCLI(file string, stderr io.Writer) {
	if filepath.Base(file) != "eas.json" {
		return
	}
	fmt.Fprintln(stderr, easJSONWarning)
}

const easJSONWarning = "eas.json: EAS CLI will not read SOPS ciphertext"

func fileFormat(path string) formats.Format {
	base := filepath.Base(path)
	switch {
	case base == ".env" || strings.HasPrefix(base, ".env.") || strings.HasSuffix(strings.ToLower(base), ".env"):
		return formats.Dotenv
	case strings.HasSuffix(strings.ToLower(base), ".json"):
		return formats.Json
	case strings.HasSuffix(strings.ToLower(base), ".yaml"), strings.HasSuffix(strings.ToLower(base), ".yml"):
		return formats.Yaml
	default:
		return formats.FormatForPath(path)
	}
}

func formatName(format formats.Format) string {
	switch format {
	case formats.Json:
		return "json"
	case formats.Yaml:
		return "yaml"
	case formats.Dotenv:
		return "dotenv"
	default:
		return "binary"
	}
}

func lookupValue(plain []byte, format formats.Format, key string) (string, bool, error) {
	if format == formats.Dotenv {
		v, ok := dotenvValue(plain, key)
		return v, ok, nil
	}
	var doc map[string]any
	var err error
	switch format {
	case formats.Json:
		err = json.Unmarshal(plain, &doc)
	case formats.Yaml:
		err = yaml.Unmarshal(plain, &doc)
	default:
		return "", false, fmt.Errorf("unsupported format")
	}
	if err != nil {
		return "", false, err
	}
	raw, ok := doc[key]
	if !ok {
		return "", false, nil
	}
	if s, ok := raw.(string); ok {
		return s, true, nil
	}
	return fmt.Sprint(raw), true, nil
}

func dotenvValue(plain []byte, key string) (string, bool) {
	v, ok := dotenvMap(plain)[key]
	return v, ok
}

func cmdIdentity(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: sopsdeck identity create|import|key [--confirmed-backup]")
		return 1
	}
	switch args[0] {
	case "create":
		return identityCreate(args[1:], stdout, stderr, getenv)
	case "import":
		return identityImport(args[1:], stdout, stderr, getenv)
	case "key":
		return identityPrintKey(stdout, stderr, getenv)
	default:
		fmt.Fprintln(stderr, "usage: sopsdeck identity create|import|key [--confirmed-backup]")
		return 1
	}
}

func identityStateDir(getenv func(string) string, stderr io.Writer) (string, bool) {
	dir := getenv("SOPSDECK_STATE_DIR")
	if dir == "" {
		fmt.Fprintln(stderr, "identity: SOPSDECK_STATE_DIR is required")
		return "", false
	}
	return dir, true
}

func identityCreate(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if !identityConfirmed(args) {
		fmt.Fprintln(stderr, "identity create: save the private key in your password manager, then rerun with --confirmed-backup")
		return 1
	}
	if _, ok := identityStateDir(getenv, stderr); !ok {
		return 1
	}
	id, err := age.GenerateX25519Identity()
	if err != nil {
		fmt.Fprintf(stderr, "identity create: %v\n", err)
		return 1
	}
	body := "# public key: " + id.Recipient().String() + "\n" + id.String() + "\n"
	if err := putIdentity(getenv, body); err != nil {
		fmt.Fprintf(stderr, "identity create: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, id.Recipient().String())
	fmt.Fprintln(stderr, "identity: stored in the OS keychain; export SOPS_AGE_KEY_CMD='sopsdeck identity key'")
	return 0
}

func identityImport(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	_ = stdout
	var file string
	rest := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "identity import: -f requires a file")
				return 1
			}
			file = args[i]
		default:
			rest = append(rest, args[i])
		}
	}
	if file == "" {
		fmt.Fprintln(stderr, "usage: sopsdeck identity import -f FILE --confirmed-backup")
		return 1
	}
	if !identityConfirmed(rest) {
		fmt.Fprintln(stderr, "identity import: confirm the private key is in your password manager with --confirmed-backup")
		return 1
	}
	if _, ok := identityStateDir(getenv, stderr); !ok {
		return 1
	}
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(stderr, "identity import: %v\n", err)
		return 1
	}
	if _, err := age.ParseIdentities(strings.NewReader(string(data))); err != nil {
		fmt.Fprintf(stderr, "identity import: %v\n", err)
		return 1
	}
	if err := putIdentity(getenv, string(data)); err != nil {
		fmt.Fprintf(stderr, "identity import: %v\n", err)
		return 1
	}
	return 0
}

func identityPrintKey(stdout, stderr io.Writer, getenv func(string) string) int {
	body, err := getIdentity(getenv)
	if err != nil {
		fmt.Fprintf(stderr, "identity key: %v\n", err)
		return 1
	}
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	fmt.Fprint(stdout, body)
	return 0
}

func identityConfirmed(args []string) bool {
	for _, a := range args {
		if a == "--confirmed-backup" {
			return true
		}
	}
	return false
}

func setCreate(file, key, value string, stderr io.Writer, getenv func(string) string) int {
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
	if key != "" {
		branch = append(branch, sops.TreeItem{Key: key, Value: value})
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

func ageRecipientFromEnv(getenv func(string) string) (string, error) {
	if path := getenv("SOPS_AGE_KEY_FILE"); path != "" {
		return recipientFromIdentityFile(path)
	}
	if body, err := getIdentity(getenv); err == nil && strings.TrimSpace(body) != "" {
		return recipientFromIdentityReader(strings.NewReader(body))
	}
	if dir := getenv("SOPSDECK_STATE_DIR"); dir != "" {
		return recipientFromIdentityFile(filepath.Join(dir, "age.txt"))
	}
	return "", fmt.Errorf("no age identity (set SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD, or SOPSDECK_STATE_DIR)")
}

func recipientFromIdentityFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	return recipientFromIdentityReader(f)
}

func recipientFromIdentityReader(r io.Reader) (string, error) {
	ids, err := age.ParseIdentities(r)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no age identities")
	}
	id, ok := ids[0].(*age.X25519Identity)
	if !ok {
		return "", fmt.Errorf("first identity is not an age X25519 key")
	}
	return id.Recipient().String(), nil
}

func cmdCommit(args []string, stdout, stderr io.Writer) int {
	_ = stdout
	var message, file string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-m":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "commit: -m requires a message")
				return 1
			}
			message = args[i]
		case "-f":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "commit: -f requires a file")
				return 1
			}
			file = args[i]
		default:
			fmt.Fprintln(stderr, "usage: sopsdeck commit -m MESSAGE -f FILE")
			return 1
		}
	}
	if message == "" || file == "" {
		fmt.Fprintln(stderr, "usage: sopsdeck commit -m MESSAGE -f FILE")
		return 1
	}
	dir := filepath.Dir(file)
	if err := runGitCmd(dir, "add", "--", file); err != nil {
		fmt.Fprintf(stderr, "commit: %v\n", err)
		return 1
	}
	if err := runGitCmd(dir, "commit", "-m", message, "--", file); err != nil {
		fmt.Fprintf(stderr, "commit: %v\n", err)
		return 1
	}
	return 0
}

func runGitCmd(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func cmdSync(args []string, stdout, stderr io.Writer) int {
	_ = stdout
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: sopsdeck sync")
		return 1
	}
	if err := syncAt("."); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func syncAt(dir string) error {
	dirty, err := gitWorktreeDirtyAt(dir)
	if err != nil {
		return errors.New(explainSync(err))
	}
	if dirty {
		return errors.New("sync: commit local Managed File changes before Sync")
	}
	if err := runGitCmd(dir, "fetch"); err != nil {
		return errors.New(explainSync(err))
	}
	if err := runGitCmd(dir, "pull", "--ff-only"); err != nil {
		return errors.New(explainSync(err))
	}
	if err := runGitCmd(dir, "push"); err != nil {
		return errors.New(explainSync(err))
	}
	return nil
}
