package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAgentHookBlocksManagedFileForClaudeAndCursor(t *testing.T) {
	root, managed := agentHookProject(t)
	inputs := map[string]string{
		"claude": `{"tool_name":"Read","tool_input":{"file_path":` + mustJSON(t, managed) + `}}`,
		"cursor": `{"file_path":` + mustJSON(t, managed) + `,"content":"HOOK_FAKE_SECRET_2718"}`,
	}
	for host, input := range inputs {
		t.Run(host, func(t *testing.T) {
			if agentHookAllowsPath(host, strings.NewReader(input), root) {
				t.Fatal("managed file read was allowed")
			}
			var output bytes.Buffer
			writeAgentHookDecision(host, false, &output)
			if strings.Contains(output.String(), "HOOK_FAKE_SECRET_2718") {
				t.Fatalf("hook output leaked input: %q", output.String())
			}
			assertAgentHookDenied(t, host, output.Bytes())
		})
	}
}

func TestAgentHookAllowsUnmanagedPath(t *testing.T) {
	root, _ := agentHookProject(t)
	input := `{"file_path":"src/main.go","content":"HOOK_FAKE_SECRET_2718"}`
	if !agentHookAllowsPath("cursor", strings.NewReader(input), root) {
		t.Fatal("unmanaged path was denied")
	}
	var output bytes.Buffer
	writeAgentHookDecision("cursor", true, &output)
	var got cursorAgentHookOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Permission != "allow" || got.UserMessage != "" {
		t.Fatalf("output=%+v", got)
	}
}

func TestAgentHookFailsClosedWithoutLeakingMalformedPayload(t *testing.T) {
	root, _ := agentHookProject(t)
	for _, input := range []string{
		`{"file_path":`,
		`{"file_path":"src/main.go"} {"content":"HOOK_FAKE_SECRET_2718"}`,
		`{"content":"HOOK_FAKE_SECRET_2718"}`,
	} {
		if agentHookAllowsPath("cursor", strings.NewReader(input), root) {
			t.Fatalf("malformed or incomplete payload was allowed: %q", input)
		}
		var output bytes.Buffer
		writeAgentHookDecision("cursor", false, &output)
		if strings.Contains(output.String(), "HOOK_FAKE_SECRET_2718") {
			t.Fatalf("hook output leaked input: %q", output.String())
		}
		assertAgentHookDenied(t, "cursor", output.Bytes())
	}
}

func TestAgentHookDeniesInvalidManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".sopsdeck.toml"), []byte("[[managed_file]]\npath = \"../escape.env\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := `{"file_path":"src/main.go"}`
	if agentHookAllowsPath("cursor", strings.NewReader(input), root) {
		t.Fatal("invalid manifest was allowed")
	}
}

func TestAgentHookMatchesSymlinkAliases(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional Windows privileges")
	}
	root, managed := agentHookProject(t)
	alias := filepath.Join(root, "alias.env")
	if err := os.Symlink(managed, alias); err != nil {
		t.Fatal(err)
	}
	if agentHookAllowsPath("cursor", strings.NewReader(`{"file_path":`+mustJSON(t, alias)+`}`), root) {
		t.Fatal("symlink alias to a managed file was allowed")
	}
}

func TestClaudeAgentHookRequiresReadEvent(t *testing.T) {
	root, managed := agentHookProject(t)
	input := `{"tool_name":"Bash","tool_input":{"file_path":` + mustJSON(t, managed) + `}}`
	if agentHookAllowsPath("claude", strings.NewReader(input), root) {
		t.Fatal("unexpected Claude tool was allowed")
	}
}

