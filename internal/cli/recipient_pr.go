package cli

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"filippo.io/age"
)

type recipientPRFlags struct {
	name  string
	pub   string
	files []string
	all   bool
}

func recipientRequest(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if getenv != nil {
		restore := applyProcessEnv(getenv)
		defer restore()
	}
	flags, errMsg := parseRecipientPRFlags("request", args)
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
		return 1
	}
	root, files, err := recipientPRFiles(flags)
	if err != nil {
		fmt.Fprintf(stderr, "recipient request: %v\n", err)
		return 1
	}
	title := "Request Sopsdeck access for " + flags.name
	body := recipientPRBody("Access request", flags.name, flags.pub, files)
	if err := openRecipientPR(root, "sopsdeck/request-"+slug(flags.name), title, body, nil, stdout); err != nil {
		fmt.Fprintf(stderr, "recipient request: %v\n", err)
		return 1
	}
	return 0
}

func recipientGrant(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if getenv != nil {
		restore := applyProcessEnv(getenv)
		defer restore()
	}
	flags, errMsg := parseRecipientPRFlags("grant", args)
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
		return 1
	}
	root, files, err := recipientPRFiles(flags)
	if err != nil {
		fmt.Fprintf(stderr, "recipient grant: %v\n", err)
		return 1
	}
	title := "Grant " + flags.name + " access"
	body := recipientPRBody("Re-encrypt Managed Files for Access", flags.name, flags.pub, files)
	mutate := func() error {
		for _, file := range files {
			var addErr bytes.Buffer
			if code := recipientAdd([]string{flags.pub, "-f", filepath.Join(root, filepath.FromSlash(file))}, &addErr, nil); code != 0 {
				return fmt.Errorf("%s", strings.TrimSpace(addErr.String()))
			}
		}
		args := append([]string{"add", "--"}, files...)
		return runGitCmd(root, args...)
	}
	if err := openRecipientPR(root, "sopsdeck/access-"+slug(flags.name), title, body, mutate, stdout); err != nil {
		fmt.Fprintf(stderr, "recipient grant: %v\n", err)
		return 1
	}
	return 0
}

func parseRecipientPRFlags(verb string, args []string) (recipientPRFlags, string) {
	var flags recipientPRFlags
	usage := "usage: sopsdeck recipient " + verb + " AGE1... --name NAME (-f FILE... | --all)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			i++
			if i >= len(args) {
				return recipientPRFlags{}, "recipient " + verb + ": --name requires a value"
			}
			flags.name = strings.TrimSpace(args[i])
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				return recipientPRFlags{}, "recipient " + verb + ": -f requires a file"
			}
			flags.files = append(flags.files, args[i])
		case "--all":
			flags.all = true
		default:
			if strings.HasPrefix(args[i], "-") || flags.pub != "" {
				return recipientPRFlags{}, usage
			}
			flags.pub = args[i]
		}
	}
	if flags.name == "" || flags.pub == "" || flags.all == (len(flags.files) > 0) {
		return recipientPRFlags{}, usage
	}
	if _, err := age.ParseX25519Recipient(flags.pub); err != nil {
		return recipientPRFlags{}, "recipient " + verb + ": invalid Age public key"
	}
	return flags, ""
}

func recipientPRFiles(flags recipientPRFlags) (string, []string, error) {
	start := "."
	if len(flags.files) > 0 {
		start = filepath.Dir(flags.files[0])
	}
	root, err := gitOutput(start, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", nil, err
	}
	files := append([]string(nil), flags.files...)
	if flags.all {
		manifestPath := filepath.Join(root, ".sopsdeck.toml")
		manifest, err := loadManifest(manifestPath)
		if err != nil {
			return "", nil, err
		}
		for _, managed := range manifest.ManagedFile {
			files = append(files, managed.Path)
		}
	}
	rels := make([]string, 0, len(files))
	for _, file := range files {
		if !filepath.IsAbs(file) {
			file = filepath.Join(root, file)
		}
		file, err = filepath.EvalSymlinks(file)
		if err != nil {
			return "", nil, err
		}
		rel, err := filepath.Rel(root, file)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", nil, fmt.Errorf("managed file is outside the project: %s", file)
		}
		rels = append(rels, filepath.ToSlash(rel))
	}
	if len(rels) == 0 {
		return "", nil, fmt.Errorf("project has no managed files")
	}
	return root, rels, nil
}

func recipientPRBody(kind, name, pub string, files []string) string {
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\nName: %s\nPublic key: %s\n\nManaged Files:\n", kind, name, pub)
	for _, file := range files {
		fmt.Fprintf(&body, "- %s\n", file)
	}
	return body.String()
}

func openRecipientPR(root, branch, title, body string, mutate func() error, stdout io.Writer) (err error) {
	status, err := gitOutput(root, "status", "--porcelain")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("commit or stash local changes before opening a PR")
	}
	original, err := gitOutput(root, "branch", "--show-current")
	if err != nil || original == "" {
		return fmt.Errorf("current Git branch is required")
	}
	if err := runGitCmd(root, "switch", "-c", branch); err != nil {
		return err
	}
	defer func() {
		if switchErr := runGitCmd(root, "switch", original); err == nil && switchErr != nil {
			err = switchErr
		}
	}()
	if mutate != nil {
		if err := mutate(); err != nil {
			return err
		}
	}
	commitArgs := []string{"commit", "--allow-empty", "-m", title}
	if err := runGitCmd(root, commitArgs...); err != nil {
		return err
	}
	if err := runGitCmd(root, "push", "-u", "origin", branch); err != nil {
		return err
	}
	cmd := exec.Command("gh", "pr", "create", "--head", branch, "--title", title, "--body", body)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh pr create: %s", strings.TrimSpace(string(output)))
	}
	if _, err := stdout.Write(output); err != nil {
		return err
	}
	return nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func slug(value string) string {
	var out strings.Builder
	dash := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			dash = false
		} else if out.Len() > 0 && !dash {
			out.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(out.String(), "-")
}
