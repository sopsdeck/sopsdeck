package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

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
	// Native SOPS can run programs whose stderr contains secrets. Do not persist it.
	if len(args) > 0 && (args[0] == "sops" || isSOPSInvocation(args)) {
		return run(args, stdin, stdout, stderr, getenv)
	}
	var captured bytes.Buffer
	logged := io.MultiWriter(stderr, &captured)
	code := run(args, stdin, stdout, logged, getenv)
	if code != 0 {
		recordError(getenv, captured.String())
	}
	return code
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 1
	}
	switch args[0] {
	case "--help", "-h":
		printUsage(stdout)
		return 0
	case "--version", "-V", "version":
		fmt.Fprintln(stdout, appver.Version)
		return 0
	}
	if args[0] == "sops" {
		return cmdSOPS(args[1:], stdin, stdout, stderr, getenv)
	}
	if isSOPSInvocation(args) {
		return cmdSOPS(args, stdin, stdout, stderr, getenv)
	}
	if code, ok := runLocal(args, stdin, stdout, stderr, getenv); ok {
		return code
	}
	if code, ok := runShared(args, stdin, stdout, stderr, getenv); ok {
		return code
	}
	fmt.Fprintf(stderr, "unknown command %q\n", args[0])
	return 1
}

func runLocal(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) (int, bool) {
	switch args[0] {
	case "get":
		return cmdGet(args[1:], stdout, stderr, getenv), true
	case "lock":
		return cmdLock(args[1:], stdout, stderr, getenv), true
	case "unlock":
		return cmdUnlock(args[1:], stdout, stderr), true
	case "status":
		return cmdFileStatus(args[1:], stdout, stderr), true
	case "copy":
		return cmdCopy(args[1:], stdin, stderr), true
	case "set":
		return cmdSet(args[1:], stdin, stdout, stderr, getenv), true
	case "del":
		return cmdDel(args[1:], stdout, stderr), true
	case "run":
		return cmdRun(args[1:], stdin, stdout, stderr, getenv), true
	case "identity":
		return cmdIdentity(args[1:], stdout, stderr, getenv), true
	case "account":
		return cmdAccount(args[1:], stdout, stderr, getenv), true
	case "robot":
		return cmdRobot(args[1:], stdout, stderr), true
	default:
		return 0, false
	}
}

func runShared(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) (int, bool) {
	switch args[0] {
	case "review":
		return cmdReview(args[1:], stdout, stderr), true
	case "history":
		return cmdHistory(args[1:], stdout, stderr), true
	case "restore":
		return cmdRestore(args[1:], stdout, stderr), true
	case "recipient":
		return cmdRecipient(args[1:], stdout, stderr, getenv), true
	case "sync":
		return cmdSyncSecrets(args[1:], stdout, stderr, getenv), true
	case "files":
		return cmdFiles(args[1:], stdout, stderr), true
	case "drive":
		return cmdDrive(args[1:], stdout, stderr, getenv), true
	case "team":
		return cmdTeam(args[1:], stdout, stderr), true
	case "scan":
		return cmdScan(args[1:], stdout, stderr), true
	case "project":
		return cmdProject(args[1:], stdout, stderr, getenv), true
	case "references", "unused", "rename":
		return runReferenceCommands(args[0], args[1:], stdout, stderr), true
	default:
		return 0, false
	}
}

