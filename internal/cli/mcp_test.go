package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

type jsonrpcReq struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

type jsonrpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcErr     `json:"error,omitempty"`
}

type jsonrpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolsListResult struct {
	Tools []struct {
		Name string `json:"name"`
	} `json:"tools"`
}

type toolsCallResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError"`
}

func mcpRoundTrip(t *testing.T, stdin io.Reader, stdout, stderr *bytes.Buffer, getenv func(string) string, req jsonrpcReq) jsonrpcResp {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	in := io.MultiReader(stdin, strings.NewReader(string(body)+"\n"))
	code := cmdMCP(nil, in, stdout, stderr, getenv)
	if code != 0 {
		t.Fatalf("cmdMCP exit %d stderr=%q", code, stderr.String())
	}
	return readJSONRPCResp(t, stdout)
}

func readJSONRPCResp(t *testing.T, stdout *bytes.Buffer) jsonrpcResp {
	t.Helper()
	sc := bufio.NewScanner(stdout)
	if !sc.Scan() {
		t.Fatal("no JSON-RPC response on stdout")
	}
	var resp jsonrpcResp
	if err := json.Unmarshal(sc.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v line=%q", err, sc.Text())
	}
	return resp
}

func mcpToolsCall(t *testing.T, stdout, stderr *bytes.Buffer, getenv func(string) string, name string, args map[string]any) toolsCallResult {
	t.Helper()
	resp := mcpRoundTrip(t, strings.NewReader(""), stdout, stderr, getenv, jsonrpcReq{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      name,
			"arguments": args,
		},
	})
	if resp.Error != nil {
		t.Fatalf("tools/call error: %s", resp.Error.Message)
	}
	var result toolsCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("parse tools/call result: %v", err)
	}
	return result
}

func toolsCallText(result toolsCallResult) string {
	var b strings.Builder
	for _, c := range result.Content {
		b.WriteString(c.Text)
	}
	return b.String()
}

func TestMCPToolsListExposesMetadataTools(t *testing.T) {
	var stdout, stderr bytes.Buffer
	getenv := func(string) string { return "" }

	initResp := mcpRoundTrip(t, strings.NewReader(""), &stdout, &stderr, getenv, jsonrpcReq{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "0"},
		},
	})
	if initResp.Error != nil {
		t.Fatalf("initialize error: %s", initResp.Error.Message)
	}

	stdout.Reset()
	listResp := mcpRoundTrip(t, strings.NewReader(""), &stdout, &stderr, getenv, jsonrpcReq{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	})
	if listResp.Error != nil {
		t.Fatalf("tools/list error: %s", listResp.Error.Message)
	}
	var listed toolsListResult
	if err := json.Unmarshal(listResp.Result, &listed); err != nil {
		t.Fatalf("parse tools/list: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range listed.Tools {
		names[tool.Name] = true
		if strings.Contains(tool.Name, "cat") || strings.Contains(tool.Name, "read_file") {
			t.Fatalf("must not expose plaintext dump tool %q", tool.Name)
		}
	}
	for _, want := range []string{"list_managed_files", "list_keys"} {
		if !names[want] {
			t.Fatalf("tools/list missing %q; got %v", want, names)
		}
	}
}

func TestMCPListKeysReturnsNamesNotValues(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	envFile := testdata(t, "hello.env")

	var stdout, stderr bytes.Buffer
	result := mcpToolsCall(t, &stdout, &stderr, os.Getenv, "list_keys", map[string]any{
		"path": envFile,
	})
	text := toolsCallText(result)
	if result.IsError {
		t.Fatalf("list_keys isError stderr=%q text=%q", stderr.String(), text)
	}
	if !strings.Contains(text, "HELLO") {
		t.Fatalf("result=%q want key HELLO", text)
	}
	if strings.Contains(text, "world") {
		t.Fatalf("result must not contain value world: %q", text)
	}
}

func TestMCPGetValueRequiresApproval(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	envFile := testdata(t, "hello.env")

	var stdout, stderr bytes.Buffer
	getenv := func(key string) string {
		if key == "SOPSDECK_MCP_APPROVE" {
			return ""
		}
		return os.Getenv(key)
	}
	result := mcpToolsCall(t, &stdout, &stderr, getenv, "get_value", map[string]any{
		"path": envFile,
		"key":  "HELLO",
	})
	text := toolsCallText(result)
	if !result.IsError {
		t.Fatalf("get_value without approval should error; text=%q", text)
	}
	if strings.Contains(text, "world") {
		t.Fatalf("result must not contain value: %q", text)
	}
	if !strings.Contains(stderr.String(), "get_value") {
		t.Fatalf("stderr should log tool name: %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "world") {
		t.Fatalf("stderr must not log value: %q", stderr.String())
	}
}

func TestMCPGetValueWithApprovalReturnsPlaintext(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	envFile := testdata(t, "hello.env")

	var stdout, stderr bytes.Buffer
	getenv := func(key string) string {
		if key == "SOPSDECK_MCP_APPROVE" {
			return "get_value"
		}
		return os.Getenv(key)
	}
	result := mcpToolsCall(t, &stdout, &stderr, getenv, "get_value", map[string]any{
		"path": envFile,
		"key":  "HELLO",
	})
	text := toolsCallText(result)
	if result.IsError {
		t.Fatalf("get_value denied stderr=%q text=%q", stderr.String(), text)
	}
	if !strings.Contains(text, "world") {
		t.Fatalf("result=%q want value world", text)
	}
	if strings.Contains(stderr.String(), "world") {
		t.Fatalf("stderr must not contain value: %q", stderr.String())
	}
}

func TestMCPRunReturnsOutcomeNotChildOutput(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY_FILE", testdata(t, "age.txt"))
	envFile := testdata(t, "hello.env")

	var stdout, stderr bytes.Buffer
	result := mcpToolsCall(t, &stdout, &stderr, os.Getenv, "run", map[string]any{
		"path":    envFile,
		"command": []any{"printenv", "HELLO"},
	})
	text := toolsCallText(result)
	if result.IsError {
		t.Fatalf("run failed stderr=%q text=%q", stderr.String(), text)
	}
	if strings.Contains(text, "world") {
		t.Fatalf("MCP result must not contain child stdout: %q", text)
	}
	if !strings.Contains(text, "0") && !strings.Contains(strings.ToLower(text), "ok") {
		t.Fatalf("result should report outcome/exit: %q", text)
	}
}
