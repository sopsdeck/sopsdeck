# Sopsdeck local MCP

Use the **local** `sopsdeck mcp` server for agent access to secrets. No remote MCP, no cloud processing.

## Setup

Run from a project root:

```bash
sopsdeck mcp
```

Wire it as a stdio MCP server in your editor. It speaks JSON-RPC 2.0 (newline-delimited JSON on stdin/stdout).

## Rules

- **Metadata by default** — use `list_managed_files`, `list_keys`, `list_recipients`, and `git_status` to explore without reading values.
- **Never dump Managed File plaintext into the prompt** — there is no `read_file` or `cat` tool by design.
- **Prefer `run`** — inject secrets into a child command (`run` with `path` + `command`) instead of pulling values into the model context.
- **`get_value` needs host approval** — the MCP host must set `SOPSDECK_MCP_APPROVE=get_value` for that call. Without it, the tool is denied.
- **No mutation tools** without explicit product approval paths; use the CLI directly for `set`, `del`, and recipient changes.

## Logging

The server logs tool name and outcome to stderr (`mcp: <tool> ok|denied`). It never logs secret values.