func runReferenceCommands(cmd string, args []string, stdout, stderr io.Writer) int {
	switch cmd {
	case "references":
		return cmdReferences(args, stdout, stderr)
	case "unused":
		return cmdUnused(args, stdout, stderr)
	case "rename":
		return cmdRename(args, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", cmd)
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

func cmdGet(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
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
		mapping, _, _ := mappingFor(file)
		if flags.at == "" && mapping.Path != "" {
			if raw, readErr := os.ReadFile(file); readErr == nil && !isEncryptedBytes(raw) {
				plain, err = raw, nil
			}
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, explainGet(err))
		return 1
	}
	if key == "" {
		if output == "json" {
			pairs, err := plainPairs(plain, format, getenv)
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
	pairs, err := plainPairs(plain, format, getenv)
	if err != nil {
		fmt.Fprintf(stderr, "get: %v\n", err)
		return 1
	}
	value, ok := pairs[key]
	if !ok {
		fmt.Fprintf(stderr, "get: missing key %s\n", key)
		return 1
	}
	fmt.Fprintln(stdout, value)
	return 0
}

func cmdSet(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(setPositionals(args)) < 2 {
		if payload := readPaste(stdin); len(payload) > 0 {
			return applyPaste(args, payload, stdout, stderr, getenv)
		}
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
	path, err := treePath(key, format != formats.Dotenv)
	if err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	if raw, readErr := os.ReadFile(file); readErr == nil && !isEncryptedBytes(raw) {
		return setUnlocked(file, store, path, value, raw, stderr)
	}
	return setEncrypted(file, store, path, value, stderr)
}

func setPositionals(args []string) []string {
	var positionals []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
		case "--yes":
		default:
			if !strings.HasPrefix(args[i], "-") {
				positionals = append(positionals, args[i])
			}
		}
	}
	return positionals
}

func setUnlocked(file string, store sops.Store, path []interface{}, value string, raw []byte, stderr io.Writer) int {
	mapping, _, _ := mappingFor(file)
	if mapping.Path == "" {
		fmt.Fprintln(stderr, "set: not a SOPS-encrypted file")
		return 1
	}
	branches, err := loadPlainBranches(fileFormat(file), raw)
	if err != nil {
		fmt.Fprintf(stderr, "set: %v\n", err)
		return 1
	}
	branches[0], _ = branches[0].Set(path, value)
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

func setEncrypted(file string, store sops.Store, path []interface{}, value string, stderr io.Writer) int {
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
	tree.Branches[0], _ = tree.Branches[0].Set(path, value)
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
	if _, err := os.Stat(transientRunLockPath(path)); err == nil {
		return fmt.Errorf("%s is in use by sopsdeck run", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	return writeAtomicMode(path, data, 0o600)
}

func writeAtomicMode(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".sopsdeck-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(mode.Perm()); err != nil {
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
	path, err := treePath(key, format != formats.Dotenv)
	if err != nil {
		fmt.Fprintf(stderr, "del: %v\n", err)
		return 1
	}
	if raw, readErr := os.ReadFile(file); readErr == nil && !isEncryptedBytes(raw) {
		return delUnlocked(file, store, path, raw, stderr)
	}
	return delEncrypted(file, store, path, stderr)
}

func delUnlocked(file string, store sops.Store, path []interface{}, raw []byte, stderr io.Writer) int {
	mapping, _, _ := mappingFor(file)
	if mapping.Path == "" {
		fmt.Fprintln(stderr, "del: not a SOPS-encrypted file")
		return 1
	}
	branches, err := loadPlainBranches(fileFormat(file), raw)
	if err != nil {
		fmt.Fprintf(stderr, "del: %v\n", err)
		return 1
	}
	branch, err := branches[0].Unset(path)
	if err != nil {
		fmt.Fprintf(stderr, "del: %v\n", err)
		return 1
	}
	branches[0] = branch
	out, err := store.EmitPlainFile(branches)
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

func delEncrypted(file string, store sops.Store, path []interface{}, stderr io.Writer) int {
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
	branch, err := tree.Branches[0].Unset(path)
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

func parseRunFlags(args []string) (file string, noRedact bool, argv []string, usage bool) {
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
				return "", false, nil, true
			}
			file = args[i]
		case "--no-redact":
			noRedact = true
		default:
			return "", false, nil, true
		}
	}
	if dash < 0 || dash+1 >= len(args) || (file == "" && !noRedact) {
		return "", false, nil, true
	}
	return file, noRedact, args[dash+1:], false
}

func cmdRun(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) (code int) {
	file, noRedact, argv, usage := parseRunFlags(args)
	if usage {
		fmt.Fprintln(stderr, "usage: sopsdeck run [-f FILE] [--no-redact] -- CMD [ARG...]")
		return 1
	}
	// Arm SIGINT/SIGTERM handling before any plaintext can touch disk: a
	// signal arriving between transient unlock and signal.Notify would
	// otherwise kill sopsdeck with the default handler and skip the relock.
	signals := make(chan os.Signal, 4)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	expansionCtx, stopExpansion := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopExpansion()

	runner := childRunner{
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		signals: signals,
	}
	if file != "" {
		format := fileFormat(file)
		if format == formats.Dotenv {
			plain, err := decrypt.File(file, formatName(format))
			if err != nil {
				fmt.Fprintf(stderr, "run: %v\n", err)
				return 1
			}
			fileEnv, err := dotenvPairsContext(expansionCtx, plain, getenv)
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
			runner.env = childEnv
			if !noRedact {
				runner.redact = newRedactorValues(redactionPairs(file, fileEnv, true, nil, false, ""))
			}
		} else {
			encryptedKeys, encryptsAll, regex := fileEncryptionPolicy(file)
			runner.unlock = &transientUnlock{file: file}
			defer func() {
				if err := runner.unlock.close(stderr); err != nil {
					code = 1
				}
			}()
			if err := runner.unlock.open(stderr); err != nil {
				return 1
			}
			if !noRedact {
				pairs, err := plainPairs(runner.unlock.plain, format, getenv)
				if err != nil {
					fmt.Fprintf(stderr, "run: %v\n", err)
					return 1
				}
				runner.redact = newRedactorValues(redactionPairs(file, pairs, false, encryptedKeys, encryptsAll, regex))
			}
		}
	}
	return runner.run(argv)
}

// childRunner spawns the run child and reports its outcome. The zero value
// inherits the parent environment; transientUnlock supplies a decrypted
// working copy for the duration of one command. signals is the already-armed
// SIGINT/SIGTERM channel from cmdRun.
type childRunner struct {
	env     []string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	signals chan os.Signal
	unlock  *transientUnlock
	redact  *redactor
}

func (r childRunner) run(argv []string) int {
	stdout := r.outputWriter(r.stdout)
	stderr := r.outputWriter(r.stderr)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = r.stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = r.env
	configureChildCommand(cmd)
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(r.stderr, "run: %v\n", err)
		return 1
	}

	// While the child runs, forward SIGINT/SIGTERM into its process group so
	// sopsdeck stays alive long enough to relock any transiently unlocked
	// file before exiting with the same signal. The channel was armed before
	// the transient unlock, so no window exists where a signal can kill
	// sopsdeck while plaintext is on disk.
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	err := func() error {
		for {
			select {
			case sig := <-r.signals:
				forwardChildSignal(cmd, sig)
			case err := <-done:
				return err
			}
		}
	}()

	// The child is done; flush any redacted tail the streams held back.
	var flushErr error
	for _, w := range []io.Writer{stdout, stderr} {
		if f, ok := w.(interface{ Flush() error }); ok {
			if err := f.Flush(); err != nil && flushErr == nil {
				flushErr = err
			}
		}
	}
	if flushErr != nil {
		fmt.Fprintln(r.stderr, "run: could not write redacted command output")
		return 1
	}

	if err == nil {
		return 0
	}
	return childExitCode(err, r.stderr)
}

// transientUnlock decrypts a structured Managed File to its real path for
// the duration of one child command, then relocks it. The unlocked working
// copy carries no sops metadata — zero remnants.
type transientUnlock struct {
	file     string
	plain    []byte
	previous []byte
	mode     os.FileMode
	lock     string
	locked   bool
}

func (u *transientUnlock) open(stderr io.Writer) error {
	if canonical, err := filepath.EvalSymlinks(u.file); err == nil {
		u.file = canonical
	}
	u.lock = transientRunLockPath(u.file)
	lockFile, err := os.OpenFile(u.lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			err = fmt.Errorf("another run is active or a stale run lock exists at %s", u.lock)
		}
		fmt.Fprintf(stderr, "run: %v\n", err)
		return err
	}
	u.locked = true
	if _, err = fmt.Fprintf(lockFile, "%d\n", os.Getpid()); err == nil {
		err = lockFile.Sync()
	}
	closeErr := lockFile.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		fmt.Fprintf(stderr, "run: could not create run lock: %v\n", err)
		return err
	}

	info, err := os.Stat(u.file)
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return err
	}
	raw, err := os.ReadFile(u.file)
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return err
	}
	u.mode = info.Mode().Perm()
	if !isEncryptedBytes(raw) {
		u.plain = raw
		return nil
	}
	plain, err := decrypt.File(u.file, formatName(fileFormat(u.file)))
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return err
	}
	if err := writeAtomicMode(u.file, plain, 0o600); err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return err
	}
	u.plain = plain
	u.previous = raw
	return nil
}

