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
	if unmerged, err := fileIsUnmerged(file); err != nil {
		return fmt.Errorf("review: %w", err)
	} else if unmerged {
		return writeThreeWay(file, format, stdout)
	}
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

func writeThreeWay(file string, format formats.Format, stdout io.Writer) error {
	base, err := stagePairs(file, "1", format)
	if err != nil {
		return err
	}
	ours, err := stagePairs(file, "2", format)
	if err != nil {
		return err
	}
	theirs, err := stagePairs(file, "3", format)
	if err != nil {
		return err
	}
	keys := map[string]struct{}{}
	for k := range base {
		keys[k] = struct{}{}
	}
	for k := range ours {
		keys[k] = struct{}{}
	}
	for k := range theirs {
		keys[k] = struct{}{}
	}
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		b, o, th := base[k], ours[k], theirs[k]
		if b == o && o == th {
			continue
		}
		fmt.Fprintf(stdout, "%s: base=%s ours=%s theirs=%s\n", k, threeWayValue(b), threeWayValue(o), threeWayValue(th))
	}
	return nil
}

func threeWayValue(v string) string {
	if v == "" {
		return "(missing)"
	}
	return v
}

func stagePairs(file, stage string, format formats.Format) (map[string]string, error) {
	raw, err := gitShowAt(file, ":"+stage)
	if err != nil {
		return nil, fmt.Errorf("review: leave this conflict to Git")
	}
	plain, err := decrypt.Data(raw, formatName(format))
	if err != nil {
		return nil, fmt.Errorf("review: leave this conflict to Git")
	}
	pairs, err := secretPairs(plain, format)
	if err != nil {
		return nil, fmt.Errorf("review: leave this conflict to Git")
	}
	return pairs, nil
}

func fileIsUnmerged(file string) (bool, error) {
	dir, rel, err := gitTrackedRel(file)
	if err != nil {
		return false, err
	}
	cmd := exec.Command("git", "ls-files", "-u", "--", rel)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
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
