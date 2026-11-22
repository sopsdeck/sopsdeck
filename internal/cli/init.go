package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/zalando/go-keyring"
)

func cmdInit(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	files, yes, done := initArgs(args, stdout, stderr)
	if done != 0 {
		return done - 1
	}

	root, err := canonicalProjectFolder(".")
	if err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 1
	}
	if owner, _ := findManifest(root); owner != "" {
		fmt.Fprintf(stderr, "init: this folder already belongs to Project %s\n", owner)
		return 1
	}
	files, ok := initFiles(root, files, stderr)
	if !ok {
		return 1
	}

	fmt.Fprintf(stdout, "Project: %s\nFiles to encrypt: %s\nGit ignore exceptions may be added for encrypted files.\n", root, strings.Join(files, ", "))
	reader := bufio.NewReader(stdin)
	if !yes && !confirmInit(reader, stdout, "Encrypt these files and initialize the project? [y/N] ") {
		fmt.Fprintln(stderr, "init: cancelled")
		return 1
	}
	if !ensureInitIdentity(reader, stdout, stderr, getenv, yes) {
		return 1
	}
	projectArgs := []string{root}
	for _, file := range files {
		projectArgs = append(projectArgs, "--file", file)
	}
	if code := initProject(projectArgs, stdout, stderr, getenv); code != 0 {
		return code
	}
	fmt.Fprintln(stdout, "Next: run your app with sopsdeck run -- <command> and commit the encrypted files and .sopsdeck.toml.")
	if getenv("SOPS_AGE_KEY") == "" && getenv("SOPS_AGE_KEY_FILE") == "" && getenv("SOPS_AGE_KEY_CMD") == "" {
		fmt.Fprintln(stdout, "For CLI decryption, run: export SOPS_AGE_KEY_CMD='npx --no-install sopsdeck identity key'")
	}
	fmt.Fprintln(stdout, "Keep managed files encrypted before committing. Install the optional protection hook with sopsdeck scan --install.")
	return 0
}

func initFiles(root string, files []string, stderr io.Writer) ([]string, bool) {
	if len(files) == 0 {
		entries, err := os.ReadDir(root)
		if err != nil {
			fmt.Fprintf(stderr, "init: %v\n", err)
			return nil, false
		}
		for _, entry := range entries {
			name := entry.Name()
			lower := strings.ToLower(name)
			if entry.Type().IsRegular() && (name == ".env" || strings.HasPrefix(name, ".env.")) && !strings.Contains(lower, "example") && !strings.Contains(lower, "sample") && !strings.Contains(lower, "template") {
				files = append(files, name)
			}
		}
	}
	if len(files) == 0 {
		fmt.Fprintln(stderr, "init: no dotenv files found; use --file PATH to choose a file")
		return nil, false
	}
	for _, file := range files {
		path, _, err := projectPath(root, file)
		if err != nil {
			fmt.Fprintf(stderr, "init: %v\n", err)
			return nil, false
		}
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "init: %v\n", err)
			return nil, false
		}
		if isEncryptedBytes(data) {
			fmt.Fprintf(stderr, "init: %s is already encrypted; use sopsdeck project init to add it explicitly\n", file)
			return nil, false
		}
	}
	return files, true
}

func initArgs(args []string, stdout, stderr io.Writer) ([]string, bool, int) {
	var files []string
	yes := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--yes":
			yes = true
		case "--file", "-f":
			i++
			if i == len(args) {
				fmt.Fprintln(stderr, "init: --file requires a path")
				return nil, false, 2
			}
			files = append(files, args[i])
		case "--help", "-h":
			fmt.Fprintln(stdout, "usage: sopsdeck init [--file PATH]... [--yes]")
			return nil, false, 1
		default:
			fmt.Fprintf(stderr, "init: unknown argument %s\n", args[i])
			return nil, false, 2
		}
	}
	return files, yes, 0
}

func ensureInitIdentity(reader *bufio.Reader, stdout, stderr io.Writer, getenv func(string) string, yes bool) bool {
	if _, err := ageRecipientFromEnv(getenv); err != nil {
		if getenv("SOPS_AGE_KEY") != "" || getenv("SOPS_AGE_KEY_FILE") != "" || getenv("SOPS_AGE_KEY_CMD") != "" {
			fmt.Fprintf(stderr, "init: invalid configured identity: %v\n", err)
			return false
		}
		if _, keyErr := getIdentity(getenv); !errors.Is(keyErr, keyring.ErrNotFound) && !os.IsNotExist(keyErr) {
			fmt.Fprintf(stderr, "init: cannot use or check existing identity: %v\n", err)
			return false
		}
		if yes {
			fmt.Fprintln(stderr, "init: create and back up an identity first using sopsdeck .; --yes cannot confirm a private-key backup")
			return false
		}
		identity, keyErr := age.GenerateX25519Identity()
		if keyErr != nil {
			fmt.Fprintf(stderr, "init: %v\n", keyErr)
			return false
		}
		body := "# public key: " + identity.Recipient().String() + "\n" + identity.String() + "\n"
		fmt.Fprintf(stdout, "Save this private key in your password manager; anyone with it can decrypt your secrets:\n\n%s\n", body)
		if !confirmInit(reader, stdout, "Have you saved the private key? [y/N] ") {
			fmt.Fprintln(stderr, "init: identity was not saved; cancelled")
			return false
		}
		if err := putIdentity(getenv, body); err != nil {
			fmt.Fprintf(stderr, "init: save identity: %v\n", err)
			return false
		}
	}
	return true
}

func confirmInit(reader *bufio.Reader, stdout io.Writer, prompt string) bool {
	fmt.Fprint(stdout, prompt)
	answer, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}