func transientRunLockPath(file string) string {
	if canonical, err := filepath.EvalSymlinks(file); err == nil {
		file = canonical
	}
	return file + ".sopsdeck-run.lock"
}

func (u *transientUnlock) close(stderr io.Writer) error {
	if u.locked {
		defer func() { _ = os.Remove(u.lock) }()
	}
	if u.previous == nil {
		return nil
	}
	current, err := os.ReadFile(u.file)
	if err == nil && !bytes.Equal(current, u.plain) {
		err = fmt.Errorf("file changed during run")
	}
	if err == nil {
		err = writeAtomicMode(u.file, u.previous, u.mode)
	}
	if err != nil {
		fmt.Fprintf(stderr, "run: could not restore encrypted file %s: %v; inspect it and recover with `sopsdeck lock -f %s` if needed\n", u.file, err, u.file)
		return err
	}
	return nil
}

func plainEnv(plain []byte, format formats.Format) (map[string]string, error) {
	if format == formats.Dotenv {
		return parseDotenvMap(plain)
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

func plainPairs(plain []byte, format formats.Format, getenv func(string) string) (map[string]string, error) {
	if format == formats.Dotenv {
		return dotenvPairs(plain, getenv)
	}
	var doc any
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
	flattenPairs(out, "", doc)
	return out, nil
}

// dotenvPairs parses a dotenv file and resolves live computed values in file
// order, so earlier keys feed later ones. Single-quoted values stay literal
// (dotenvx's expansion opt-out).
func dotenvPairs(plain []byte, getenv func(string) string) (map[string]string, error) {
	return dotenvPairsContext(context.Background(), plain, getenv)
}

func dotenvPairsContext(ctx context.Context, plain []byte, getenv func(string) string) (map[string]string, error) {
	branches, single, err := parseDotenvQuoted(plain)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, item := range branches[0] {
		key, ok := item.Key.(string)
		if !ok {
			continue
		}
		value := fmt.Sprint(item.Value)
		if !single[key] {
			value, err = expandDotenvValue(ctx, value, out, getenv)
			if err != nil {
				return nil, err
			}
		}
		out[key] = value
	}
	return out, nil
}

func flattenPairs(out map[string]string, prefix string, value any) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			flattenPairs(out, path, child)
		}
	case []any:
		for i, child := range value {
			flattenPairs(out, fmt.Sprintf("%s[%d]", prefix, i), child)
		}
	case nil:
		out[prefix] = ""
	default:
		out[prefix] = fmt.Sprint(value)
	}
}

