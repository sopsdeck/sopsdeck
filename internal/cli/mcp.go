package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/decrypt"

	"sopsdeck/internal/managed"
	appver "sopsdeck/internal/version"
)

const mcpProtocolVersion = "2024-11-05"

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      any          `json:"id,omitempty"`
	Result  any          `json:"result,omitempty"`
	Error   *mcpRPCError `json:"error,omitempty"`
}

type mcpRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type mcpToolResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func cmdMCP(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	_ = args
	sc := bufio.NewScanner(stdin)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req mcpRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			writeMCPError(stdout, nil, -32700, "parse error")
			continue
		}
		if len(req.ID) == 0 || string(req.ID) == "null" {
			continue
		}
		id := parseMCPID(req.ID)
		resp := dispatchMCP(req, stderr, getenv)
		writeMCPResponse(stdout, id, resp)
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintf(stderr, "mcp: read stdin: %v\n", err)
		return 1
	}
	return 0
}

func parseMCPID(raw json.RawMessage) any {
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f
	}
	return raw
}

type mcpDispatchResult struct {
	Result any
	Error  *mcpRPCError
}

func dispatchMCP(req mcpRequest, stderr io.Writer, getenv func(string) string) mcpDispatchResult {
	switch req.Method {
	case "initialize":
		return mcpDispatchResult{Result: mcpInitializeResult()}
	case "ping":
		return mcpDispatchResult{Result: map[string]any{}}
	case "tools/list":
		return mcpDispatchResult{Result: map[string]any{"tools": mcpTools()}}
	case "tools/call":
		return dispatchMCPToolsCall(req.Params, stderr, getenv)
	default:
		return mcpDispatchResult{Error: &mcpRPCError{Code: -32601, Message: "method not found"}}
	}
}

func mcpInitializeResult() map[string]any {
	return map[string]any{
		"protocolVersion": mcpProtocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "sopsdeck",
			"version": appver.Version,
		},
	}
}

func mcpTools() []mcpTool {
	pathProp := map[string]any{"type": "string", "description": "Managed File or folder path"}
	return []mcpTool{
		{
			Name:        "list_managed_files",
			Description: "List Managed Files in a project folder (metadata only).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": pathProp,
				},
			},
		},
		{
			Name:        "list_keys",
			Description: "List secret key names in a Managed File (names only, not values).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": pathProp,
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "get_value",
			Description: "Read one secret value (requires host approval via SOPSDECK_MCP_APPROVE).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": pathProp,
					"key":  map[string]any{"type": "string", "description": "Secret key name"},
				},
				"required": []string{"path", "key"},
			},
		},
		{
			Name:        "run",
			Description: "Run a command with secrets injected into its environment (returns exit status only).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    pathProp,
					"command": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
				"required": []string{"path", "command"},
			},
		},
		{
			Name:        "list_recipients",
			Description: "List age recipient public keys for a Managed File (metadata only).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": pathProp,
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "git_status",
			Description: "Git porcelain status for a project folder.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": pathProp,
				},
			},
		},
	}
}

func dispatchMCPToolsCall(params json.RawMessage, stderr io.Writer, getenv func(string) string) mcpDispatchResult {
	var call struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return mcpDispatchResult{Error: &mcpRPCError{Code: -32602, Message: "invalid params"}}
	}
	if call.Arguments == nil {
		call.Arguments = map[string]any{}
	}
	result, ok := invokeMCPTool(call.Name, call.Arguments, stderr, getenv)
	if !ok {
		return mcpDispatchResult{Error: &mcpRPCError{Code: -32602, Message: "unknown tool"}}
	}
	return mcpDispatchResult{Result: result}
}

func invokeMCPTool(name string, args map[string]any, stderr io.Writer, getenv func(string) string) (mcpToolResult, bool) {
	switch name {
	case "list_managed_files":
		return mcpListManagedFiles(args, stderr)
	case "list_keys":
		return mcpListKeys(args, stderr)
	case "get_value":
		return mcpGetValue(args, stderr, getenv)
	case "run":
		return mcpRun(args, stderr)
	case "list_recipients":
		return mcpListRecipients(args, stderr)
	case "git_status":
		return mcpGitStatus(args, stderr)
	default:
		return mcpToolResult{}, false
	}
}

func mcpListManagedFiles(args map[string]any, stderr io.Writer) (mcpToolResult, bool) {
	root := mcpStringArg(args, "path", ".")
	files, err := managed.List(root)
	if err != nil {
		mcpLog(stderr, "list_managed_files", false)
		return mcpErrorResult(fmt.Sprintf("list_managed_files: %v", err)), true
	}
	text, err := json.Marshal(files)
	if err != nil {
		mcpLog(stderr, "list_managed_files", false)
		return mcpErrorResult("list_managed_files: encode error"), true
	}
	mcpLog(stderr, "list_managed_files", true)
	return mcpOKResult(string(text)), true
}

func mcpListKeys(args map[string]any, stderr io.Writer) (mcpToolResult, bool) {
	path := mcpStringArg(args, "path", "")
	if path == "" {
		mcpLog(stderr, "list_keys", false)
		return mcpErrorResult("list_keys: path required"), true
	}
	keys, err := managedFileKeys(path)
	if err != nil {
		mcpLog(stderr, "list_keys", false)
		return mcpErrorResult(fmt.Sprintf("list_keys: %v", err)), true
	}
	text, err := json.Marshal(keys)
	if err != nil {
		mcpLog(stderr, "list_keys", false)
		return mcpErrorResult("list_keys: encode error"), true
	}
	mcpLog(stderr, "list_keys", true)
	return mcpOKResult(string(text)), true
}