func TestAgentProtectionExamplesAreFailClosedAndUseRun(t *testing.T) {
	root := repoRoot(t)
	var claude struct {
		Hooks struct {
			PreToolUse []struct {
				Matcher string `json:"matcher"`
				Hooks   []struct {
					Command string `json:"command"`
					Timeout int    `json:"timeout"`
				} `json:"hooks"`
			} `json:"PreToolUse"`
		} `json:"hooks"`
		Sandbox struct {
			Enabled                  bool `json:"enabled"`
			AllowUnsandboxedCommands bool `json:"allowUnsandboxedCommands"`
		} `json:"sandbox"`
	}
	readAgentExampleJSON(t, filepath.Join(root, "examples", "agent-protection", ".claude", "settings.json"), &claude)
	if len(claude.Hooks.PreToolUse) != 1 || claude.Hooks.PreToolUse[0].Matcher != "Read" ||
		len(claude.Hooks.PreToolUse[0].Hooks) != 1 || claude.Hooks.PreToolUse[0].Hooks[0].Timeout == 0 ||
		!strings.Contains(claude.Hooks.PreToolUse[0].Hooks[0].Command, "|| exit 2") ||
		!claude.Sandbox.Enabled || claude.Sandbox.AllowUnsandboxedCommands {
		t.Fatalf("Claude protection example is incomplete: %+v", claude)
	}

	var cursor struct {
		Version int `json:"version"`
		Hooks   struct {
			BeforeReadFile []struct {
				FailClosed bool `json:"failClosed"`
				Timeout    int  `json:"timeout"`
			} `json:"beforeReadFile"`
			BeforeTabFileRead []struct {
				FailClosed bool `json:"failClosed"`
				Timeout    int  `json:"timeout"`
			} `json:"beforeTabFileRead"`
		} `json:"hooks"`
	}
	readAgentExampleJSON(t, filepath.Join(root, "examples", "agent-protection", ".cursor", "hooks.json"), &cursor)
	if cursor.Version != 1 || len(cursor.Hooks.BeforeReadFile) != 1 || len(cursor.Hooks.BeforeTabFileRead) != 1 ||
		!cursor.Hooks.BeforeReadFile[0].FailClosed || !cursor.Hooks.BeforeTabFileRead[0].FailClosed ||
		cursor.Hooks.BeforeReadFile[0].Timeout == 0 || cursor.Hooks.BeforeTabFileRead[0].Timeout == 0 {
		t.Fatalf("Cursor protection example is incomplete: %+v", cursor)
	}

	var task struct {
		Tasks []struct {
			Type    string   `json:"type"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"tasks"`
	}
	readAgentExampleJSON(t, filepath.Join(root, "examples", "agent-protection", ".vscode", "tasks.json"), &task)
	if len(task.Tasks) != 1 || task.Tasks[0].Type != "process" || task.Tasks[0].Command != "sopsdeck" ||
		len(task.Tasks[0].Args) < 5 || task.Tasks[0].Args[0] != "run" || task.Tasks[0].Args[1] != "-f" {
		t.Fatalf("application task does not use process arguments through run: %+v", task.Tasks)
	}
	for _, arg := range task.Tasks[0].Args {
		if arg == "--no-redact" {
			t.Fatal("application task disables redaction")
		}
	}
}

func agentHookProject(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(root, "secrets", "production values.env")
	if err := os.MkdirAll(filepath.Dir(managed), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managed, []byte("TOKEN=HOOK_FAKE_SECRET_2718\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := "[[managed_file]]\npath = \"secrets/production values.env\"\n"
	if err := os.WriteFile(filepath.Join(root, ".sopsdeck.toml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, managed
}

func assertAgentHookDenied(t *testing.T, host string, output []byte) {
	t.Helper()
	if host == "claude" {
		var got claudeAgentHookOutput
		if err := json.Unmarshal(output, &got); err != nil {
			t.Fatal(err)
		}
		if got.HookSpecificOutput == nil || got.HookSpecificOutput.HookEventName != "PreToolUse" || got.HookSpecificOutput.PermissionDecision != "deny" {
			t.Fatalf("output=%s", output)
		}
		return
	}
	var got cursorAgentHookOutput
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatal(err)
	}
	if got.Permission != "deny" || got.UserMessage != agentHookDenyText {
		t.Fatalf("output=%+v", got)
	}
}

func mustJSON(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func readAgentExampleJSON(t *testing.T, path string, dst any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatalf("decode %s: %v", filepath.Base(path), err)
	}
}
