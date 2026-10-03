package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// expandVar re interpolates ${VAR} and $VAR expressions the way dotenvx does:
// file values first, then the real environment; undefined refs expand empty;
// \$ escapes a literal dollar. One layer — a resolved value's own ${...}
// content is not re-expanded. The leading \\\$ alternative matches an escaped
// dollar followed by anything (dotenvx's lookbehind + post-unescape pass).
var expandVar = regexp.MustCompile(`\\\$|\\?\$\{([^{}]+)\}|\\?\$([A-Za-z_][A-Za-z0-9_]*)`)

const commandSubstitutionTimeout = 30 * time.Second

func expandDotenvValue(ctx context.Context, value string, running map[string]string, getenv func(string) string) (string, error) {
	if getenv == nil {
		getenv = defaultGetenv
	}
	value, err := expandCommands(ctx, value, running)
	if err != nil {
		return "", fmt.Errorf("command substitution failed: %w", err)
	}
	return expandString(value, running, getenv), nil
}

// expandCommands evaluates each substitution present in the stored value once.
// Command output is never scanned for another substitution.
func expandCommands(ctx context.Context, value string, running map[string]string) (string, error) {
	var result strings.Builder
	for offset := 0; offset < len(value); {
		relative := strings.Index(value[offset:], "$(")
		if relative == -1 {
			result.WriteString(value[offset:])
			break
		}
		start := offset + relative
		end, ok := scanShellCommand(value, start+2)
		if start > 0 && value[start-1] == '\\' {
			result.WriteString(value[offset : start-1])
			if !ok {
				result.WriteString(value[start:])
				break
			}
			result.WriteString(value[start:end])
			offset = end
			continue
		}
		result.WriteString(value[offset:start])
		if !ok {
			result.WriteString(value[start:])
			break // unbalanced; leave as written
		}
		command := value[start+2 : end-1]
		commandCtx, cancel := context.WithTimeout(ctx, commandSubstitutionTimeout)
		output, err := runSubstitution(commandCtx, command, running)
		cancel()
		if err != nil {
			return "", err
		}
		result.WriteString(output)
		offset = end
	}
	return result.String(), nil
}

// scanShellCommand returns the index just past the closing paren of the
// $(...) group opened at depth, tracking quotes and nesting like dotenvx.
func scanShellCommand(value string, depth int) (int, bool) {
	var quote byte
	for i := depth; i < len(value); i++ {
		c := value[i]
		switch {
		case quote != 0:
			switch c {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '(':
			next, ok := scanShellCommand(value, i+1)
			if !ok {
				return 0, false
			}
			i = next - 1
		case c == ')':
			return i + 1, true
		}
	}
	return 0, false
}

// runSubstitution requires POSIX sh, exposing resolved file keys over the
// process environment, and trims trailing newlines like shell substitution.
func runSubstitution(ctx context.Context, command string, running map[string]string) (string, error) {
	env := os.Environ()
	for k, v := range running {
		env = append(env, k+"="+v)
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	configureChildCommand(cmd)
	cmd.Cancel = func() error { return cancelChildCommand(cmd) }
	cmd.Env = env
	cmd.WaitDelay = time.Second
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func expandString(value string, running map[string]string, getenv func(string) string) string {
	return expandVar.ReplaceAllStringFunc(value, func(match string) string {
		if strings.HasPrefix(match, "\\") {
			return match[1:] // \$ → literal $
		}
		name := match[1:]
		if strings.HasPrefix(name, "{") {
			name = name[1 : len(name)-1]
		}
		if v, ok := running[name]; ok {
			return v
		}
		return getenv(name)
	})
}

func defaultGetenv(key string) string {
	return ""
}