func mcpGetValue(args map[string]any, stderr io.Writer, getenv func(string) string) (mcpToolResult, bool) {
	if !mcpApproved(getenv, "get_value") {
		mcpLog(stderr, "get_value", false)
		return mcpErrorResult("get_value requires approval"), true
	}
	path := mcpStringArg(args, "path", "")
	key := mcpStringArg(args, "key", "")
	if path == "" || key == "" {
		mcpLog(stderr, "get_value", false)
		return mcpErrorResult("get_value: path and key required"), true
	}
	value, err := managedFileValue(path, key)
	if err != nil {
		mcpLog(stderr, "get_value", false)
		return mcpErrorResult(fmt.Sprintf("get_value: %v", err)), true
	}
	mcpLog(stderr, "get_value", true)
	return mcpOKResult(value), true
}

func mcpRun(args map[string]any, stderr io.Writer) (mcpToolResult, bool) {
	path := mcpStringArg(args, "path", "")
	cmd, err := mcpStringSliceArg(args, "command")
	if path == "" || err != nil || len(cmd) == 0 {
		mcpLog(stderr, "run", false)
		return mcpErrorResult("run: path and command required"), true
	}
	runArgs := append([]string{"-f", path, "--"}, cmd...)
	var childOut, childErr bytes.Buffer
	code := cmdRun(runArgs, strings.NewReader(""), &childOut, &childErr)
	status := "ok"
	if code != 0 {
		status = "error"
	}
	text := fmt.Sprintf(`{"exit":%d,"status":%q}`, code, status)
	mcpLog(stderr, "run", code == 0)
	if code != 0 {
		return mcpErrorResult(text), true
	}
	return mcpOKResult(text), true
}

func mcpListRecipients(args map[string]any, stderr io.Writer) (mcpToolResult, bool) {
	path := mcpStringArg(args, "path", "")
	if path == "" {
		mcpLog(stderr, "list_recipients", false)
		return mcpErrorResult("list_recipients: path required"), true
	}
	recipients, err := managedFileRecipients(path)
	if err != nil {
		mcpLog(stderr, "list_recipients", false)
		return mcpErrorResult(fmt.Sprintf("list_recipients: %v", err)), true
	}
	text, err := json.Marshal(recipients)
	if err != nil {
		mcpLog(stderr, "list_recipients", false)
		return mcpErrorResult("list_recipients: encode error"), true
	}
	mcpLog(stderr, "list_recipients", true)
	return mcpOKResult(string(text)), true
}

func mcpGitStatus(args map[string]any, stderr io.Writer) (mcpToolResult, bool) {
	root := mcpStringArg(args, "path", ".")
	out, err := gitPorcelain(root)
	if err != nil {
		mcpLog(stderr, "git_status", false)
		return mcpErrorResult(fmt.Sprintf("git_status: %v", err)), true
	}
	mcpLog(stderr, "git_status", true)
	return mcpOKResult(out), true
}

func managedFileKeys(path string) ([]string, error) {
	format := fileFormat(path)
	plain, err := decrypt.File(path, formatName(format))
	if err != nil {
		return nil, err
	}
	pairs, err := plainEnv(plain, format)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

func managedFileValue(path, key string) (string, error) {
	format := fileFormat(path)
	plain, err := decrypt.File(path, formatName(format))
	if err != nil {
		return "", err
	}
	value, ok, err := lookupValue(plain, format, key)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("missing key %s", key)
	}
	return value, nil
}

func managedFileRecipients(path string) ([]string, error) {
	store := common.StoreForFormat(fileFormat(path), config.NewStoresConfig())
	tree, err := common.LoadEncryptedFile(store, path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, group := range tree.Metadata.KeyGroups {
		for _, key := range group {
			out = append(out, key.ToString())
		}
	}
	sort.Strings(out)
	return out, nil
}

func gitPorcelain(dir string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func mcpApproved(getenv func(string) string, tool string) bool {
	if getenv == nil {
		return false
	}
	return strings.Contains(getenv("SOPSDECK_MCP_APPROVE"), tool)
}

func mcpStringArg(args map[string]any, key, fallback string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return fallback
	}
	s, ok := v.(string)
	if !ok {
		return fallback
	}
	return s
}

func mcpStringSliceArg(args map[string]any, key string) ([]string, error) {
	v, ok := args[key]
	if !ok {
		return nil, fmt.Errorf("missing %s", key)
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be array", key)
	}
	out := make([]string, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%s element must be string", key)
		}
		out[i] = s
	}
	return out, nil
}

func mcpOKResult(text string) mcpToolResult {
	return mcpToolResult{Content: []mcpContent{{Type: "text", Text: text}}}
}

func mcpErrorResult(text string) mcpToolResult {
	return mcpToolResult{Content: []mcpContent{{Type: "text", Text: text}}, IsError: true}
}

func mcpLog(stderr io.Writer, tool string, ok bool) {
	if ok {
		fmt.Fprintf(stderr, "mcp: %s ok\n", tool)
	} else {
		fmt.Fprintf(stderr, "mcp: %s denied\n", tool)
	}
}

func writeMCPResponse(w io.Writer, id any, resp mcpDispatchResult) {
	out := mcpResponse{JSONRPC: "2.0", ID: id}
	if resp.Error != nil {
		out.Error = resp.Error
	} else {
		out.Result = resp.Result
	}
	enc := json.NewEncoder(w)
	_ = enc.Encode(out)
}

func writeMCPError(w io.Writer, id any, code int, message string) {
	writeMCPResponse(w, id, mcpDispatchResult{Error: &mcpRPCError{Code: code, Message: message}})
}