func fileFormat(path string) formats.Format {
	if mapping, _, _ := mappingFor(path); mapping.Format != "" {
		return formats.FormatFromString(mapping.Format)
	}
	data, err := os.ReadFile(path)
	if err == nil && !isEncryptedBytes(data) {
		return detectFileFormat(data)
	}
	base := filepath.Base(path)
	switch {
	case base == ".env" || strings.HasPrefix(base, ".env.") || strings.HasSuffix(strings.ToLower(base), ".env"):
		return formats.Dotenv
	case strings.HasSuffix(strings.ToLower(base), ".json"):
		return formats.Json
	case strings.HasSuffix(strings.ToLower(base), ".yaml"), strings.HasSuffix(strings.ToLower(base), ".yml"):
		return formats.Yaml
	}
	if os.IsNotExist(err) {
		return formats.Dotenv
	}
	return formats.Binary
}

func detectFileFormat(data []byte) formats.Format {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return formats.Dotenv
	}
	if json.Valid(trimmed) {
		return formats.Json
	}
	if bytes.Contains(trimmed, []byte("=")) {
		return formats.Dotenv
	}
	if _, err := parseDotenv(trimmed); err == nil {
		return formats.Dotenv
	}
	var document any
	if yaml.Unmarshal(trimmed, &document) == nil {
		switch document.(type) {
		case map[string]any, []any:
			return formats.Yaml
		}
	}
	return formats.Binary
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

