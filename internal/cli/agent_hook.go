package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	agentHookMaxInput = 16 << 20
	agentHookDenyText = "Managed Files are protected. Ask the project owner to run the application through sopsdeck run."
)

type claudeAgentHookInput struct {
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
}

type cursorAgentHookInput struct {
	FilePath string `json:"file_path"`
}

type claudeAgentHookOutput struct {
	HookSpecificOutput *claudeAgentHookDecision `json:"hookSpecificOutput,omitempty"`
}

type claudeAgentHookDecision struct {
	HookEventName      string `json:"hookEventName"`
	PermissionDecision string `json:"permissionDecision"`
	Reason             string `json:"reason"`
}

type cursorAgentHookOutput struct {
	Permission  string `json:"permission"`
	UserMessage string `json:"user_message,omitempty"`
}

// cmdAgentHook adapts native pre-read hook protocols. It intentionally reads
// only the target path; fields such as Cursor's file content are discarded.
func cmdAgentHook(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) != 1 || (args[0] != "claude" && args[0] != "cursor") {
		fmt.Fprintln(stderr, "usage: sopsdeck agent-hook <claude|cursor>")
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		writeAgentHookDecision(args[0], false, stdout)
		return 0
	}
	writeAgentHookDecision(args[0], agentHookAllowsPath(args[0], stdin, cwd), stdout)
	return 0
}

func agentHookAllowsPath(host string, input io.Reader, cwd string) bool {
	limited := io.LimitReader(input, agentHookMaxInput+1)
	decoder := json.NewDecoder(limited)
	var path string
	switch host {
	case "claude":
		var event claudeAgentHookInput
		if err := decodeAgentHookJSON(decoder, &event); err != nil || event.ToolName != "Read" {
			return false
		}
		path = event.ToolInput.FilePath
	case "cursor":
		var event cursorAgentHookInput
		if err := decodeAgentHookJSON(decoder, &event); err != nil {
			return false
		}
		path = event.FilePath
	default:
		return false
	}
	if strings.TrimSpace(path) == "" {
		return false
	}

	blocked, err := isManagedAgentPath(path, cwd)
	return err == nil && !blocked
}

func decodeAgentHookJSON(decoder *json.Decoder, dst any) error {
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple hook payloads")
		}
		return err
	}
	return nil
}

func isManagedAgentPath(path, cwd string) (bool, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	requested, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	requestedReal := canonicalAgentPath(requested)

	// Search from both the spelling supplied by the host and its resolved
	// target. This covers symlink aliases on either side of a Project root.
	roots := map[string]bool{}
	for _, start := range []string{filepath.Dir(requested), filepath.Dir(requestedReal)} {
		root, manifestPath := findManifest(start)
		if manifestPath == "" || roots[root] {
			continue
		}
		roots[root] = true
		manifest, err := loadManifest(manifestPath)
		if err != nil {
			return false, err
		}
		for _, entry := range manifest.ManagedFile {
			if filepath.IsAbs(entry.Path) {
				return false, fmt.Errorf("invalid managed path")
			}
			managed := filepath.Clean(filepath.Join(root, filepath.FromSlash(entry.Path)))
			rel, err := filepath.Rel(root, managed)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return false, fmt.Errorf("invalid managed path")
			}
			managedReal := canonicalAgentPath(managed)
			if sameAgentPath(requested, managed) || sameAgentPath(requestedReal, managedReal) {
				return true, nil
			}
		}
	}
	return false, nil
}

func canonicalAgentPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	return filepath.Clean(path)
}

func sameAgentPath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func writeAgentHookDecision(host string, allow bool, stdout io.Writer) {
	var output any
	if host == "claude" {
		if allow {
			output = claudeAgentHookOutput{}
		} else {
			output = claudeAgentHookOutput{HookSpecificOutput: &claudeAgentHookDecision{
				HookEventName:      "PreToolUse",
				PermissionDecision: "deny",
				Reason:             agentHookDenyText,
			}}
		}
	} else {
		permission := "allow"
		message := ""
		if !allow {
			permission = "deny"
			message = agentHookDenyText
		}
		output = cursorAgentHookOutput{Permission: permission, UserMessage: message}
	}
	_ = json.NewEncoder(stdout).Encode(output)
}
