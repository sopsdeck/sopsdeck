package cli

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getsops/sops/v3/cmd/sops/formats"
	"github.com/getsops/sops/v3/decrypt"
)

func cmdReview(args []string, stdout, stderr io.Writer) int {
	file := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--env-file":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "review: -f requires a file")
				return 1
			}
			file = args[i]
		default:
			fmt.Fprintln(stderr, "usage: sopsdeck review -f FILE")
			return 1
		}
	}
	if file == "" {
		fmt.Fprintln(stderr, "usage: sopsdeck review -f FILE")
		return 1
	}
	if err := writeReview(file, stdout); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func writeReview(file string, stdout io.Writer) error {
	format := fileFormat(file)
	workPlain, err := decrypt.File(file, formatName(format))
	if err != nil {
		return fmt.Errorf("%s", explainReview(err))
	}
	work, err := secretPairs(workPlain, format)
	if err != nil {
		return fmt.Errorf("review: %w", err)
	}
	head, err := headSecretPairs(file, format)
	if err != nil {
		return fmt.Errorf("review: %w", err)
	}
	keys := make([]string, 0, len(work)+len(head))
	seen := map[string]bool{}
	for k := range work {
		keys = append(keys, k)
		seen[k] = true
	}
	for k := range head {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		oldVal, had := head[k]
		newVal, has := work[k]
		switch {
		case had && has && oldVal == newVal:
			continue
		case !had && has:
			fmt.Fprintf(stdout, "%s: (added) %s\n", k, newVal)
		case had && !has:
			fmt.Fprintf(stdout, "%s: (removed) %s\n", k, oldVal)
		default:
			fmt.Fprintf(stdout, "%s: %s -> %s\n", k, oldVal, newVal)
		}
	}
	return nil
}

func headSecretPairs(file string, format formats.Format) (map[string]string, error) {
	raw, err := gitShowHEAD(file)
	if err != nil {
		return map[string]string{}, nil
	}
	plain, err := decrypt.Data(raw, formatName(format))
	if err != nil {
		return nil, err
	}
	return secretPairs(plain, format)
}

func gitShowAt(file, rev string) ([]byte, error) {
	dir, rel, err := gitTrackedRel(file)
	if err != nil {
		return nil, err
	}
	show := exec.Command("git", "show", rev+":"+rel)
	show.Dir = dir
	out, err := show.Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func gitShowHEAD(file string) ([]byte, error) {
	return gitShowAt(file, "HEAD")
}

func gitTrackedRel(file string) (dir, rel string, err error) {
	dir = filepath.Dir(file)
	cmd := exec.Command("git", "rev-parse", "--show-prefix")
	cmd.Dir = dir
	prefixOut, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	rel = filepath.ToSlash(strings.TrimSpace(string(prefixOut)) + filepath.Base(file))
	return dir, rel, nil
}

func secretPairs(plain []byte, format formats.Format) (map[string]string, error) {
	pairs, err := plainEnv(plain, format)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(pairs))
	for k, v := range pairs {
		if k == "sops" || strings.HasPrefix(k, "sops_") {
			continue
		}
		out[k] = v
	}
	return out, nil
}