func cmdIdentity(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: sopsdeck identity create|import|key|remove [--confirmed-backup]")
		return 1
	}
	switch args[0] {
	case "create":
		return identityCreate(args[1:], stdout, stderr, getenv)
	case "import":
		return identityImport(args[1:], stdout, stderr, getenv)
	case "key":
		return identityPrintKey(stdout, stderr, getenv)
	case "remove":
		return identityRemove(args[1:], stderr, getenv)
	default:
		fmt.Fprintln(stderr, "usage: sopsdeck identity create|import|key|remove [--confirmed-backup]")
		return 1
	}
}

func identityCreate(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if !identityConfirmed(args) {
		fmt.Fprintln(stderr, "identity create: save the private key in your password manager, then rerun with --confirmed-backup")
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

func identityRemove(args []string, stderr io.Writer, getenv func(string) string) int {
	if len(args) != 1 || args[0] != "--yes" {
		fmt.Fprintln(stderr, "usage: sopsdeck identity remove --yes")
		return 1
	}
	if err := deleteIdentity(getenv); err != nil {
		fmt.Fprintf(stderr, "identity remove: %v\n", err)
		return 1
	}
	fmt.Fprintln(stderr, "identity remove: removed this machine's keychain identity; encrypted files and Recipient access are unchanged")
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
	body, err := ageIdentityFromEnv(getenv)
	if err != nil {
		return "", err
	}
	return recipientFromIdentityReader(strings.NewReader(body))
}

func ageIdentityFromEnv(getenv func(string) string) (string, error) {
	if body := getenv("SOPS_AGE_KEY"); body != "" {
		return body, nil
	}
	if path := getenv("SOPS_AGE_KEY_FILE"); path != "" {
		body, err := os.ReadFile(path)
		return string(body), err
	}
	if command := getenv("SOPS_AGE_KEY_CMD"); command != "" {
		body, err := exec.Command("sh", "-c", command).Output()
		if err != nil {
			return "", fmt.Errorf("age identity command failed")
		}
		return string(body), nil
	}
	if body, err := getIdentity(getenv); err == nil && strings.TrimSpace(body) != "" {
		return body, nil
	}
	if dir := getenv("SOPSDECK_STATE_DIR"); dir != "" {
		body, err := os.ReadFile(filepath.Join(dir, "age.txt"))
		return string(body), err
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

func gitCLIArgs(args []string) []string {
	out := []string{"-c", "commit.gpgsign=false"}
	if len(args) > 0 && args[0] == "commit" {
		return append(append(out, "commit", "--no-gpg-sign"), args[1:]...)
	}
	return append(out, args...)
}

func runGitCmd(dir string, args ...string) error {
	cmd := exec.Command("git", gitCLIArgs(args)...)
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
